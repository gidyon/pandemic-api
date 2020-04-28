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

const quarantineGroup = "QUARANTINE"

type quarantineAPI struct {
	rootDir     string
	filePrefix  string
	revision    int
	mu          *sync.RWMutex
	quarantines map[int]*QuarantineData
}

// RegisterQuarantineMeasuresAPI registers a http router for the quarantines API
func RegisterQuarantineMeasuresAPI(router *httprouter.Router, opt *Options) {
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

	quarantineAPI := &quarantineAPI{
		rootDir:     opt.RootDir,
		filePrefix:  opt.FilePrefix,
		revision:    opt.Revision,
		mu:          &sync.RWMutex{},
		quarantines: make(map[int]*QuarantineData, 0),
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	handleError(err)

	// update data from file
	quarantine := &QuarantineData{}
	err = json.NewDecoder(file).Decode(quarantine)

	// get json
	bs, err := json.Marshal(quarantine)
	handleError(err)

	// add the quarantine only if it doesn't exist
	_, err = revisionManager.Get(quarantineGroup, quarantine.Revision)
	if gorm.IsRecordNotFoundError(err) {
		err = revisionManager.Add(&revision{
			Revision:      quarantine.Revision,
			ResourceGroup: quarantineGroup,
			Data:          bs,
		})
		handleError(err)
	}

	dur := time.Duration(int(30*time.Minute) + rand.Intn(30))

	go updateRevisionWorker(dur, func() {
		// get new revision
		revisions, err := revisionManager.List(quarantineGroup)
		if err != nil {
			logrus.Infof("failed to list revisions from database: %v", err)
			return
		}

		// Update Map
		quarantineAPI.mu.Lock()
		defer quarantineAPI.mu.Unlock()

		for _, revision := range revisions {
			quarantine := &QuarantineData{}
			err = json.Unmarshal(revision.Data, quarantine)
			if err != nil {
				logrus.Infof("failed to unmarshal revision: %v", err)
				continue
			}

			quarantineAPI.quarantines[revision.Revision] = quarantine
			quarantineAPI.revision = revision.Revision
		}

		logrus.Infoln("Quarantine measures updated")
	})

	// Update endpoints
	router.GET("/rest/v1/quarantine/measures", quarantineAPI.GetMeasures)
	router.PUT("/rest/v1/quarantine", quarantineAPI.UpdateQurantine)
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

func (quarantineAPI *quarantineAPI) GetMeasures(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	revisionStr := r.URL.Query().Get("revision")
	if revisionStr == "" {
		revisionStr = "0"
	}
	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		http.Error(w, "failed to convert revision to number", http.StatusBadRequest)
		return
	}

	quarantineAPI.mu.RLock()
	defer quarantineAPI.mu.RUnlock()

	quarantine, ok := quarantineAPI.quarantines[revision]
	if !ok {
		err := json.NewEncoder(w).Encode(quarantineAPI.quarantines[quarantineAPI.revision])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	err = json.NewEncoder(w).Encode(quarantine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (quarantineAPI *quarantineAPI) UpdateQurantine(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	newQuarantine := &QuarantineData{}
	err := json.NewDecoder(r.Body).Decode(newQuarantine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = newQuarantine.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update time
	newQuarantine.LastUpdated = time.Now()

	// Get new revision json
	bs, err := json.Marshal(newQuarantine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add revision to database
	err = revisionManager.Add(&revision{
		Revision:      newQuarantine.Revision,
		ResourceGroup: quarantineGroup,
		Data:          bs,
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to add revision to db: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	w.Write([]byte("quarantine measures scheduled for update"))
}
