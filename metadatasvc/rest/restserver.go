package rest

import (
	"encoding/json"
	"net/http"

	"github.com/apcera/kurma/metadatasvc/backend"
	"github.com/gorilla/mux"

	"github.com/appc/spec/schema/types"
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
	pmr.HandleFunc("/annotations/", s.podAnnotations)
	pmr.HandleFunc("/manifest", s.podManifest)
	pmr.HandleFunc("/uuid", s.podUUID)

	amr := router.PathPrefix("/acMetadata/v1/apps/{appName}/").Subrouter()
	amr.HandleFunc("/annotations/", s.appAnnotations)
	amr.HandleFunc("/image/manifest", s.appImageManifest)
	amr.HandleFunc("/image/id", s.appImageID)

	ier := pmr.PathPrefix("/hmac").Methods("POST").Subrouter()
	ier.HandleFunc("/sign", s.sign)
	ier.HandleFunc("/verify", s.verify)

	return s
}

// Listen ...
func (rs *server) Listen() error {
	return http.ListenAndServe(":8080", rs.router)
}

func (rs *server) podAnnotations(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	pod := rs.store.GetPod(token)
	if pod == nil {
		http.Error(res, "Unable to find pod", http.StatusNotFound)
		return
	}

	b, err := json.Marshal(pod.Annotations)
	if err != nil {
		http.Error(res, "Unable to do stuff.", http.StatusInternalServerError)
	}

	res.Write(b)

}

func (rs *server) podManifest(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	pod := rs.store.GetPod(token)
	if pod == nil {
		http.Error(res, "Unable to find pod", http.StatusNotFound)
		return
	}

	b, err := pod.MarshalJSON()
	if err != nil {
		http.Error(res, "Unable to do stuff.", http.StatusInternalServerError)
	}

	res.Write(b)
}

func (rs *server) podUUID(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	podUUID, err := rs.store.GetPodUUID(token)
	if err != nil {
		http.Error(res, "Unable to find pod", http.StatusNotFound)
		return
	}

	res.Write([]byte(podUUID))
}

func (rs *server) appAnnotations(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]
	appName := vars["appName"]

	pod := rs.store.GetPod(token)
	if pod == nil {
		http.Error(res, "Unable to find pod", http.StatusNotFound)
		return
	}

	app := pod.Apps.Get(types.ACName(appName))
	if app == nil {
		http.Error(res, "Unable to find app", http.StatusNotFound)
		return
	}

	b, err := json.Marshal(app.Annotations)
	if err != nil {
		http.Error(res, "Unable to do stuff.", http.StatusInternalServerError)
	}

	res.Write(b)

}

func (rs *server) appImageManifest(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]
	appName := vars["appName"]

	pod := rs.store.GetPod(token)
	if pod == nil {
		http.Error(res, "Unable to find pod", http.StatusNotFound)
		return
	}

	app := pod.Apps.Get(types.ACName(appName))
	if app == nil {
		http.Error(res, "Unable to find app", http.StatusNotFound)
		return
	}

	b, err := json.Marshal(app.Image)
	if err != nil {
		http.Error(res, "Unable to do stuff.", http.StatusInternalServerError)
	}

	res.Write(b)
}

func (rs *server) appImageID(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]
	appName := vars["appName"]

	pod := rs.store.GetPod(token)
	if pod == nil {
		http.Error(res, "Unable to find pod", http.StatusNotFound)
		return
	}

	app := pod.Apps.Get(types.ACName(appName))
	if app == nil {
		http.Error(res, "Unable to find app", http.StatusNotFound)
		return
	}

	res.Write([]byte(app.Image.ID.String()))
}

func (rs *server) sign(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	if err := req.ParseForm(); err != nil {
		http.Error(res, "Invalid form", http.StatusBadRequest)
	}

	content := req.PostForm.Get("content")
	if content == "" {
		http.Error(res, "No content", http.StatusNoContent)
		return
	}

	signature, err := rs.store.Sign(token, content)
	if err != nil {
		http.Error(res, "Error", http.StatusInternalServerError)
		return
	}
	res.Write([]byte(signature))
}

func (rs *server) verify(res http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		http.Error(res, "Invalid form", http.StatusBadRequest)
	}

	content := req.PostForm.Get("content")

	if content == "" {
		http.Error(res, "No content", http.StatusBadRequest)
	}
	signature := req.PostForm.Get("signature")

	if signature == "" {
		http.Error(res, "No signature", http.StatusBadRequest)
	}

	uuid := req.PostForm.Get("uuid")
	if uuid == "" {
		http.Error(res, "No UUID", http.StatusBadRequest)
	}

	err := rs.store.Verify(content, signature, uuid)
	if err != nil {
		http.Error(res, "Invalid signature", http.StatusForbidden)
	}
}
