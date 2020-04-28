package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
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

const symptomGroup = "SYMPTOMS"

// Options contains parameters for NewSymptomsAPI
type Options struct {
	RootDir    string
	FilePrefix string
	Revision   int
}

// RegisterPandemicSymptomsAPI registers http router for the symptoms API
func RegisterPandemicSymptomsAPI(router *httprouter.Router, opt *Options) {
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

	symptomsAPI := &symptomsAPI{
		rootDir:    opt.RootDir,
		filePrefix: opt.FilePrefix,
		mu:         &sync.RWMutex{},
		symptoms:   make(map[int]*SymptomsData, 0),
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	handleError(err)

	// update data from file
	symptom := &SymptomsData{}
	err = json.NewDecoder(file).Decode(symptom)

	// get json
	bs, err := json.Marshal(symptom)
	handleError(err)

	// add the symptom only if it doesn't exist
	_, err = revisionManager.Get(symptomGroup, symptom.Revision)
	if gorm.IsRecordNotFoundError(err) {
		err = revisionManager.Add(&revision{
			Revision:      symptom.Revision,
			ResourceGroup: symptomGroup,
			Data:          bs,
		})
		handleError(err)
	}

	dur := time.Duration(int(30*time.Minute) + rand.Intn(30))

	go updateRevisionWorker(dur, func() {
		// get new revision
		revisions, err := revisionManager.List(symptomGroup)
		if err != nil {
			logrus.Infof("failed to list revisions from database: %v", err)
			return
		}

		// Update Map
		symptomsAPI.mu.Lock()
		defer symptomsAPI.mu.Unlock()

		for _, revision := range revisions {
			symptom := &SymptomsData{}
			err = json.Unmarshal(revision.Data, symptom)
			if err != nil {
				logrus.Infof("failed to unmarshal revision: %v", err)
				continue
			}

			symptomsAPI.symptoms[revision.Revision] = symptom
			symptomsAPI.revision = revision.Revision
		}

		logrus.Infoln("Symptoms measures updated")
	})

	// Update endpoints
	router.GET("/rest/v1/symptoms", symptomsAPI.GetSymptoms)
	router.PUT("/rest/v1/symptoms", symptomsAPI.UpdateSymptom)
}

type symptomsAPI struct {
	rootDir    string
	filePrefix string
	revision   int
	mu         *sync.RWMutex
	symptoms   map[int]*SymptomsData
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

func (symptomsAPI *symptomsAPI) GetSymptoms(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	revisionStr := r.URL.Query().Get("revision")
	if revisionStr == "" {
		revisionStr = "0"
	}
	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		http.Error(w, "failed to convert revision to number", http.StatusBadRequest)
		return
	}

	symptomsAPI.mu.RLock()
	defer symptomsAPI.mu.RUnlock()

	symptom, ok := symptomsAPI.symptoms[revision]
	if !ok {
		err = json.NewEncoder(w).Encode(symptomsAPI.symptoms[symptomsAPI.revision])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	err = json.NewEncoder(w).Encode(symptom)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (symptomsAPI *symptomsAPI) UpdateSymptom(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	// Get new revision json
	bs, err := json.Marshal(newSymptoms)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add revision to database
	err = revisionManager.Add(&revision{
		Revision:      newSymptoms.Revision,
		ResourceGroup: symptomGroup,
		Data:          bs,
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to add revision to db: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	w.Write([]byte("symptoms scheduled for update"))
}
