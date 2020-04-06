package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type contactAPI struct {
	rootDir    string
	filePrefix string
	mu         *sync.RWMutex
	contacts   *ContactData
}

// RegisterContactAPIRouter registers http router for the contact API
func RegisterContactAPIRouter(router *httprouter.Router, opt *Options) {
	// Validation
	var err error
	switch {
	case router == nil:
		err = errors.New("router must not be nil")
	case strings.TrimSpace(opt.RootDir) == "":
		err = errors.New("root directory must not be empty")
	case strings.TrimSpace(opt.FilePrefix) == "":
		err = errors.New("file prefix must not be empty")
	case opt.Revision == 0:
		err = errors.New("revision must not be 0")
	}

	c := &contactAPI{
		rootDir:    opt.RootDir,
		filePrefix: opt.FilePrefix,
		mu:         &sync.RWMutex{},
		contacts:   &ContactData{},
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	if err != nil {
		logrus.Fatalln(err)
	}

	// update data from file
	err = json.NewDecoder(file).Decode(c.contacts)
	if err != nil {
		logrus.Fatalln(err)
	}

	// Update endpoints
	router.GET("/contacts", c.GetContact)
	router.POST("/contacts", c.UpdateContact)
}

// ContactData contains contact data
type ContactData struct {
	Revision           int       `json:"revision,omitempty"`
	Source             string    `json:"source,omitempty"`
	SourceOrganization string    `json:"source_organization,omitempty"`
	LastUpdated        time.Time `json:"last_updated,omitempty"`
	GeneralHotlines    []struct {
		Number      string `json:"number,omitempty"`
		Description string `json:"description,omitempty"`
	} `json:"general_hotlines,omitempty"`
	GeneralEmails []struct {
		Email       string `json:"email,omitempty"`
		Description string `json:"description,omitempty"`
	} `json:"general_emails,omitempty"`
	CountiesHotlines []struct {
		County   string   `json:"county,omitempty"`
		Hotlines []string `json:"hotlines,omitempty"`
		Emails   []string `json:"emails,omitempty"`
	} `json:"counties_hotlines,omitempty"`
}

func (c *ContactData) validate() error {
	var err error
	switch {
	case c.Revision == 0:
		err = errors.New("revision number missing")
	case strings.TrimSpace(c.Source) == "":
		err = errors.New("missing source")
	case strings.TrimSpace(c.SourceOrganization) == "":
		err = errors.New("missing source organization")
	case len(c.GeneralHotlines) == 0:
		err = errors.New("general hotlines missing")
	case len(c.GeneralEmails) == 0:
		err = errors.New("general emails missing")
	case len(c.CountiesHotlines) == 0:
		err = errors.New("counties hotlines missing")
	}
	return err
}

func (contact *contactAPI) GetContact(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	contact.mu.RLock()
	err := json.NewEncoder(w).Encode(contact.contacts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	contact.mu.RUnlock()
}

func (contact *contactAPI) UpdateContact(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Unmarshaling
	newContact := &ContactData{}
	err := json.NewDecoder(r.Body).Decode(newContact)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = newContact.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update time
	newContact.LastUpdated = time.Now()

	// Create local file backup
	fileName := filepath.Join(contact.rootDir, fmt.Sprintf("%s-v%d.json", contact.filePrefix, newContact.Revision))
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Writing response
	err = json.NewEncoder(file).Encode(newContact)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	contact.mu.Lock()
	contact.contacts = newContact
	contact.mu.Unlock()
}
