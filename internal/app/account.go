package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gidyon/fightcovid19/internal/auth"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"time"
)

const accountsTable = "accounts"

type accountAPI struct {
	sqlDB *gorm.DB
}

func handleError(err error) {
	if err != nil {
		logrus.Fatalln(err)
	}
}

// RegisterAccountAPI registers http router for the acc API
func RegisterAccountAPI(router *httprouter.Router, sqlDB *gorm.DB) {
	// Validation
	var err error
	switch {
	case sqlDB == nil:
		err = errors.New("sqlDB must not be nil")
	case router == nil:
		err = errors.New("router must not be nil")
	}
	handleError(err)

	acc := &accountAPI{
		sqlDB: sqlDB,
	}

	acc.sqlDB.Debug()

	// Auto migrate
	err = acc.sqlDB.AutoMigrate(&Account{}).Error
	handleError(err)

	router.GET("/api/accounts", acc.ListAccounts)
	router.GET("/api/accounts/:accountId", acc.GetAccount)
	router.POST("/api/accounts", acc.CreateAccount)
	router.POST("/api/accounts/login", acc.Login)
	router.PUT("/api/accounts/:accountId", acc.UpdateAccount)
	router.PATCH("/api/accounts/:accountId/change_password", acc.ChangePassword)
	router.PATCH("/api/accounts/:accountId/verify", acc.VerifyAccount)
}

// Account is an entity performing actions
type Account struct {
	AccountID       string     `json:"account_id,omitempty" gorm:"primary_key;type:varchar(50);not null"`
	NationalID      string     `json:"national_id,omitempty" gorm:"index:query_index;type:varchar(15);unique_index;not null"`
	FullName        string     `json:"full_name,omitempty" gorm:"type:varchar(50);not null"`
	Email           string     `json:"email,omitempty" gorm:"index:query_index;type:varchar(50);unique_index;not null"`
	Phone           string     `json:"phone,omitempty" gorm:"index:query_index;type:varchar(15);unique_index;not null"`
	County          string     `json:"county,omitempty" gorm:"type:varchar(50);not null"`
	SubCounty       string     `json:"sub_county,omitempty" gorm:"type:varchar(50);not null"`
	Constituency    string     `json:"constituency,omitempty" gorm:"type:varchar(50);not null"`
	Ward            string     `json:"ward,omitempty" gorm:"type:varchar(50);not null"`
	Residence       string     `json:"residence,omitempty" gorm:"type:varchar(50);not null"`
	Profession      string     `json:"profession,omitempty" gorm:"type:varchar(50);not null"`
	InstitutionName string     `json:"institution_name,omitempty" gorm:"type:varchar(50);not null"`
	Gender          string     `json:"gender,omitempty" gorm:"type:varchar(10);default:'unknown'"`
	Verified        bool       `json:"verified" gorm:"type:tinyint(1);default:0"`
	Group           string     `json:"group,omitempty" gorm:"type:varchar(50);not null"`
	Password        string     `json:"-" gorm:"type:text"`
	CreatedAt       time.Time  `json:"-"`
	DeletedAt       *time.Time `json:"-"`
}

// BeforeCreate is a hook that is set before creating object
func (*Account) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("AccountID", uuid.New().String())
}

// TableName is the table name
func (*Account) TableName() string {
	return accountsTable
}

func (accountAPI *Account) validate() error {
	var err error

	switch {
	case strings.TrimSpace(accountAPI.NationalID) == "":
		err = errors.New("national is required")
	case strings.TrimSpace(accountAPI.FullName) == "":
		err = errors.New("fullname is required")
	case strings.TrimSpace(accountAPI.Email) == "" && strings.TrimSpace(accountAPI.Phone) == "":
		err = errors.New("email or phone is required")
	case strings.TrimSpace(accountAPI.County) == "":
		err = errors.New("county is required")
	case strings.TrimSpace(accountAPI.SubCounty) == "":
		err = errors.New("sub county is required")
	case strings.TrimSpace(accountAPI.Constituency) == "":
		err = errors.New("constituency is required")
	case strings.TrimSpace(accountAPI.Ward) == "":
		err = errors.New("ward is required")
	case strings.TrimSpace(accountAPI.Residence) == "":
		err = errors.New("residence is required")
	}

	return err
}

