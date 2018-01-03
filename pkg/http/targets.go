package http

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
)

type Targets struct {
	state HasState
}

func (h *Targets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	info := h.state.Query()

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(info)
	if err != nil {
		glog.Warningf("error writing http response: %v", err)
	}
}
