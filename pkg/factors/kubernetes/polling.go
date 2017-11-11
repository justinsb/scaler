package kubernetes

import (
	"github.com/justinsb/scaler/pkg/factors"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"sync"
)

type pollingKubernetesFactors struct {
	client kubernetes.Interface
}

var _ factors.Interface = &pollingKubernetesFactors{}

type pollingKubernetesSnapshot struct {
	client kubernetes.Interface

	mutex              sync.Mutex
	nodeSumAllocatable v1.ResourceList
	nodeCount          int
}

var _ factors.Snapshot = &pollingKubernetesSnapshot{}

func NewPollingKubernetesFactors(client kubernetes.Interface) factors.Interface {
	p := &pollingKubernetesFactors{
		client: client,
	}
	return p
}

func (k *pollingKubernetesFactors) Snapshot() (factors.Snapshot, error) {
	return &pollingKubernetesSnapshot{
		client: k.client,
	}, nil
}

func (s *pollingKubernetesSnapshot) Get(key string) (float64, bool, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch key {
	// TODO: Syntax here is not very consistent e.g. sum(nodes.allocatable.cpu) or count(nodes)
	case "cores":
		{
			if err := s.ensureNodeStats(); err != nil {
				return 0, true, err
			}
			r, found := s.nodeSumAllocatable[v1.ResourceCPU]
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
			if err := s.ensureNodeStats(); err != nil {
				return 0, true, err
			}
			r, found := s.nodeSumAllocatable[v1.ResourceMemory]
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
			if err := s.ensureNodeStats(); err != nil {
				return 0, true, err
			}
			return float64(s.nodeCount), true, nil
		}
	default:
		// unknown
		return 0, false, nil
	}
}

func (s *pollingKubernetesSnapshot) ensureNodeStats() (error) {
	if s.nodeCount != 0 {
		return nil
	}

	nodes, err := s.client.CoreV1().Nodes().List(meta_v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing nodes: %v", err)
	}

	allocatable := make(v1.ResourceList)
	nodeCount := 0
	for i := range nodes.Items {
		node := &nodes.Items[i]

		nodeCount++
		addResourceList(allocatable, node.Status.Allocatable)
	}

	s.nodeCount = nodeCount
	s.nodeSumAllocatable = allocatable

	return nil
}

func addResourceList(sum v1.ResourceList, inc v1.ResourceList) {
	for k, v := range inc {
		a, found := sum[k]
		if !found {
			sum[k] = v
		} else {
			a.Add(v)
			sum[k] = a
		}
	}
}
