package http

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/webapp/templates"
	"github.com/justinsb/scaler/pkg/graph"
)

type UI struct {
	graphable graph.Graphable
}

func (u *UI) AddHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/ui/graph", u.ServeGraphPage)
}

func (u *UI) ServeGraphPage(w http.ResponseWriter, r *http.Request) {
	graph, err := u.graphable.BuildGraph()
	if err != nil {
		internalError(w, r, err)
		return
	}

	contents, err := templates.BuildGraphPage(graph)
	w.Header().Set("Content-Type", "text/html")
	if err != nil {
		internalError(w, r, err)
		return
	}

	if _, err := w.Write(contents); err != nil {
		glog.Warningf("error writing http response: %v", err)
	}
}

func internalError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, fmt.Sprintf("Internal error %v", err), 500)
}