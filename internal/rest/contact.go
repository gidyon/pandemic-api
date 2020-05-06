package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
	http_error "github.com/gidyon/pandemic-api/pkg/errors"
	"github.com/jinzhu/gorm"
	"github.com/julienschmidt/httprouter"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const contactGroup = "CONTACTS"

type contactAPI struct {
	rootDir     string
	filePrefix  string
	revision    int
	mu          *sync.RWMutex
	contacts    map[int]*ContactData
	countiesMap map[string]int
}

// RegisterContactAPI registers http router for the contact API
func RegisterContactAPI(router *httprouter.Router, opt *Options) {
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
	handleError(err)

	c := &contactAPI{
		rootDir:     opt.RootDir,
		filePrefix:  opt.FilePrefix,
		revision:    opt.Revision,
		mu:          &sync.RWMutex{},
		contacts:    make(map[int]*ContactData, 0),
		countiesMap: make(map[string]int, 0),
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	handleError(err)
	defer file.Close()

	// update data from file
	contact := &ContactData{}
	err = json.NewDecoder(file).Decode(contact)
	handleError(err)

	// get json
	bs, err := json.Marshal(contact)
	handleError(err)

	// Update contacts map
	for index, countyHotline := range contact.CountiesHotlines {
		c.countiesMap[countyHotline.County] = index
	}

	// add the contact only if it doesn't exist
	_, err = revisionManager.Get(contactGroup, contact.Revision)
	if gorm.IsRecordNotFoundError(err) {
		err = revisionManager.Add(&revision{
			Revision:      contact.Revision,
			ResourceGroup: contactGroup,
			Data:          bs,
		})
		handleError(err)
	}

	dur := time.Duration(int(30*time.Minute) + rand.Intn(30))

	go updateRevisionWorker(dur, func() {
		// get new revision
		revisions, err := revisionManager.List(contactGroup)
		if err != nil {
			logrus.Infof("failed to list revisions from database: %v", err)
			return
		}

		// Update Map
		c.mu.Lock()
		defer c.mu.Unlock()

		for _, revision := range revisions {
			contact := &ContactData{}
			err = json.Unmarshal(revision.Data, contact)
			if err != nil {
				logrus.Infof("failed to unmarshal revision: %v", err)
				continue
			}

			c.contacts[revision.Revision] = contact
			c.revision = revision.Revision
		}

		logrus.Infoln("Contacts updated")
	})

	// Update endpoints
	router.GET("/rest/v1/contacts", c.GetContact)
	router.PUT("/rest/v1/contacts", c.UpdateContact)
	router.GET("/rest/v1/contacts/hotlines/nearest", c.GetNearestHotlines)
	router.GET("/rest/v1/contacts/hotlines/county", c.GetCountyHotlines)
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
	revisionStr := r.URL.Query().Get("revision")
	if revisionStr == "" {
		revisionStr = "0"
	}
	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		http.Error(w, "failed to convert revision to number", http.StatusBadRequest)
		return
	}

	contact.mu.RLock()
	defer contact.mu.RUnlock()

	contacts, ok := contact.contacts[revision]
	if !ok {
		err := json.NewEncoder(w).Encode(contact.contacts[contact.revision])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	err = json.NewEncoder(w).Encode(contacts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

	// Get new revision json
	bs, err := json.Marshal(newContact)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add revision to database
	err = revisionManager.Add(&revision{
		Revision:      newContact.Revision,
		ResourceGroup: contactGroup,
		Data:          bs,
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to add revision to db: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	w.Write([]byte("contacts scheduled for update"))
}

// GetCountyHotlines fetches hotlines contacts for the county
func (contact *contactAPI) GetCountyHotlines(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Validation
	county := r.URL.Query().Get("county")
	if county == "" {
		http_error.Write(w, http_error.New("county is required", nil, http.StatusBadRequest))
		return
	}

	contact.mu.RLock()
	defer contact.mu.RUnlock()

	hotlines := make([]string, 0)

	index, ok := contact.countiesMap[county]
	if !ok {
		for _, generalHotline := range contact.contacts[contact.revision].GeneralHotlines {
			hotlines = append(hotlines, fmt.Sprintf("%s - %s", generalHotline.Number, generalHotline.Description))
		}

		err := json.NewEncoder(w).Encode(hotlines)
		if err != nil {
			http_error.Write(w, http_error.New("failed to json encode", err, http.StatusBadRequest))
			return
		}

		return
	}

	for _, hotline := range contact.contacts[contact.revision].CountiesHotlines[index].Hotlines {
		hotlines = append(hotlines, hotline)
	}

	err := json.NewEncoder(w).Encode(hotlines)
	if err != nil {
		http_error.Write(w, http_error.New("failed to json encode", err, http.StatusBadRequest))
		return
	}
}

// GetNearestHotlines fetches nearest hotlines around the user
func (contact *contactAPI) GetNearestHotlines(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Validation
	county := r.URL.Query().Get("county")
	longitude := r.URL.Query().Get("longitude")
	latitude := r.URL.Query().Get("latitude")

	var err error
	switch {
	case county == "":
		http_error.Write(w, http_error.New("county is required", nil, http.StatusBadRequest))
		return
	case longitude == "":
		http_error.Write(w, http_error.New("longitude is required", nil, http.StatusBadRequest))
		return
	case latitude == "":
		http_error.Write(w, http_error.New("latitude is required", nil, http.StatusBadRequest))
		return
	}

	contact.mu.RLock()
	defer contact.mu.RUnlock()

	hotlines := make([]string, 0)

	index, ok := contact.countiesMap[county]
	if !ok {
		for _, generalHotline := range contact.contacts[contact.revision].GeneralHotlines {
			hotlines = append(hotlines, fmt.Sprintf("%s - %s", generalHotline.Number, generalHotline.Description))
		}

		err := json.NewEncoder(w).Encode(hotlines)
		if err != nil {
			http_error.Write(w, http_error.New("failed to json encode hotlines", err, http.StatusBadRequest))
			return
		}

		return
	}

	for _, hotline := range contact.contacts[contact.revision].CountiesHotlines[index].Hotlines {
		hotlines = append(hotlines, hotline)
	}

	err = json.NewEncoder(w).Encode(hotlines)
	if err != nil {
		http_error.Write(w, http_error.New("failed to json encode", err, http.StatusBadRequest))
		return
	}
}
