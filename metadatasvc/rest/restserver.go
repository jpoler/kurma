package rest

import (
	"encoding/json"
	"net/http"

	"github.com/apcera/kurma/metadatasvc/backend"
	"github.com/gorilla/mux"
)

type server struct {
	router *mux.Router
	store  backend.Backend
}

// Server ...
type Server interface {
	Listen() error
}

// NewServer ...
func NewRestServer(store backend.Backend) Server {
	rootRouter := mux.NewRouter()

	s := &server{
		router: rootRouter,
		store:  store,
	}

	router := rootRouter.
		PathPrefix("/{token}/").
		Subrouter()

	pmr := router.PathPrefix("/acMetadata/v1/pod").Subrouter()
	pmr.HandleFunc("/annotations/", helloWorld)
	pmr.HandleFunc("/manifest", helloWorld)
	pmr.HandleFunc("/uuid", helloWorld)

	amr := router.PathPrefix("/acMetadata/v1/apps/{appName}/").Subrouter()
	amr.HandleFunc("/annotations/", s.appAnnotations)
	amr.HandleFunc("/image/manifest", s.appImageManifest)
	amr.HandleFunc("/image/id", s.appImageID)

	ier := pmr.PathPrefix("/hmac").Subrouter()
	ier.HandleFunc("/sign", helloWorld)
	ier.HandleFunc("/verify", helloWorld)

	return s
}

// Listen ...
func (rs *server) Listen() error {
	return http.ListenAndServe(":8080", rs.router)
}

func helloWorld(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	res.Write([]byte("Hello World!\n"))
	b, e := json.Marshal(vars)
	if e != nil {
		panic("Derp")
	}

	res.Write(b)
}

func (rs *server) appAnnotations(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]
	appName := vars["appName"]

	app := rs.store.GetAppDefinition()
}

func (rs *server) appImageManifest(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	res.Write([]byte("Hello World!\n"))
	b, e := json.Marshal(vars)
	if e != nil {
		panic("Derp")
	}

	res.Write(b)
}

func (rs *server) appImageID(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	res.Write([]byte("Hello World!\n"))
	b, e := json.Marshal(vars)
	if e != nil {
		panic("Derp")
	}

	res.Write(b)
}
