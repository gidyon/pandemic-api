package app

import (
	"encoding/json"
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const reportsTable = "covid19_cases_report"

type reportAPI struct {
	sqlDB *gorm.DB
}

// RegisterReportAPIRouter registers http router for the report API
func RegisterReportAPIRouter(router *httprouter.Router, sqlDB *gorm.DB) {
	// Validation
	var err error
	switch {
	case sqlDB == nil:
		err = errors.New("sqlDB must not be nil")
	case router == nil:
		err = errors.New("router must not be nil")
	}
	if err != nil {
		logrus.Fatalln(err)
	}

	report := &reportAPI{
		sqlDB: sqlDB,
	}

	router.GET("/case/:reportID", report.GetReport)
	router.GET("/case", report.GetReport)
	router.POST("/case", report.AddReport)
}

// COVID19CaseReport is payload data for reporting COVID-19 cases
type COVID19CaseReport struct {
	ReportID              string
	ReporteeID            string
	ReporterFullName      string
	ReporterEmail         string
	ReporterPhone         string
	ReporterProfileThumb  string
	County                string
	SubCounty             string
	Constituency          string
	Ward                  string
	ReporteeFullName      string
	ReporteePhone         string
	ReporteeCondition     string
	ReporteeRelationship  string
	AdditionalInformation string
	Attended              bool
	CreatedAt             time.Time
	DeletedAt             *time.Time
}

// BeforeCreate is a hook that is set before creating object
func (*COVID19CaseReport) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ReportID", uuid.New().String())
}

// TableName is the table name
func (*COVID19CaseReport) TableName() string {
	return reportsTable
}

func (report *COVID19CaseReport) validate() error {
	var err error
	switch {
	case strings.TrimSpace(report.ReporterFullName) == "":
		err = errors.New("missing reporter fullname")
	case strings.TrimSpace(report.ReporterEmail) == "" && strings.TrimSpace(report.ReporterPhone) == "":
		err = errors.New("missing reporter email and phone")
	case strings.TrimSpace(report.County) == "":
		err = errors.New("missing report county")
	case strings.TrimSpace(report.SubCounty) == "":
		err = errors.New("missing report sub-county")
	case strings.TrimSpace(report.Constituency) == "":
		err = errors.New("missing report constituency")
	case strings.TrimSpace(report.Ward) == "":
		err = errors.New("missing report ward")
	case strings.TrimSpace(report.ReporteeFullName) == "":
		err = errors.New("missing reportee fullname")
	case strings.TrimSpace(report.ReporteeRelationship) == "":
		err = errors.New("missing reportee relationship")
	}
	return err
}

func (reportAPI *reportAPI) GetReport(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get report id
	reportID := p.ByName("reportID")
	if strings.TrimSpace(reportID) == "" {
		http.Error(w, "missing report id", http.StatusBadRequest)
		return
	}
	report := &COVID19CaseReport{}

	// Get from database
	err := reportAPI.sqlDB.First(report, "report_id=?", reportID).Error
	switch {
	case err == nil:
	case gorm.IsRecordNotFoundError(err):
		http.Error(w, "report not found", http.StatusBadRequest)
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
	counties := strings.Split(query.Get("counties"), ",")
	subCounties := strings.Split(query.Get("sub_counties"), ",")
	constituencies := strings.Split(query.Get("constituencies"), ",")
	wards := strings.Split(query.Get("wards"), ",")

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
	}(reportAPI.sqlDB)
	defer db.Close()

	// Execute query
	reports := make([]*COVID19CaseReport, 0)
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
	report := &COVID19CaseReport{}
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

	// Update to database
	err = reportAPI.sqlDB.Create(report).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (reportAPI *reportAPI) MarkAttended(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get report id
	reportID := p.ByName("reportID")
	if strings.TrimSpace(reportID) == "" {
		http.Error(w, "missing report id", http.StatusBadRequest)
		return
	}

	// Update database
	err := reportAPI.sqlDB.Table(reportsTable).Unscoped().Where("report_id=?", reportID).
		Update("attended", true).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write success message response
	_, err = w.Write([]byte("success"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
