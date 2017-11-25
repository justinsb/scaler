package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/pkg/graph"
	"github.com/justinsb/scaler/webapp/templates"
)

type UI struct {
	graphable graph.Graphable
}

func (u *UI) AddHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/ui/graph/", u.ServeGraphPage)
}

func (u *UI) ServeGraphPage(w http.ResponseWriter, r *http.Request) {
	tokens := strings.SplitN(strings.Trim(r.URL.Path, "/"), "/", 3)

	graphs, err := u.graphable.ListGraphs()
	if err != nil {
		internalError(w, r, err)
		return
	}
	if len(tokens) == 3 {
		key := tokens[2]
		var graph *graph.Metadata
		for _, g := range graphs {
			if g.Key == key {
				graph = g
				break
			}
		}
		if graph == nil {
			internalError(w, r, fmt.Errorf("graph not found"))
			return
		}

		data, err := graph.Builder()
		if err != nil {
			internalError(w, r, err)
			return
		}

		contents, err := templates.BuildGraphPage(data)
		w.Header().Set("Content-Type", "text/html")
		if err != nil {
			internalError(w, r, err)
			return
		}

		if _, err := w.Write(contents); err != nil {
			glog.Warningf("error writing http response: %v", err)
		}
		return
	}

	{
		contents, err := templates.BuildGraphListPage(graphs)
		w.Header().Set("Content-Type", "text/html")
		if err != nil {
			internalError(w, r, err)
			return
		}

		if _, err := w.Write(contents); err != nil {
			glog.Warningf("error writing http response: %v", err)
		}
		return
	}
}

func internalError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, fmt.Sprintf("Internal error %v", err), 500)
}
