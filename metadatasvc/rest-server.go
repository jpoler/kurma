package metadatasvc

import (
	"net/http"

	"github.com/gorilla/mux"
)

type restServer struct {
	router *mux.Router
}

// RestServer ...
type RestServer interface {
	Listen()
}

// NewRestServer ...
func NewRestServer() RestServer {
	rs := &restServer{
		router: mux.NewRouter(),
	}

	rs.router.Headers("Metadata-Flavor", "AppContainer")

	return rs
}

// Listen ...
func (rs *restServer) Listen() {
	http.ListenAndServe(":8080", rs.router)
}
