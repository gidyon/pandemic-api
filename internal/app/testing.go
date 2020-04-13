package app

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
	"time"
)

const questionnaireCaseTable = "questionnaire_cases"

type questionnaireCase struct {
	sqlDB *gorm.DB
}

// RegisterQuestionnaireCasesAPI registers http router for the suspect API
func RegisterQuestionnaireCasesAPI(router *httprouter.Router, sqlDB *gorm.DB) {
	// Validation
	var err error
	switch {
	case sqlDB == nil:
		err = errors.New("sqlDB must not be nil")
	case router == nil:
		err = errors.New("router must not be nil")
	}
	handleError(err)

	questionnaireCase := &questionnaireCase{
		sqlDB: sqlDB,
	}

	// Auto migration
	err = questionnaireCase.sqlDB.AutoMigrate(&QuestionnaireCase{}).Error
	handleError(err)

	router.GET("/api/cases/questionnaire/:caseId", questionnaireCase.GetQuestionnaireCase)
	router.GET("/api/cases/questionnaire", questionnaireCase.ListQuestionnaireCases)
	router.POST("/api/cases/questionnaire", questionnaireCase.AddQuestionnaireCase)
	router.PATCH("/api/cases/questionnaire/:caseId/attend", questionnaireCase.MarkCaseAttended)
}

// QuestionnaireCase represent a COVID19 questionnaire case
type QuestionnaireCase struct {
	CaseID              string      `json:"case_id,omitempty" gorm:"primary_key;type:varchar(50);not null"`
	SuspectFullName     string      `json:"suspect_full_name,omitempty" gorm:"type:varchar(50);not null"`
	SuspectEmail        string      `json:"suspect_email,omitempty"  gorm:"type:varchar(50);not null"`
	SuspectPhone        string      `json:"suspect_phone,omitempty" gorm:"index:query_index;type:varchar(15);not null"`
	SuspectProfileThumb string      `json:"suspect_profile_thumb,omitempty" gorm:"type:varchar(256)"`
	SuspectAgeGroup     string      `json:"suspect_age_group,omitempty" gorm:"type:varchar(20);not null"`
	Location            string      `json:"location,omitempty" gorm:"type:varchar(50);not null"`
	LocationLongitude   float32     `json:"location_longitude,omitempty" gorm:"type:float(10);not null"`
	LocationLatitude    float32     `json:"location_latitude,omitempty" gorm:"type:float(10);not null"`
	LocationVerified    bool        `json:"location_verified" gorm:"type:tinyint(1);default:0"`
	Trace               bool        `json:"trace" gorm:"type:tinyint(1);default:1"`
	Attended            bool        `json:"attended" gorm:"type:tinyint(1);default:0"`
	TestResults         interface{} `json:"test_results,omitempty" gorm:"-"`
	Results             []byte      `json:"-" gorm:"type:json;not null"`
	CreatedAt           time.Time   `json:"-"`
	DeletedAt           *time.Time  `json:"-"`
}

// BeforeCreate is a hook that is set before creating object
func (*QuestionnaireCase) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("CaseID", uuid.New().String())
}

// TableName is the table name
func (*QuestionnaireCase) TableName() string {
	return questionnaireCaseTable
}

func (suspect *QuestionnaireCase) validate() error {
	var err error
	switch {
	case strings.TrimSpace(suspect.SuspectFullName) == "":
		err = errors.New("missing suspect fullname")
	case strings.TrimSpace(suspect.SuspectEmail) == "" && strings.TrimSpace(suspect.SuspectPhone) == "":
		err = errors.New("missing suspect email and phone")
	case strings.TrimSpace(suspect.SuspectAgeGroup) == "":
		err = errors.New("missing suspect age group")
	case strings.TrimSpace(suspect.Location) == "":
		err = errors.New("missing suspect location")
	case suspect.LocationLatitude == 0:
		err = errors.New("missing suspect latitude location")
	case suspect.LocationLongitude == 0:
		err = errors.New("missing suspect longitude location")
	case suspect.TestResults == nil:
		err = errors.New("missing suspect test results")
	}
	return err
}

func (sap *questionnaireCase) AddQuestionnaireCase(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	suspect := &QuestionnaireCase{}
	err := json.NewDecoder(r.Body).Decode(suspect)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = suspect.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	suspect.Attended = false

	results, err := json.Marshal(suspect.TestResults)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	suspect.Results = results

	// Update to database
	err = sap.sqlDB.Create(suspect).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write success message response
	err = json.NewEncoder(w).Encode(map[string]string{"case_id": suspect.CaseID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (sap *questionnaireCase) MarkCaseAttended(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get suspect id
	caseID := p.ByName("caseId")
	if strings.TrimSpace(caseID) == "" {
		http.Error(w, "missing case id", http.StatusBadRequest)
		return
	}

	// Update database
	err := sap.sqlDB.Table(questionnaireCaseTable).Unscoped().Where("case_id=?", caseID).
		Update("attended", true).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write success message response
	_, err = w.Write([]byte("attended"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (sap *questionnaireCase) ListQuestionnaireCases(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	// Get filters
	counties := splitQuery(query.Get("counties"), ",")
	subCounties := splitQuery(query.Get("sub_counties"), ",")
	constituencies := splitQuery(query.Get("constituencies"), ",")
	wards := splitQuery(query.Get("wards"), ",")

	// Pagination
	ps, pn, err := getPaginationData(query.Get("page_size"), query.Get("page_number"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Prepare query
	db := func(db *gorm.DB) *gorm.DB {
		if len(counties) != 0 {
			db = db.Where("county IN (?)", counties)
		}
		if len(subCounties) != 0 {
			db = db.Where("sub_county IN (?)", subCounties)
		}
		if len(constituencies) != 0 {
			db = db.Where("constituencies IN (?)", constituencies)
		}
		if len(wards) != 0 {
			db = db.Where("wards IN (?)", wards)
		}
		return db.Limit(ps).Offset(ps*pn - ps)
	}(sap.sqlDB)

	// Execute query
	suspects := make([]*QuestionnaireCase, 0, ps)
	err = db.Find(&suspects).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Write response back
	err = json.NewEncoder(w).Encode(suspects)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (sap *questionnaireCase) GetQuestionnaireCase(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get suspect id
	caseID := p.ByName("caseId")
	if strings.TrimSpace(caseID) == "" {
		http.Error(w, "missing case id", http.StatusBadRequest)
		return
	}
	suspect := &QuestionnaireCase{}

	// Get from database
	err := sap.sqlDB.First(suspect, "case_id=?", caseID).Error
	switch {
	case err == nil:
	case gorm.IsRecordNotFoundError(err):
		http.Error(w, "questionnaire case not found", http.StatusBadRequest)
		return
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = json.Unmarshal(suspect.Results, &suspect.TestResults)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Write to connection
	err = json.NewEncoder(w).Encode(suspect)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
