package metadatasvc

import (
	"encoding/json"
	"net/http"

	backend "github.com/apcera/kurma/metadatasvc/backend"
	"github.com/gorilla/mux"
)

type restServer struct {
	router *mux.Router
}

// RestServer ...
type RestServer interface {
	Listen() error
}

// NewRestServer ...
func NewRestServer(backend.Backend) RestServer {
	rootRouter := mux.NewRouter()

	router := rootRouter.Headers("Metadata-Flavor", "AppContainer").
		PathPrefix("/{id}/").
		Subrouter()

	amr := router.PathPrefix("/acMetadata/v1/apps/{appName}/").Subrouter()
	amr.HandleFunc("/annotations/", helloWorld)
	amr.HandleFunc("/manifest", helloWorld)
	amr.HandleFunc("/uuid", helloWorld)

	pmr := router.PathPrefix("/acMetadata/v1/pod").Subrouter()
	pmr.HandleFunc("/annotations/", helloWorld)
	pmr.HandleFunc("/image/manifest", helloWorld)
	pmr.HandleFunc("/image/id", helloWorld)

	ier := pmr.PathPrefix("/hmac").Subrouter()
	ier.HandleFunc("/sign", helloWorld)
	ier.HandleFunc("/verify", helloWorld)

	return &restServer{
		router: rootRouter,
	}
}

// Listen ...
func (rs *restServer) Listen() error {
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
