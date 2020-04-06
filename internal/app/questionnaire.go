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

// RegisterQuestionnaireAPIRouter registers http router for the questionnaire API
func RegisterQuestionnaireAPIRouter(router *httprouter.Router, opt *Options) {
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

	questionnaire := &questionnaireAPI{
		rootDir:       opt.RootDir,
		filePrefix:    opt.FilePrefix,
		mu:            &sync.RWMutex{},
		questionnaire: &QuestionnaireData{},
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	if err != nil {
		logrus.Fatalln(err)
	}

	// update data from file
	err = json.NewDecoder(file).Decode(questionnaire.questionnaire)
	if err != nil {
		logrus.Fatalln(err)
	}

	// Update endpoints
	router.GET("/questionnaire", questionnaire.GetQuestionnaire)
	router.POST("/questionnaire", questionnaire.UpdateQuestionnaire)
}

type questionnaireAPI struct {
	rootDir       string
	filePrefix    string
	mu            *sync.RWMutex
	questionnaire *QuestionnaireData
}

// QuestionnaireData contains Frequently Asked Questions
type QuestionnaireData struct {
	Revision           int       `json:"revision,omitempty"`
	Source             string    `json:"source,omitempty"`
	SourceOrganization string    `json:"source_organization,omitempty"`
	LastUpdated        time.Time `json:"last_updated,omitempty"`
	Questionnaire      []struct {
		Key         string `json:"key,omitempty"`
		Question    string `json:"question,omitempty"`
		AnswerRadio struct {
			Text  string `json:"text,omitempty"`
			Value string `json:"value,omitempty"`
		} `json:"answer_radio,omitempty"`
		AnswerMulti []struct {
			Text  string `json:"text,omitempty"`
			Value string `json:"value,omitempty"`
		} `json:"answer_multi,omitempty"`
		AnswersSelection []struct {
			Text  string `json:"text,omitempty"`
			Value string `json:"value,omitempty"`
		} `json:"answers_selection,omitempty"`
		Multi bool `json:"multi,omitempty"`
	} `json:"questionnaire,omitempty"`
}

func (questionnaire *QuestionnaireData) validate() error {
	var err error
	switch {
	case questionnaire.Revision == 0:
		err = errors.New("revision number missing")
	case len(questionnaire.Questionnaire) == 0:
		err = errors.New("questionnaires measures missing")
	case strings.TrimSpace(questionnaire.Source) == "":
		err = errors.New("source is required")
	case strings.TrimSpace(questionnaire.SourceOrganization) == "":
		err = errors.New("source organization is required")
	}
	return err
}

func (questionnaire *questionnaireAPI) GetQuestionnaire(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	questionnaire.mu.RLock()
	err := json.NewEncoder(w).Encode(questionnaire.questionnaire)
	if err != nil {
		questionnaire.mu.RUnlock()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	questionnaire.mu.RUnlock()
}

func (questionnaire *questionnaireAPI) UpdateQuestionnaire(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Marshaling
	newQuestionnaire := &QuestionnaireData{}
	err := json.NewDecoder(r.Body).Decode(newQuestionnaire)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validation
	err = newQuestionnaire.validate()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update time
	newQuestionnaire.LastUpdated = time.Now()

	fileName := filepath.Join(questionnaire.rootDir, fmt.Sprintf("%s-v%d.json", questionnaire.filePrefix, newQuestionnaire.Revision))
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Write to file
	err = json.NewEncoder(file).Encode(newQuestionnaire)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	questionnaire.mu.Lock()
	questionnaire.questionnaire = newQuestionnaire
	questionnaire.mu.Unlock()
}
