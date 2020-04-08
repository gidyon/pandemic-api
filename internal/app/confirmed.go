package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
	"time"
)

const confirmedCasesTable = "confirmed_cases"

type confirmedCasesAPI struct {
	sqlDB *gorm.DB
}

// RegisterConfirmedCasesAPI registers http router for the contact API
func RegisterConfirmedCasesAPI(router *httprouter.Router, sqlDB *gorm.DB) {
	// Validation
	var err error
	switch {
	case router == nil:
		err = errors.New("router must not be nil")
	case sqlDB == nil:
		err = errors.New("sqlDB must not be nil")
	}
	handleError(err)

	c := &confirmedCasesAPI{
		sqlDB: sqlDB,
	}

	// Auto migration
	err = c.sqlDB.AutoMigrate(&ConfirmedPatient{}).Error
	handleError(err)

	// Update endpoints
	router.POST("/api/cases/confirmed", c.AddConfirmedPatient)
	router.PATCH("/api/cases/confirmed/:caseId/attend", c.MarkAttended)
	router.GET("/api/cases/confirmed/:caseId", c.GetConfirmedPatient)
	router.GET("/api/cases/confirmed", c.ListConfirmedPatients)
}

// ConfirmedPatient is a pandemic confirmed patient
type ConfirmedPatient struct {
	CaseID       string     `json:"case_id,omitempty" gorm:"primary_key;type:varchar(50);not null"`
	NationalID   string     `json:"national_id,omitempty" gorm:"index:query_index;type:varchar(15);unique_index;not null"`
	FullName     string     `json:"full_name,omitempty" gorm:"type:varchar(50);not null"`
	Email        string     `json:"email,omitempty" gorm:"index:query_index;type:varchar(50);unique_index;not null"`
	Phone        string     `json:"phone,omitempty" gorm:"index:query_index;type:varchar(15);unique_index;not null"`
	County       string     `json:"county,omitempty" gorm:"type:varchar(50);not null"`
	SubCounty    string     `json:"sub_county,omitempty" gorm:"type:varchar(50);not null"`
	Constituency string     `json:"constituency,omitempty" gorm:"type:varchar(50);not null"`
	Ward         string     `json:"ward,omitempty" gorm:"type:varchar(50);not null"`
	Residence    string     `json:"residence,omitempty" gorm:"type:varchar(50);not null"`
	ReportedDate string     `json:"reported_date,omitempty" gorm:"type:varchar(50);not null"`
	Status       string     `json:"status,omitempty" gorm:"type:varchar(20);not null"`
	Facility     string     `json:"facility,omitempty" gorm:"type:varchar(50);not null"`
	Attended     bool       `json:"attended" gorm:"type:tinyint(1);default:0"`
	CreatedAt    time.Time  `json:"-"`
	DeletedAt    *time.Time `json:"-"`
}

// BeforeCreate is a hook that is set before creating object
func (*ConfirmedPatient) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("CaseID", uuid.New().String())
}

// TableName is the table name
func (*ConfirmedPatient) TableName() string {
	return confirmedCasesTable
}

func (patient *ConfirmedPatient) validate() error {
	var err error
	switch {
	case strings.TrimSpace(patient.FullName) == "":
		err = errors.New("missing patient full name")
	case strings.TrimSpace(patient.Email) == "" && strings.TrimSpace(patient.Phone) == "":
		err = errors.New("missing patient email and phone")
	case strings.TrimSpace(patient.County) == "":
		err = errors.New("missing patient county")
	case strings.TrimSpace(patient.SubCounty) == "":
		err = errors.New("missing patient sub-county")
	case strings.TrimSpace(patient.Constituency) == "":
		err = errors.New("missing patient constituency")
	case strings.TrimSpace(patient.Ward) == "":
		err = errors.New("missing patient ward")
	case strings.TrimSpace(patient.Residence) == "":
		err = errors.New("missing patient residence")
	case strings.TrimSpace(patient.Status) == "":
		err = errors.New("missing patient status")
	case strings.TrimSpace(patient.Facility) == "":
		err = errors.New("missing facility name")
	}

	return err
}

func (cap *confirmedCasesAPI) AddConfirmedPatient(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	patient := &ConfirmedPatient{}
	err := json.NewDecoder(r.Body).Decode(patient)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = patient.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	patient.Attended = false

	// Add to database
	err = cap.sqlDB.Create(patient).Error
	switch {
	case err == nil:
	case strings.Contains(strings.ToLower(err.Error()), "duplicate"):
		errStr := strings.ToLower(err.Error())
		var errMsg string
		switch {
		case strings.Contains(errStr, "national_id"):
			errMsg = fmt.Sprintf("Patient with national id %s has been uploaded", patient.NationalID)
		case strings.Contains(errStr, "email"):
			errMsg = fmt.Sprintf("Patient with email %s has been uploaded", patient.Email)
		case strings.Contains(errStr, "phone"):
			errMsg = fmt.Sprintf("Patient with phone %s has been uploaded", patient.Phone)

		}
		http.Error(w, errMsg, http.StatusForbidden)
		return
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(map[string]string{"case_id": patient.CaseID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (cap *confirmedCasesAPI) MarkAttended(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get patient id
	caseID := p.ByName("caseId")
	if strings.TrimSpace(caseID) == "" {
		http.Error(w, "missing case id", http.StatusBadRequest)
		return
	}

	// Update database
	err := cap.sqlDB.Table(confirmedCasesTable).Unscoped().Where("case_id=?", caseID).
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

func (cap *confirmedCasesAPI) ListConfirmedPatients(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
	}(cap.sqlDB)

	// Execute query
	patients := make([]*ConfirmedPatient, 0, ps)
	err = db.Find(&patients).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Write response back
	err = json.NewEncoder(w).Encode(patients)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (cap *confirmedCasesAPI) GetConfirmedPatient(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get patient id
	caseID := p.ByName("caseId")
	if strings.TrimSpace(caseID) == "" {
		http.Error(w, "missing case id", http.StatusBadRequest)
		return
	}
	patient := &ConfirmedPatient{}

	// Get from database
	err := cap.sqlDB.First(patient, "case_id=?", caseID).Error
	switch {
	case err == nil:
	case gorm.IsRecordNotFoundError(err):
		http.Error(w, "patient not found", http.StatusBadRequest)
		return
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Write to connection
	err = json.NewEncoder(w).Encode(patient)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}
