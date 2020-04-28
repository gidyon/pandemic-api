package main

import (
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gidyon/gateway"
	http_middleware "github.com/gidyon/micros/pkg/http"
	"net/http"
	"os"
	"strings"
)

const (
	certPath      = "/home/gideon/go/src/github.com/gidyon/mcfp/certs/blockchain/cert.pem"
	keyPath       = "/home/gideon/go/src/github.com/gidyon/mcfp/certs/blockchain/key.pem"
	gatewayConfig = "/home/gideon/go/src/github.com/gidyon/pandemic-api/cmd/gateway/gateway.yml"
	pushFilesPath = "/home/gideon/go/src/github.com/gidyon/mcfp/configs/pushfiles.yml"
)

var (
	servicesFile = flag.String("services-file", gatewayConfig, "File containing list of to services")
	certFile     = flag.String("cert", certPath, "Path to public key")
	keyFile      = flag.String("key", keyPath, "Path to tls key")
	port         = flag.String("port", ":443", "Port to serve files")
	insecure     = flag.Bool("insecure", false, "Whether to use insecure http")
	cors         = flag.Bool("cors", false, "Whether to run app with CORS enabled")
	env          = flag.Bool("env", false, "Read parameters from env variables")
)

func setFromGateway(g *gateway.Gateway) {
	*certFile = setIfEmpty(g.TLSCertFile(), *certFile)
	*keyFile = setIfEmpty(g.TLSKeyFile(), *keyFile)
	*port = setIfEmpty(fmt.Sprintf("%d", g.Port()), *port)
}

func setFromEnv() {
	*servicesFile = setIfEmpty(strings.TrimSpace(os.Getenv("SERVICES_FILE")), *servicesFile)
	*certFile = setIfEmpty(os.Getenv("TLS_CERT_FILE"), *certFile)
	*keyFile = setIfEmpty(os.Getenv("TLS_KEY_FILE"), *keyFile)
	*port = setIfEmpty(os.Getenv("SERVICE_PORT"), *port)
}

func main() {
	flag.Parse()

	// Create gateway
	g, err := gateway.NewFromFile(*servicesFile)
	handleErr(err)

	// Set from file
	setFromGateway(g)

	// Set from env
	if *env {
		// Set from environemnt variables
		setFromEnv()
	}

	// Update documentation handler
	updateAPIDocumentationHandler(g)

	// Static file server
	updateStaticFilesHandler(g)

	// Update endpoints
	updateEndpoints(g)

	if *cors {
		// Enable CORS
		g.AddMiddlewares(http_middleware.SupportCORS)
	}

	handler := g.Handler()

	*port = ":" + strings.TrimPrefix(*port, ":")

	logrus.Infof("started gateway on port: %s", *port)

	if *insecure {
		logrus.Fatalln(http.ListenAndServe(*port, handler))
	} else {
		logrus.Fatalln(http.ListenAndServeTLS(*port, *certFile, *keyFile, handler))
	}
}

func setIfEmpty(val, def string) string {
	if val == "" && def == "" {
		return ""
	}
	if val == "" {
		return def
	}
	return val
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}
