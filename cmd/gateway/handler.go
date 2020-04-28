package main

import (
	"encoding/json"
	"github.com/gidyon/file-handlers/static"
	"github.com/gidyon/gateway"
	"github.com/pkg/errors"
	"net/http"
	"path/filepath"
)

func updateEndpoints(g *gateway.Gateway) {
	// Update health checks endpoints
	g.HandleFunc("/readyq", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := "am ready and running :)"
		w.Write([]byte(data))
	}))
	g.HandleFunc("/liveq", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := "am alive and running :)"
		w.Write([]byte(data))
	}))

	g.Handle("/services", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "\t")
		err := encoder.Encode(g.Services())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))

	g.Handle("/hello", http.HandlerFunc(helloHandler))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello"))
}

func updateStaticFilesHandler(g *gateway.Gateway) {
	handlerInfo, err := g.StaticHandlerInfo("StaticFiles")
	handleErr(err)

	notFound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(handlerInfo.RootDir(), handlerInfo.Index()))
	})

	handler, err := static.NewHandler(&static.ServerOptions{
		RootDir:         handlerInfo.RootDir(),
		Index:           handlerInfo.Index(),
		NotFoundHandler: notFound,
		URLPathPrefix:   handlerInfo.PathPrefix(),
		PushContent:     handlerInfo.PushFileInfoMap(),
		FallBackIndex:   handlerInfo.FallbackIndex,
	})
	handleErr(errors.Wrap(err, "failed to setup static file server"))

	g.Handle(handlerInfo.PathPrefix(), http.StripPrefix(handlerInfo.PathPrefix(), handler))
}

func updateAPIDocumentationHandler(g *gateway.Gateway) {
	handlerInfo, err := g.StaticHandlerInfo("Documentation")
	handleErr(err)

	handler, err := static.NewHandler(&static.ServerOptions{
		RootDir:         handlerInfo.RootDir(),
		Index:           handlerInfo.Index(),
		AllowedDirs:     handlerInfo.AllowedDirs(),
		NotFoundHandler: http.NotFoundHandler(),
		URLPathPrefix:   handlerInfo.PathPrefix(),
		FallBackIndex:   handlerInfo.FallbackIndex,
	})
	handleErr(errors.Wrap(err, "failed to setup API documentation"))

	g.Handle(handlerInfo.PathPrefix(), http.StripPrefix(handlerInfo.PathPrefix(), handler))
}
