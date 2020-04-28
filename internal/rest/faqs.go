package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Sirupsen/logrus"
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

const faqGroup = "FAQ"

type faqAPI struct {
	rootDir    string
	filePrefix string
	revision   int
	mu         *sync.RWMutex
	faqs       map[int]*FAQData
}

// RegisterFAQAPI registers a router for the FAQ API
func RegisterFAQAPI(router *httprouter.Router, opt *Options) {
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

	faqAPI := &faqAPI{
		rootDir:    opt.RootDir,
		filePrefix: opt.FilePrefix,
		revision:   opt.Revision,
		mu:         &sync.RWMutex{},
		faqs:       make(map[int]*FAQData, 0),
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	handleError(err)
	defer file.Close()

	// update data from file
	faq := &FAQData{}
	err = json.NewDecoder(file).Decode(faq)

	// get json of faq
	bs, err := json.Marshal(faq)
	handleError(err)

	// add the fat to database
	err = revisionManager.Add(&revision{
		Revision:      faq.Revision,
		ResourceGroup: faqGroup,
		Data:          bs,
	})
	handleError(err)

	dur := time.Duration(int(30*time.Minute) + rand.Intn(30))

	go updateRevisionWorker(dur, func() {
		// get new revision
		revisions, err := revisionManager.List(faqGroup)
		if err != nil {
			logrus.Infof("failed to list revisions from database: %v", err)
			return
		}

		// Update Map
		faqAPI.mu.Lock()
		defer faqAPI.mu.Unlock()

		for _, revision := range revisions {
			faq := &FAQData{}
			err = json.Unmarshal(revision.Data, faq)
			if err != nil {
				logrus.Infof("failed to unmarshal revision: %v", err)
				continue
			}

			faqAPI.faqs[revision.Revision] = faq
			faqAPI.revision = revision.Revision
		}

		logrus.Infoln("FAQs updated")
	})

	// Update endpoints
	router.GET("/rest/v1/faqs", faqAPI.GetFAQ)
	router.PUT("/rest/v1/faqs", faqAPI.UpdateFaq)
}

// FAQData contains Frequently Asked Questions
type FAQData struct {
	Revision           int       `json:"revision,omitempty"`
	Source             string    `json:"source,omitempty"`
	SourceOrganization string    `json:"source_organization,omitempty"`
	LastUpdated        time.Time `json:"last_updated,omitempty"`
	FAQs               []struct {
		Question string   `json:"question,omitempty"`
		Answers  []string `json:"answers,omitempty"`
	} `json:"faqs,omitempty"`
}

func (faq *FAQData) validate() error {
	var err error
	switch {
	case faq.Revision == 0:
		err = errors.New("revision number missing")
	case len(faq.FAQs) == 0:
		err = errors.New("FAQs missing")
	case strings.TrimSpace(faq.Source) == "":
		err = errors.New("missing source")
	case strings.TrimSpace(faq.SourceOrganization) == "":
		err = errors.New("missing source organization")
	}
	return err
}

func (faqAPI *faqAPI) GetFAQ(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	revisionStr := r.URL.Query().Get("revision")
	if revisionStr == "" {
		revisionStr = "0"
	}
	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		http.Error(w, "failed to convert revision to number", http.StatusBadRequest)
		return
	}

	faqAPI.mu.RLock()
	defer faqAPI.mu.RUnlock()

	faq, ok := faqAPI.faqs[revision]
	if !ok {
		err := json.NewEncoder(w).Encode(faqAPI.faqs[faqAPI.revision])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	err = json.NewEncoder(w).Encode(faq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (faqAPI *faqAPI) UpdateFaq(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	newFaq := &FAQData{}
	err := json.NewDecoder(r.Body).Decode(newFaq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = newFaq.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update time
	newFaq.LastUpdated = time.Now()

	// Get new revision json
	bs, err := json.Marshal(newFaq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add revision to database
	err = revisionManager.Add(&revision{
		Revision:      newFaq.Revision,
		ResourceGroup: faqGroup,
		Data:          bs,
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to add revision to db: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	w.Write([]byte("faqs scheduled for update"))
}
