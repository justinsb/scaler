package kubernetes

import (
	"sync"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/pkg/control/target"
	"github.com/justinsb/scaler/pkg/factors"
	v1 "k8s.io/api/core/v1"
)

type pollingKubernetesFactors struct {
	target target.Interface
}

var _ factors.Interface = &pollingKubernetesFactors{}

type pollingKubernetesSnapshot struct {
	target target.Interface

	mutex sync.Mutex
	stats *target.ClusterStats
}

var _ factors.Snapshot = &pollingKubernetesSnapshot{}

func NewPollingKubernetesFactors(target target.Interface) factors.Interface {
	return &pollingKubernetesFactors{
		target: target,
	}
}

func (k *pollingKubernetesFactors) Snapshot() (factors.Snapshot, error) {
	glog.V(4).Infof("querying kubernetes for cluster metrics")
	s := &pollingKubernetesSnapshot{
		target: k.target,
	}
	if err := s.ensureClusterStats(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *pollingKubernetesSnapshot) Get(key string) (float64, bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch key {
	// TODO: Syntax here is not very consistent e.g. sum(nodes.allocatable.cpu) or count(nodes)
	case "cores":
		{
			if err := s.ensureClusterStats(); err != nil {
				return 0, true, err
			}
			r, found := s.stats.NodeSumAllocatable[v1.ResourceCPU]
			if found {
				return float64(r.Value()), true, nil
			} else {
				// Return found=true: We recognized the value, even though we didn't have any statistics on it
				// TODO: Is this correct?
				return 0, true, nil
			}
		}
	case "memory":
		{
			if err := s.ensureClusterStats(); err != nil {
				return 0, true, err
			}
			r, found := s.stats.NodeSumAllocatable[v1.ResourceMemory]
			if found {
				return float64(r.Value()), true, nil
			} else {
				// Return found=true: We recognized the value, even though we didn't have any statistics on it
				// TODO: Is this correct?
				return 0, true, nil
			}
		}
	case "nodes":
		{
			if err := s.ensureClusterStats(); err != nil {
				return 0, true, err
			}
			return float64(s.stats.NodeCount), true, nil
		}
	default:
		// unknown
		return 0, false, nil
	}
}

func (s *pollingKubernetesSnapshot) ensureClusterStats() error {
	if s.stats != nil {
		return nil
	}

	stats, err := s.target.ReadClusterState()
	if err != nil {
		return err
	}
	s.stats = stats
	return nil
}
