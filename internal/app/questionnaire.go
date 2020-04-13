package app

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

const questionnaireGroup = "QUESTIONNAIRE"

type questionnaireAPI struct {
	rootDir        string
	filePrefix     string
	revision       int
	mu             *sync.RWMutex
	questionnaires map[int]*QuestionnaireData
}

// RegisterQuestionnaireAPI registers http router for the questionnaire API
func RegisterQuestionnaireAPI(router *httprouter.Router, opt *Options) {
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

	questionnaireAPI := &questionnaireAPI{
		rootDir:        opt.RootDir,
		filePrefix:     opt.FilePrefix,
		mu:             &sync.RWMutex{},
		questionnaires: make(map[int]*QuestionnaireData, 0),
	}

	// read from file
	file, err := os.Open(filepath.Join(opt.RootDir, fmt.Sprintf("%s-v%d.json", opt.FilePrefix, opt.Revision)))
	handleError(err)

	// update data from file
	questionnaire := &QuestionnaireData{}
	err = json.NewDecoder(file).Decode(questionnaire)

	// get json
	bs, err := json.Marshal(questionnaire)
	handleError(err)

	// add the questionnaire only if it doesn't exist
	_, err = revisionManager.Get(questionnaireGroup, questionnaire.Revision)
	if gorm.IsRecordNotFoundError(err) {
		err = revisionManager.Add(&revision{
			Revision:      questionnaire.Revision,
			ResourceGroup: questionnaireGroup,
			Data:          bs,
		})
		handleError(err)
	}

	dur := time.Duration(int(30*time.Second) + rand.Intn(30))

	go updateRevisionWorker(dur, func() {
		// get new revision
		revisions, err := revisionManager.List(questionnaireGroup)
		if err != nil {
			logrus.Infof("failed to list revisions from database: %v", err)
			return
		}

		// Update Map
		questionnaireAPI.mu.Lock()
		defer questionnaireAPI.mu.Unlock()

		for _, revision := range revisions {
			questionnaire := &QuestionnaireData{}
			err = json.Unmarshal(revision.Data, questionnaire)
			if err != nil {
				logrus.Infof("failed to unmarshal revision: %v", err)
				continue
			}

			questionnaireAPI.questionnaires[revision.Revision] = questionnaire
			questionnaireAPI.revision = revision.Revision
		}

		logrus.Infoln("Questionnaire measures updated")
	})

	// Update endpoints
	router.GET("/api/questionnaire", questionnaireAPI.GetQuestionnaire)
	router.PUT("/api/questionnaire", questionnaireAPI.UpdateQuestionnaire)
}

// QuestionnaireData contains Frequently Asked Questions
type QuestionnaireData struct {
	Revision           int       `json:"revision,omitempty"`
	Source             string    `json:"source,omitempty"`
	SourceOrganization string    `json:"source_organization,omitempty"`
	LastUpdated        time.Time `json:"last_updated,omitempty"`
	Questionnaires     []struct {
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
			Text            string `json:"text,omitempty"`
			Value           string `json:"value,omitempty"`
			Recommendations []struct {
				Keyword string `json:"keyword,omitempty"`
				Text    string `json:"text,omitempty"`
				Type    string `json:"type,omitempty"`
			} `json:"recommendations,omitempty"`
		} `json:"answers_selection,omitempty"`
		Multi bool `json:"multi,omitempty"`
	} `json:"questionnaires,omitempty"`
}

func (questionnaire *QuestionnaireData) validate() error {
	var err error
	switch {
	case questionnaire.Revision == 0:
		err = errors.New("revision number missing")
	case len(questionnaire.Questionnaires) == 0:
		err = errors.New("questionnaires measures missing")
	case strings.TrimSpace(questionnaire.Source) == "":
		err = errors.New("source is required")
	case strings.TrimSpace(questionnaire.SourceOrganization) == "":
		err = errors.New("source organization is required")
	}
	return err
}

func (questionnaireAPI *questionnaireAPI) GetQuestionnaire(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	revisionStr := r.URL.Query().Get("revision")
	if revisionStr == "" {
		revisionStr = "0"
	}
	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		http.Error(w, "failed to convert revision to number", http.StatusBadRequest)
		return
	}

	questionnaireAPI.mu.RLock()
	defer questionnaireAPI.mu.RUnlock()

	questionnaire, ok := questionnaireAPI.questionnaires[revision]
	if !ok {
		err = json.NewEncoder(w).Encode(questionnaireAPI.questionnaires[questionnaireAPI.revision])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	err = json.NewEncoder(w).Encode(questionnaire)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (questionnaireAPI *questionnaireAPI) UpdateQuestionnaire(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

	// Get new revision json
	bs, err := json.Marshal(newQuestionnaire)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add revision to database
	err = revisionManager.Add(&revision{
		Revision:      newQuestionnaire.Revision,
		ResourceGroup: questionnaireGroup,
		Data:          bs,
	})
	if err != nil {
		errMsg := fmt.Sprintf("failed to add revision to db: %v", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	w.Write([]byte("questionnaire scheduled for update"))
}
