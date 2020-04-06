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

// Options contains parameters for NewSymptomsAPI
type Options struct {
	RootDir    string
	FilePrefix string
	Revision   int
}

// RegisterSymptomsAPIRouter registers http router for the symptoms API
func RegisterSymptomsAPIRouter(router *httprouter.Router, opt *Options) {
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
	if err != nil {
		logrus.Fatalln(err)
	}

	symptoms := &symptomsAPI{
		rootDir:    opt.RootDir,
		filePrefix: opt.FilePrefix,
		mu:         &sync.RWMutex{},
		symptoms:   &SymptomsData{},
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	if err != nil {
		logrus.Fatalln(err)
	}

	// update data from file
	err = json.NewDecoder(file).Decode(symptoms.symptoms)
	if err != nil {
		logrus.Fatalln(err)
	}

	// Update endpoints
	router.GET("/symptoms", symptoms.GetSymptoms)
	router.POST("/symptoms", symptoms.UpdateSymptom)
}

type symptomsAPI struct {
	rootDir    string
	filePrefix string
	mu         *sync.RWMutex
	symptoms   *SymptomsData
}

// SymptomsData contains Frequently Asked Questions
type SymptomsData struct {
	Revision           int       `json:"revision,omitempty"`
	Source             string    `json:"source,omitempty"`
	SourceOrganization string    `json:"source_organization,omitempty"`
	LastUpdated        time.Time `json:"last_updated,omitempty"`
	Symptoms           []struct {
		Prevalence string `json:"prevalence,omitempty"`
		Symptom    string `json:"symptom,omitempty"`
	} `json:"symptoms,omitempty"`
}

func (symptoms *SymptomsData) validate() error {
	var err error
	switch {
	case symptoms.Revision == 0:
		err = errors.New("revision number missing")
	case len(symptoms.Symptoms) == 0:
		err = errors.New("symptoms measures missing")
	case strings.TrimSpace(symptoms.Source) == "":
		err = errors.New("source is required")
	case strings.TrimSpace(symptoms.SourceOrganization) == "":
		err = errors.New("source organization is required")
	}
	return err
}

func (symptoms *symptomsAPI) GetSymptoms(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	symptoms.mu.RLock()
	err := json.NewEncoder(w).Encode(symptoms.symptoms)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	symptoms.mu.RUnlock()
}

func (symptoms *symptomsAPI) UpdateSymptom(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	newSymptoms := &SymptomsData{}
	err := json.NewDecoder(r.Body).Decode(newSymptoms)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = newSymptoms.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update time
	newSymptoms.LastUpdated = time.Now()

	fileName := filepath.Join(symptoms.rootDir, symptoms.filePrefix, fmt.Sprintf("-v%d.json", newSymptoms.Revision))
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Write to file
	err = json.NewEncoder(file).Encode(newSymptoms)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	symptoms.mu.Lock()
	symptoms.symptoms = newSymptoms
	symptoms.mu.Unlock()
}
