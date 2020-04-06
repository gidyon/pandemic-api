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

// RegisterFAQAPIRouter registers a router for the FAQ API
func RegisterFAQAPIRouter(router *httprouter.Router, opt *Options) {
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

	faq := &faqAPI{
		rootDir:    opt.RootDir,
		filePrefix: opt.FilePrefix,
		mu:         &sync.RWMutex{},
		faq:        &FAQData{},
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	if err != nil {
		logrus.Fatalln(err)
	}

	// update data from file
	err = json.NewDecoder(file).Decode(faq.faq)
	if err != nil {
		logrus.Fatalln(err)
	}

	// Update endpoints
	router.GET("/faqs", faq.GetFAQ)
	router.POST("/faqs", faq.UpdateFaq)
}

type faqAPI struct {
	rootDir    string
	filePrefix string
	mu         *sync.RWMutex
	faq        *FAQData
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

func (faq *faqAPI) GetFAQ(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	faq.mu.RLock()
	err := json.NewEncoder(w).Encode(faq.faq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	faq.mu.RUnlock()
}

func (faq *faqAPI) UpdateFaq(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	newfaq := &FAQData{}
	err := json.NewDecoder(r.Body).Decode(newfaq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = newfaq.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update time
	newfaq.LastUpdated = time.Now()

	fileName := filepath.Join(faq.rootDir, fmt.Sprintf("%s-v%d.json", faq.filePrefix, newfaq.Revision))
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Write to file
	err = json.NewEncoder(file).Encode(newfaq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	faq.mu.Lock()
	faq.faq = newfaq
	faq.mu.Unlock()
}
