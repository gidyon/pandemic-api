package rest

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gidyon/pandemic-api/internal/services"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/gidyon/pandemic-api/internal/auth"
	http_error "github.com/gidyon/pandemic-api/pkg/errors"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"
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

	router.GET("/rest/v1/accounts", acc.ListAccounts)
	router.GET("/rest/v1/accounts/:accountId", acc.GetAccount)
	router.POST("/rest/v1/accounts/file", acc.AddUsersFromFile)
	router.POST("/rest/v1/accounts", acc.CreateAccount)
	router.POST("/rest/v1/accounts/login", acc.Login)
	router.PUT("/rest/v1/accounts/:accountId", acc.UpdateAccount)
	router.PATCH("/rest/v1/accounts/:accountId/change_password", acc.ChangePassword)
	router.PATCH("/rest/v1/accounts/:accountId/verify", acc.VerifyAccount)
}

// Account is an entity performing actions
type Account struct {
	AccountID  string     `json:"account_id,omitempty" gorm:"primary_key;type:varchar(50);not null"`
	NationalID string     `json:"national_id,omitempty" gorm:"index:query_index;type:varchar(15);unique_index;not null"`
	FullName   string     `json:"full_name,omitempty" gorm:"type:varchar(50);not null"`
	Email      string     `json:"email,omitempty" gorm:"index:query_index;type:varchar(50);unique_index;not null"`
	Phone      string     `json:"phone,omitempty" gorm:"index:query_index;type:varchar(15);unique_index;not null"`
	County     string     `json:"county,omitempty" gorm:"type:varchar(50);not null"`
	Profession string     `json:"profession,omitempty" gorm:"type:varchar(50);not null"`
	Gender     string     `json:"gender,omitempty" gorm:"type:varchar(10);default:'unknown'"`
	Verified   bool       `json:"verified" gorm:"type:tinyint(1);default:0"`
	Group      string     `json:"group,omitempty" gorm:"type:varchar(50);not null"`
	Password   string     `json:"password,omitempty" gorm:"type:text"`
	CreatedAt  time.Time  `json:"	-"`
	DeletedAt  *time.Time `json:"-"`
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
	case strings.TrimSpace(accountAPI.Password) == "":
		err = errors.New("password is required")
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
		http_error.Write(w, http_error.New("failed to decode request", err, http.StatusBadRequest))
		return
	}

	// Validation
	err = acc.validate()
	if err != nil {
		http_error.Write(w, http_error.New(err.Error(), err, http.StatusBadRequest))
		return
	}

	// Update account
	acc.Verified = false

	// Hash password
	acc.Password, err = genHash(acc.Password)
	if err != nil {
		http_error.Write(w, http_error.New("failed to generate password hash", err, http.StatusInternalServerError))
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
		http_error.Write(w, http_error.New(errMsg, err, http.StatusForbidden))
		return
	default:
		http_error.Write(w, http_error.New("failed to add user to database", err, http.StatusInternalServerError))
		return
	}

	// Write acc id to response
	err = json.NewEncoder(w).Encode(map[string]string{"account_id": acc.AccountID})
	if err != nil {
		http_error.Write(w, http_error.New("failed to json encode response", err, http.StatusInternalServerError))
		return
	}
}

func (accountAPI *accountAPI) GetAccount(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// Get acc id
	accountID := p.ByName("accountId")
	if strings.TrimSpace(accountID) == "" {
		errMsg := "missing account id"
		http_error.Write(w, http_error.New(errMsg, nil, http.StatusBadRequest))
		return
	}
	acc := &Account{}

	// Get from database
	err := accountAPI.sqlDB.First(acc, "account_id=?", accountID).Error
	switch {
	case err == nil:
	case gorm.IsRecordNotFoundError(err):
		http_error.Write(w, http_error.New("account not found", err, http.StatusBadRequest))
		return
	default:
		http_error.Write(w, http_error.New("failed to get account from db", err, http.StatusInternalServerError))
		return
	}

	// Write to connection
	err = json.NewEncoder(w).Encode(acc)
	if err != nil {
		http_error.Write(w, http_error.New("failed to json encode response", err, http.StatusInternalServerError))
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
		http_error.Write(w, http_error.New("failed to decode request", err, http.StatusBadRequest))
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
		http_error.Write(w, http_error.New(err.Error(), err, http.StatusBadRequest))
		return
	}

	// Get the acc
	acc := &Account{}
	err = accountAPI.sqlDB.First(acc, "email=? OR phone=?", login.UserName, login.UserName).Error
	switch {
	case err == nil:
	case gorm.IsRecordNotFoundError(err):
		http_error.Write(w, http_error.New("account not found", err, http.StatusBadRequest))
		return
	default:
		http_error.Write(w, http_error.New("failed to get account from db", err, http.StatusInternalServerError))
		return
	}

	// Compare password
	err = compareHash(acc.Password, login.Password)
	if err != nil {
		http_error.Write(w, http_error.New("incorrect password", err, http.StatusBadRequest))
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
		http_error.Write(w, http_error.New("failed to generate token", err, http.StatusBadRequest))
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
		http_error.Write(w, http_error.New("failed to json encode response", err, http.StatusInternalServerError))
		return
	}
}

