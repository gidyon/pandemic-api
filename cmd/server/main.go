package main

import (
	"flag"
	"github.com/Sirupsen/logrus"
	"github.com/gidyon/fightcovid19/internal/app"
	"github.com/gidyon/fightcovid19/pkg/middleware"
	"github.com/gidyon/micros/pkg/conn"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	rootDir       = flag.String("root", "api/json", "Root directory")
	port          = flag.String("port", ":9090", "Server port")
	certFile      = flag.String("cert", "certs/localhost/cert.pem", "TLS certificate file")
	keyFile       = flag.String("key", "certs/localhost/key.pem", "TLS key file")
	mysqlHost     = flag.String("mysql-host", "localhost", "Mysql host")
	mysqlPort     = flag.String("mysql-port", "3306", "Mysql host")
	mysqUser      = flag.String("mysql-user", "root", "Mysql user")
	mysqlPassword = flag.String("mysql-password", "hakty11", "Mysql password")
	mysqlSchema   = flag.String("mysql-schema", "fightcovid19", "Mysql schema")
	env           = flag.Bool("env", false, "Assigns higher priority to env variables")
)

func main() {
	flag.Parse()

	const (
		contactsPreffix     = "contacts"
		faqPreffix          = "faqs"
		quarantinePreffix   = "quarantine"
		questionnairePrefix = "questionnaire"
		symptomsPreffix     = "symptoms"
	)

	*rootDir = setIfempty(*rootDir, os.Getenv("ROOT_DIR"), *env)
	*port = setIfempty(*port, os.Getenv("PORT"), *env)
	*certFile = setIfempty(*certFile, os.Getenv("TLS_CERT_FILE"), *env)
	*keyFile = setIfempty(*keyFile, os.Getenv("TLS_KEY_FILE"), *env)
	*mysqlHost = setIfempty(*mysqlHost, os.Getenv("MYSQL_HOST"), *env)
	*mysqlPort = setIfempty(*mysqlPort, os.Getenv("MYSQL_PORT"), *env)
	*mysqUser = setIfempty(*mysqUser, os.Getenv("MYSQL_USER"), *env)
	*mysqlPassword = setIfempty(*mysqlPassword, os.Getenv("MYSQL_PASSWORD"), *env)
	*mysqlSchema = setIfempty(*mysqlSchema, os.Getenv("MYSQL_SCHEMA"), *env)

	sqlDB, err := conn.ToSQLDBUsingORM(&conn.DBOptions{
		Dialect:  "mysql",
		Host:     *mysqlHost,
		Port:     *mysqlPort,
		User:     *mysqUser,
		Password: *mysqlPassword,
		Schema:   *mysqlSchema,
	})
	handleErr(err)

	router := httprouter.New()

	// Contacts API
	app.RegisterContactAPIRouter(router, &app.Options{
		RootDir:    filepath.Join(*rootDir, contactsPreffix),
		FilePrefix: contactsPreffix,
		Revision:   1,
	})

	// FAQ API
	app.RegisterFAQAPIRouter(router, &app.Options{
		RootDir:    filepath.Join(*rootDir, faqPreffix),
		FilePrefix: faqPreffix,
		Revision:   1,
	})

	// SelfQuarantine API
	app.RegisterQuarantineAPIRouter(router, &app.Options{
		RootDir:    filepath.Join(*rootDir, quarantinePreffix),
		FilePrefix: quarantinePreffix,
		Revision:   1,
	})

	// Questionnaire API
	app.RegisterQuestionnaireAPIRouter(router, &app.Options{
		RootDir:    filepath.Join(*rootDir, questionnairePrefix),
		FilePrefix: questionnairePrefix,
		Revision:   1,
	})

	// Report API
	app.RegisterReportAPIRouter(router, sqlDB)

	// Symptoms API
	app.RegisterSymptomsAPIRouter(router, &app.Options{
		RootDir:    filepath.Join(*rootDir, symptomsPreffix),
		FilePrefix: symptomsPreffix,
		Revision:   1,
	})

	// Global middlewares
	handler := middleware.Apply(router, middleware.SupportCORS, middleware.SetJSONCtype)

	parsedPort := ":" + strings.TrimPrefix(*port, ":")
	logrus.Infof("server started on port %s\n", parsedPort)
	logrus.Fatalln(http.ListenAndServeTLS(parsedPort, *certFile, *keyFile, handler))
}

func setIfempty(val1, val2 string, swap ...bool) string {
	if len(swap) > 0 && swap[0] {
		if strings.TrimSpace(val2) == "" {
			return val1
		}
		return val2
	}
	if strings.TrimSpace(val1) == "" {
		return val2
	}
	return val1
}

func handleErr(err error) {
	if err != nil {
		logrus.Fatalln(err)
	}
}
