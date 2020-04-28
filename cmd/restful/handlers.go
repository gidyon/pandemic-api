package main

import (
	"os"
	"path/filepath"

	"github.com/gidyon/micros"
	"github.com/gidyon/pandemic-api/internal/rest"
	"github.com/gidyon/pandemic-api/pkg/middleware"
	"github.com/julienschmidt/httprouter"
)

const (
	contactsPreffix     = "contacts"
	faqPreffix          = "faqs"
	quarantinePreffix   = "quarantine"
	questionnairePrefix = "questionnaire"
	symptomsPreffix     = "symptoms"
)

var rootDir = "json"

func registerHandlers(srv *micros.Service) {
	rootDir = setIfEmpty(os.Getenv("ROOT_DIR"), rootDir)
	router := httprouter.New()

	// Start revisions manager
	rest.StartRevisionManager(srv.GormDB())

	// Account API
	rest.RegisterAccountAPI(router, srv.GormDB())

	// Confirmed cases API
	rest.RegisterConfirmedCasesAPI(router, srv.GormDB())

	// Contacts API
	rest.RegisterContactAPI(router, &rest.Options{
		RootDir:    filepath.Join(rootDir, contactsPreffix),
		FilePrefix: contactsPreffix,
		Revision:   1,
	})

	// FAQ API
	rest.RegisterFAQAPI(router, &rest.Options{
		RootDir:    filepath.Join(rootDir, faqPreffix),
		FilePrefix: faqPreffix,
		Revision:   1,
	})

	// SelfQuarantine API
	rest.RegisterQuarantineMeasuresAPI(router, &rest.Options{
		RootDir:    filepath.Join(rootDir, quarantinePreffix),
		FilePrefix: quarantinePreffix,
		Revision:   1,
	})

	// Questionnaire API
	rest.RegisterQuestionnairesAPI(router, &rest.Options{
		RootDir:    filepath.Join(rootDir, questionnairePrefix),
		FilePrefix: questionnairePrefix,
		Revision:   2,
	})

	// Report API
	rest.RegisterReportedCasesAPI(router, srv.GormDB())

	// Suspected cases API
	rest.RegisterQuestionnaireAPI(router, srv.GormDB())

	// Symptoms API
	rest.RegisterPandemicSymptomsAPI(router, &rest.Options{
		RootDir:    filepath.Join(rootDir, symptomsPreffix),
		FilePrefix: symptomsPreffix,
		Revision:   1,
	})

	// Global middlewares
	handler := middleware.Apply(router, middleware.SupportCORS, middleware.SetJSONCtype)

	srv.AddEndpoint("/rest/v1/", handler)
}
