package upload

import (
	http_error "github.com/gidyon/pandemic-api/pkg/errors"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"io/ioutil"
	"mime"
	"net/http"
	"time"
)

const (
	maxUploadSize       = int64(8 * 1024 * 1024)
	urlQueryKeyFormFile = "file"
	urlQueryKeyOwnerID  = "owner"
)

type uploadsServer struct {
	sqlDB *gorm.DB
}

// fileMeta contains metadata about a file
type fileMeta struct {
	UploaderID string
	Mime       string
	Size       int64
	Name       string
	Path       string
	Data       []byte
	CreateAt   time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

// FileHandler handles file uploads of confirmed cases
func (us *uploadsServer) ServerHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		us.saveFile(w, r)
	case http.MethodGet:
		us.downloadFile(w, r)
	}
}

func (us *uploadsServer) downloadFile(w http.ResponseWriter, r *http.Request) {

}

func (us *uploadsServer) saveFile(w http.ResponseWriter, r *http.Request) {
	// validate size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		http_error.Write(w, http_error.New("failed to parse multi part form", err, http.StatusInternalServerError))
		return
	}

	// get file from request
	file, header, err := r.FormFile(urlQueryKeyFormFile)
	if err != nil {
		http_error.Write(w, http_error.New("failed to retrieve form", err, http.StatusInternalServerError))
		return
	}
	defer file.Close()

	// read content
	bs, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// detect content-type
	ctype := http.DetectContentType(bs)

	fileName, err := func() (string, error) {
		if header.Filename != "" {
			return header.Filename, nil
		}
		fileEndings, err := mime.ExtensionsByType(ctype)
		if err != nil {
			return "", errors.New("CANT_READ_FILE_EXT_TYPE")
		}
		return uuid.New().String() + fileEndings[0], nil
	}()
	if err != nil {
		http_error.Write(w, http_error.New("failed to retrieve filename", err, http.StatusInternalServerError))
		return
	}

	// Create a transaction
	tx := us.sqlDB.Begin()
	defer func() {
		if err := recover(); err != nil {
			http_error.Write(w, http_error.New("panic happened in transaction", nil, http.StatusInternalServerError))
		}
	}()
	if tx.Error != nil {
		http_error.Write(w, http_error.New("failed to begin transaction", tx.Error, http.StatusInternalServerError))
		return
	}

	path := ""

	// save file info metadata to database
	fileInfo := fileMeta{
		UploaderID: r.URL.Query().Get(urlQueryKeyOwnerID),
		Mime:       ctype,
		Size:       header.Size,
		Name:       fileName,
		Path:       path,
		Data:       bs,
	}

	// save file info
	err = tx.Unscoped().Save(fileInfo).Error
	if err != nil {
		tx.Rollback()
		http_error.Write(w, http_error.New("failed to save file", err, http.StatusInternalServerError))
		return
	}

	// Save users data manually

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("saved all users successfully"))
}
