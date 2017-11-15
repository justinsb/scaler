package http

import "k8s.io/api/core/v1"

type Info struct {
	LatestTarget *v1.PodSpec `json:"latestTarget"`
	LatestActual *v1.PodSpec `json:"latestActual"`

	Histograms map[string]*HistogramInfo `json:"histograms"`
}

type HistogramInfo struct {
	Data []HistogramDataPoint `json:"data"`
}

type HistogramDataPoint struct {
	Time  int64 `json:"time"`
	Value int64 `json:"value"`
}
