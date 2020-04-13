package app

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const reportedCasesTable = "reported_cases"

type reportAPI struct {
	sqlDB *gorm.DB
}

// RegisterReportedCasesAPI registers http router for the report API
func RegisterReportedCasesAPI(router *httprouter.Router, sqlDB *gorm.DB) {
	// Validation
	var err error
	switch {
	case sqlDB == nil:
		err = errors.New("sqlDB must not be nil")
	case router == nil:
		err = errors.New("router must not be nil")
	}
	handleError(err)

	reportAPI := &reportAPI{
		sqlDB: sqlDB,
	}

	// Auto migration
	err = reportAPI.sqlDB.AutoMigrate(&ReportedCase{}).Error
	handleError(err)

	router.GET("/api/cases/reported/:caseId", reportAPI.GetReport)
	router.GET("/api/cases/reported", reportAPI.ListReport)
	router.POST("/api/cases/reported", reportAPI.AddReport)
	router.PATCH("/api/cases/reported/:caseId/attend", reportAPI.MarkAttended)
}

// ReportedCase is payload data for reporting COVID-19 cases
type ReportedCase struct {
	CaseID                string     `json:"case_id,omitempty" gorm:"primary_key;type:varchar(50);not null"`
	ReporterFullName      string     `json:"reporter_full_name,omitempty" gorm:"type:varchar(50);not null"`
	ReporterEmail         string     `json:"reporter_email,omitempty" gorm:"index:query_index;type:varchar(50);not null"`
	ReporterPhone         string     `json:"reporter_phone,omitempty" gorm:"index:query_index;type:varchar(15);not null"`
	ReporterProfileThumb  string     `json:"reporter_profile_thumb,omitempty" gorm:"type:varchar(256)"`
	County                string     `json:"county,omitempty" gorm:"type:varchar(50);not null"`
	Constituency          string     `json:"constituency,omitempty" gorm:"type:varchar(50);not null"`
	Ward                  string     `json:"ward,omitempty" gorm:"type:varchar(30);not null"`
	Location              string     `json:"location,omitempty" gorm:"type:varchar(50);not null"`
	LocationLongitude     float32    `json:"location_longitude,omitempty" gorm:"type:float(10);not null"`
	LocationLatitude      float32    `json:"location_latitude,omitempty" gorm:"type:float(10);not null"`
	LocationVerified      bool       `json:"location_verified" gorm:"type:tinyint(1);default:0"`
	SuspectFullName       string     `gorm:"type:varchar(50);not null" json:"suspect_full_name,omitempty"`
	SuspectPhone          string     `gorm:"type:varchar(15);not null" json:"suspect_phone,omitempty"`
	SuspectEmail          string     `gorm:"type:varchar(50);not null" json:"suspect_email,omitempty"`
	SuspectCondition      string     `gorm:"type:varchar(30);not null" json:"suspect_condition,omitempty"`
	SuspectRelationship   string     `gorm:"type:varchar(30);not null" json:"suspect_relationship,omitempty"`
	AdditionalInformation string     `json:"additional_information,omitempty" gorm:"type:text;not null"`
	Attended              bool       `json:"attended,omitempty" gorm:"type:tinyint(1);default:0"`
	CreatedAt             time.Time  `json:"-"`
	DeletedAt             *time.Time `json:"-"`
}

// BeforeCreate is a hook that is set before creating object
func (*ReportedCase) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("CaseID", uuid.New().String())
}

// TableName is the table name
func (*ReportedCase) TableName() string {
	return reportedCasesTable
}

func (report *ReportedCase) validate() error {
	var err error
	switch {
	case strings.TrimSpace(report.ReporterFullName) == "":
		err = errors.New("missing reporter fullname")
	case strings.TrimSpace(report.ReporterEmail) == "" && strings.TrimSpace(report.ReporterPhone) == "":
		err = errors.New("missing reporter email and phone")
	case strings.TrimSpace(report.County) == "":
		err = errors.New("missing report county")
	case strings.TrimSpace(report.Constituency) == "":
		err = errors.New("missing report constituency")
	case strings.TrimSpace(report.Ward) == "":
		err = errors.New("missing report ward")
	case strings.TrimSpace(report.SuspectFullName) == "":
		err = errors.New("missing reportee fullname")
	case strings.TrimSpace(report.SuspectRelationship) == "":
		err = errors.New("missing reportee relationship")
	}
	return err
}

func (reportAPI *reportAPI) GetReport(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get report id
	caseID := p.ByName("caseId")
	if strings.TrimSpace(caseID) == "" {
		http.Error(w, "missing case id", http.StatusBadRequest)
		return
	}
	report := &ReportedCase{}

	// Get from database
	err := reportAPI.sqlDB.First(report, "case_id=?", caseID).Error
	switch {
	case err == nil:
	case gorm.IsRecordNotFoundError(err):
		http.Error(w, "reported case not found", http.StatusBadRequest)
		return
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Write to connection
	err = json.NewEncoder(w).Encode(report)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (reportAPI *reportAPI) ListReport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	// Get filters
	counties := splitQuery(query.Get("counties"), ",")
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
		if len(constituencies) != 0 {
			db = db.Where("constituencies IN (?)", constituencies)
		}
		if len(wards) != 0 {
			db = db.Where("wards IN (?)", wards)
		}
		return db.Limit(ps).Offset(ps*pn - ps)
	}(reportAPI.sqlDB)

	// Execute query
	reports := make([]*ReportedCase, 0, ps)
	err = db.Find(&reports).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Write response back
	err = json.NewEncoder(w).Encode(reports)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

const defaultPagesize = 50

func getPaginationData(pageSize, pageNumber string) (int, int, error) {
	if pageSize == "" {
		pageSize = "0"
	}
	if pageNumber == "" {
		pageNumber = "0"
	}
	ps, err := strconv.Atoi(pageSize)
	if err != nil {
		return 0, 0, err
	}
	if ps <= 0 {
		ps = defaultPagesize
	}
	pn, err := strconv.Atoi(pageNumber)
	if err != nil {
		return 0, 0, err
	}
	if pn <= 0 {
		pn = 1
	}
	return ps, pn, nil
}

func (reportAPI *reportAPI) AddReport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	report := &ReportedCase{}
	err := json.NewDecoder(r.Body).Decode(report)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = report.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	report.Attended = false
	report.LocationVerified = false

	// Update to database
	err = reportAPI.sqlDB.Create(report).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write success message response
	err = json.NewEncoder(w).Encode(map[string]string{"case_id": report.CaseID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (reportAPI *reportAPI) MarkAttended(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get report id
	caseID := p.ByName("caseId")
	if strings.TrimSpace(caseID) == "" {
		http.Error(w, "missing case id", http.StatusBadRequest)
		return
	}

	// Update database
	err := reportAPI.sqlDB.Table(reportedCasesTable).Unscoped().Where("case_id=?", caseID).
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
