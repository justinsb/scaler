package target

import (
	"fmt"

	"k8s.io/api/core/v1"
)

type SimulationTarget struct {
	Current *v1.PodSpec

	ClusterState *ClusterStats
}

var _ Interface = &SimulationTarget{}

func NewSimulationTarget() *SimulationTarget {
	return &SimulationTarget{}
}

func (s *SimulationTarget) Read(kind, namespace, name string) (*v1.PodSpec, error) {
	if s.Current == nil {
		return nil, fmt.Errorf("simulated value not set")
	}
	return s.Current, nil
}

func (s *SimulationTarget) UpdateResources(kind, namespace, name string, updates *v1.PodSpec, dryrun bool) error {
	s.Current = updates
	return nil
}

func (s *SimulationTarget) ReadClusterState() (*ClusterStats, error) {
	if s.ClusterState == nil {
		return nil, fmt.Errorf("simulated cluster state not set")
	}
	return s.ClusterState, nil
}
