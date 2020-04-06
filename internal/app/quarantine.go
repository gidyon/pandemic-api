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

// RegisterQuarantineAPIRouter registers a http router for the quarantines API
func RegisterQuarantineAPIRouter(router *httprouter.Router, opt *Options) {
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

	quarantine := &quarantineAPI{
		rootDir:    opt.RootDir,
		filePrefix: opt.FilePrefix,
		mu:         &sync.RWMutex{},
		quarantine: &QuarantineData{},
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	if err != nil {
		logrus.Fatalln(err)
	}

	// update data from file
	err = json.NewDecoder(file).Decode(quarantine.quarantine)
	if err != nil {
		logrus.Fatalln(err)
	}

	// Update endpoints
	router.GET("/quarantine/measures", quarantine.GetMeasures)
	router.POST("/quarantine", quarantine.UpdateQurantine)
}

type quarantineAPI struct {
	rootDir    string
	filePrefix string
	mu         *sync.RWMutex
	quarantine *QuarantineData
}

// QuarantineData contains Frequently Asked Questions
type QuarantineData struct {
	Revision           int       `json:"revision,omitempty"`
	Source             string    `json:"source,omitempty"`
	SourceOrganization string    `json:"source_organization,omitempty"`
	LastUpdated        time.Time `json:"last_updated,omitempty"`
	Definition         string    `json:"definition,omitempty"`
	Audience           string    `json:"audience,omitempty"`
	Measures           []string  `json:"measures,omitempty"`
}

func (quarantine *QuarantineData) validate() error {
	var err error
	switch {
	case quarantine.Revision == 0:
		err = errors.New("revision number missing")
	case strings.TrimSpace(quarantine.Definition) == "":
		err = errors.New("definition is required")
	case strings.TrimSpace(quarantine.Audience) == "":
		err = errors.New("audience is required")
	case strings.TrimSpace(quarantine.Source) == "":
		err = errors.New("source is required")
	case strings.TrimSpace(quarantine.SourceOrganization) == "":
		err = errors.New("source organization is required")
	case len(quarantine.Measures) == 0:
		err = errors.New("quarantines measures is required")
	}
	return err
}

func (quarantine *quarantineAPI) GetMeasures(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	quarantine.mu.RLock()
	err := json.NewEncoder(w).Encode(quarantine.quarantine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	quarantine.mu.RUnlock()
}

func (quarantine *quarantineAPI) UpdateQurantine(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	newquarantine := &QuarantineData{}
	err := json.NewDecoder(r.Body).Decode(newquarantine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = newquarantine.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update time
	newquarantine.LastUpdated = time.Now()

	fileName := filepath.Join(quarantine.rootDir, fmt.Sprintf("%s-v%d.json", quarantine.filePrefix, newquarantine.Revision))
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Write new measures to file
	err = json.NewEncoder(file).Encode(newquarantine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	quarantine.mu.Lock()
	quarantine.quarantine = newquarantine
	quarantine.mu.Unlock()
}
