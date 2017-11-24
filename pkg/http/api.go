package http

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
)

type APIServer struct {
	server *http.Server
}

type HasState interface {
	Query() interface{}
}

func NewAPIServer(options *options.AutoScalerConfig, state HasState) (*APIServer, error) {
	mux := http.NewServeMux()

	//if *profiling {
	//	mux.HandleFunc("/debug/pprof/", pprof.Index)
	//	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	//	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	//}

	mux.Handle("/api/statz", &Targets{state: state})

	server := &http.Server{
		Addr:    options.ListenAPI,
		Handler: mux,
	}
	a := &APIServer{
		server: server,
	}
	return a, nil
}

func (s *APIServer) Start(stopCh <-chan struct{}) error {
	go func() {
		<-stopCh
		s.server.Close()
	}()

	glog.Infof("API listening on %s", s.server.Addr)
	err := s.server.ListenAndServe()
	if err != nil {
		return err
	}
	return nil
}