func (accountAPI *accountAPI) UpdateAccount(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	accountID := p.ByName("accountId")
	if strings.TrimSpace(accountID) == "" {
		http_error.Write(w, http_error.New("missing account id", nil, http.StatusBadRequest))
		return
	}

	// Marshaling
	acc := &Account{}
	err := json.NewDecoder(r.Body).Decode(acc)
	if err != nil {
		http_error.Write(w, http_error.New("failed to decode request", err, http.StatusBadRequest))
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
		http_error.Write(w, http_error.New(errMsg, err, http.StatusBadRequest))
		return
	default:
		http_error.Write(w, http_error.New("failed to update account", err, http.StatusForbidden))
		return
	}

	// Write acc id to response
	err = json.NewEncoder(w).Encode(map[string]string{"account_id": accountID})
	if err != nil {
		http_error.Write(w, http_error.New("failed to json encode response", err, http.StatusInternalServerError))
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
		http_error.Write(w, http_error.New("failed to retrieve pagination data", err, http.StatusBadRequest))
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
		http_error.Write(w, http_error.New("failed get accounts", err, http.StatusInternalServerError))
		return
	}

	// Write response back
	err = json.NewEncoder(w).Encode(accounts)
	if err != nil {
		http_error.Write(w, http_error.New("failed to json encode response", err, http.StatusInternalServerError))
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
		http_error.Write(w, http_error.New("missing account id", nil, http.StatusBadRequest))
		return
	}

	// Marshaling
	changePass := &changePassword{}
	err := json.NewDecoder(r.Body).Decode(changePass)
	if err != nil {
		http_error.Write(w, http_error.New("failed to decode request", err, http.StatusBadRequest))
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
		http_error.Write(w, http_error.New(err.Error(), err, http.StatusBadRequest))
		return
	}

	// Hash password
	changePass.ConfirmPassword, err = genHash(changePass.ConfirmPassword)
	if err != nil {
		http_error.Write(w, http_error.New("failed to generate password hash", err, http.StatusInternalServerError))
		return
	}

	// Update model
	db := accountAPI.sqlDB.Table(accountsTable).Unscoped().Where("account_id=?", accountID).
		Update("password", changePass.ConfirmPassword)
	switch {
	case db.Error == nil:
	case db.RowsAffected == 0:
		http_error.Write(w, http_error.New("Aaccount does not exist", err, http.StatusNotFound))
		return
	case db.Error != nil:
		http_error.Write(w, http_error.New("failed to update", err, http.StatusInternalServerError))
		return
	}

	// Write some response
	w.Write([]byte("success"))
}

func (accountAPI *accountAPI) VerifyAccount(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	accountID := p.ByName("accountId")
	if strings.TrimSpace(accountID) == "" {
		http_error.Write(w, http_error.New("missing account id", nil, http.StatusBadRequest))
		return
	}

	// Update model
	db := accountAPI.sqlDB.Table(accountsTable).Unscoped().Where("account_id=?", accountID).
		Update("verified", true)
	switch {
	case db.Error == nil:
	case db.Error != nil:
		http_error.Write(w, http_error.New("failed to verify account", db.Error, http.StatusInternalServerError))
		return
	}

	// Write some response
	w.Write([]byte("verified"))
}

const (
	maxUploadSize       = int64(8 * 1024 * 1024)
	urlQueryKeyFormFile = "file"
)

func (accountAPI *accountAPI) AddUsersFromFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// validate size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		http_error.Write(w, http_error.New("failed to parse multi part form", err, http.StatusInternalServerError))
		return
	}

	// get file from request
	file, _, err := r.FormFile(urlQueryKeyFormFile)
	if err != nil {
		http_error.Write(w, http_error.New("failed to retrieve form", err, http.StatusInternalServerError))
		return
	}
	defer file.Close()

	// read file contents
	bs, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// detect content-type
	ctype := http.DetectContentType(bs)

	if ctype != "text/csv" {
		http_error.Write(w, http_error.New("only csv file is supported", nil, http.StatusBadRequest))
		return
	}

	tx := accountAPI.sqlDB.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
		}
	}()
	if tx.Error != nil {
		http_error.Write(w, http_error.New("failed to begin transaction", err, http.StatusInternalServerError))
		return
	}

	usersReader := csv.NewReader(file)
	usersReader.Comment = '#'

	for {
		record, err := usersReader.Read()
		if err != io.EOF {
			break
		}
		if err != nil {
			http_error.Write(w, http_error.New("failed to read from csv file", err, http.StatusInternalServerError))
			return
		}

		fullName := record[0]
		phoneNumber := record[1]
		county := record[2]

		// Save user in database
		userDB := &services.UserModel{
			PhoneNumber: phoneNumber,
			FullName:    fullName,
			County:      county,
			Status:      int8(location.Status_POSITIVE),
			DeviceToken: "NA",
		}

		// If user already exists, performs an update
		alreadyExists := !accountAPI.sqlDB.
			First(&services.UserModel{}, "phone_number=?", phoneNumber).RecordNotFound()

		if alreadyExists {
			err = accountAPI.sqlDB.Table(services.UsersTable).Where("phone_number=?", phoneNumber).
				Updates(userDB).Error
			switch {
			case err == nil:
			default:
				http_error.Write(w, http_error.New("failed to save user", err, http.StatusInternalServerError))
				tx.Rollback()
				return
			}
		} else {
			// Save user
			err = accountAPI.sqlDB.Save(userDB).Error
			switch {
			case err == nil:
			default:
				http_error.Write(w, http_error.New("failed to update user", err, http.StatusInternalServerError))
				tx.Rollback()
				return
			}
		}
	}

	err = tx.Commit().Error
	if err != nil {
		http_error.Write(w, http_error.New("failed to commit transaction", err, http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("all records saved successffulyy"))
}