// generates hashed version of password
func genHash(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

// compares hashed password with password
func compareHash(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func (accountAPI *accountAPI) CreateAccount(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	acc := &Account{}
	err := json.NewDecoder(r.Body).Decode(acc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = acc.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update account
	acc.Verified = false

	// Hash password
	acc.Password, err = genHash(acc.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create in database
	err = accountAPI.sqlDB.Create(acc).Error
	switch {
	case err == nil:
	case strings.Contains(strings.ToLower(err.Error()), "duplicate"):
		errStr := strings.ToLower(err.Error())
		var errMsg string
		switch {
		case strings.Contains(errStr, "national_id"):
			errMsg = fmt.Sprintf("Account with national id %s exists", acc.Email)
		case strings.Contains(errStr, "email"):
			errMsg = fmt.Sprintf("Account with email %s exists", acc.Email)
		case strings.Contains(errStr, "phone"):
			errMsg = fmt.Sprintf("Account with phone %s exists", acc.Phone)

		}
		http.Error(w, errMsg, http.StatusForbidden)
		return
	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write acc id to response
	err = json.NewEncoder(w).Encode(map[string]string{"account_id": acc.AccountID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
}

func (accountAPI *accountAPI) GetAccount(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get acc id
	accountID := p.ByName("accountId")
	if strings.TrimSpace(accountID) == "" {
		http.Error(w, "missing account id", http.StatusBadRequest)
		return
	}
	acc := &Account{}

	// Get from database
	err := accountAPI.sqlDB.First(acc, "account_id=?", accountID).Error
	switch {
	case err == nil:
	case gorm.IsRecordNotFoundError(err):
		http.Error(w, "account not found", http.StatusBadRequest)
		return
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Write to connection
	err = json.NewEncoder(w).Encode(acc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

type loginRequest struct {
	UserName string
	Password string
}

type loginResponse struct {
	AccountID string `json:"account_id,omitempty"`
	Group     string `json:"group,omitempty"`
	Verified  bool   `json:"verified"`
	Token     string `json:"token,omitempty"`
}

func (accountAPI *accountAPI) Login(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	login := &loginRequest{}
	err := json.NewDecoder(r.Body).Decode(login)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	switch {
	case strings.TrimSpace(login.UserName) == "":
		err = errors.New("missing username")
	case strings.TrimSpace(login.Password) == "":
		err = errors.New("missing password")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the acc
	acc := &Account{}
	err = accountAPI.sqlDB.First(acc, "email=? OR phone=?", login.UserName, login.UserName).Error
	switch {
	case err == nil:
	case gorm.IsRecordNotFoundError(err):
		http.Error(w, "account does not exist", http.StatusBadRequest)
		return
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Compare password
	err = compareHash(acc.Password, login.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate token
	tokenStr, err := auth.GenToken(r.Context(), &auth.Payload{
		ID:           acc.AccountID,
		FullName:     acc.FullName,
		PhoneNumber:  acc.Password,
		EmailAddress: acc.Email,
		Group:        acc.Group,
	}, acc.Group, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Marshal response
	err = json.NewEncoder(w).Encode(&loginResponse{
		AccountID: acc.AccountID,
		Token:     tokenStr,
		Group:     acc.Group,
		Verified:  acc.Verified,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (accountAPI *accountAPI) UpdateAccount(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	accountID := p.ByName("accountId")
	if strings.TrimSpace(accountID) == "" {
		http.Error(w, "Missing account id", http.StatusBadRequest)
		return
	}

	// Marshaling
	acc := &Account{}
	err := json.NewDecoder(r.Body).Decode(acc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Reset acc if they updated phone or email
	if acc.Email != "" || acc.Phone != "" || acc.NationalID == "" {
		acc.Verified = false
	}

	// Update in database
	err = accountAPI.sqlDB.Table(accountsTable).Unscoped().Omit("account_id, verified, group, password").
		Where("account_id=?", accountID).Updates(acc).Error
	switch {
	case err == nil:
	case strings.Contains(strings.ToLower(err.Error()), "duplicate"):
		errStr := strings.ToLower(err.Error())
		var errMsg string
		switch {
		case strings.Contains(errStr, "national_id"):
			errMsg = fmt.Sprintf("Account with national id %s exists", acc.Email)
		case strings.Contains(errStr, "email"):
			errMsg = fmt.Sprintf("Account with email %s exists", acc.Email)
		case strings.Contains(errStr, "phone"):
			errMsg = fmt.Sprintf("Account with phone %s exists", acc.Phone)

		}
		http.Error(w, errMsg, http.StatusForbidden)
		return
	default:
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// Write acc id to response
	err = json.NewEncoder(w).Encode(map[string]string{"account_id": accountID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
}

func splitQuery(str, sep string) []string {
	if str == "" {
		return []string{}
	}
	strs := strings.Split(str, sep)
	if strs[0] == str && len(str) == 1 {
		return []string{}
	}
	return strs
}

func (accountAPI *accountAPI) ListAccounts(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	// Get filters
	counties := splitQuery(query.Get("counties"), ",")
	subCounties := splitQuery(query.Get("sub_counties"), ",")
	constituencies := splitQuery(query.Get("constituencies"), ",")
	wards := splitQuery(query.Get("wards"), ",")
	groups := splitQuery(query.Get("groups"), ",")

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
		if len(groups) != 0 {
			db = db.Where("group IN (?)", groups)
		}
		// Verified
		if status := query.Get("verified"); status == "true" {
			db = db.Where("verified = ?", true)
		} else if status == "false" {
			db = db.Where("verified = ?", false)
		}
		return db.Limit(ps).Offset(ps*pn - ps)
	}(accountAPI.sqlDB)

	// Execute query
	accounts := make([]*Account, 0, ps)
	err = db.Find(&accounts).Error
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Write response back
	err = json.NewEncoder(w).Encode(accounts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

type changePassword struct {
	NewPassword     string `json:"new_password,omitempty"`
	ConfirmPassword string `json:"confirm_password,omitempty"`
}

func (accountAPI *accountAPI) ChangePassword(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	accountID := p.ByName("accountId")
	if strings.TrimSpace(accountID) == "" {
		http.Error(w, "Missing account id", http.StatusBadRequest)
		return
	}

	// Marshaling
	changePass := &changePassword{}
	err := json.NewDecoder(r.Body).Decode(changePass)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// Validation
	switch {
	case strings.TrimSpace(changePass.NewPassword) == "":
		err = errors.New("missing password")
	case strings.TrimSpace(changePass.ConfirmPassword) == "":
		err = errors.New("missing confirm password")
	case changePass.NewPassword != changePass.ConfirmPassword:
		err = errors.New("password do not match")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// Hash password
	changePass.ConfirmPassword, err = genHash(changePass.ConfirmPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update model
	db := accountAPI.sqlDB.Table(accountsTable).Unscoped().Where("account_id=?", accountID).
		Update("password", changePass.ConfirmPassword)
	switch {
	case db.RowsAffected == 0:
		http.Error(w, "Account does not exist", http.StatusForbidden)
		return
	case db.Error == nil:
	case db.Error != nil:
		http.Error(w, db.Error.Error(), http.StatusForbidden)
		return
	}

	// Write some response
	w.Write([]byte("success"))
}

func (accountAPI *accountAPI) VerifyAccount(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	accountID := p.ByName("accountId")
	if strings.TrimSpace(accountID) == "" {
		http.Error(w, "Missing account id", http.StatusBadRequest)
		return
	}

	// Update model
	db := accountAPI.sqlDB.Table(accountsTable).Unscoped().Where("account_id=?", accountID).
		Update("verified", true)
	switch {
	case db.Error == nil:
	case db.Error != nil:
		http.Error(w, db.Error.Error(), http.StatusForbidden)
		return
	}

	// Write some response
	w.Write([]byte("verified"))
}
