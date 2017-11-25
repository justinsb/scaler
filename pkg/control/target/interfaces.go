package target

import "k8s.io/api/core/v1"

type Interface interface {
	// Read gets the current state of the target
	Read(kind, namespace, name string) (*v1.PodSpec, error)

	// UpdateResources updates the target with new resource limits/requests
	UpdateResources(kind, namespace, name string, updated *v1.PodSpec, dryrun bool) error

	// ReadClusterState gets the current state of the cluster (summary statistics)
	ReadClusterState() (*ClusterStats, error)
}

type ClusterStats struct {
	NodeCount          int
	NodeSumAllocatable v1.ResourceList
}
