package http

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/pkg/control"
)

type Targets struct {
	state *control.State
}

func (h *Targets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	info := h.state.Query()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(info)
	if err != nil {
		glog.Warningf("error writing http response: %v", err)
	}
}
