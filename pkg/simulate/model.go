package simulate

import (
	"github.com/justinsb/scaler/pkg/control/target"
	"github.com/justinsb/scaler/pkg/graph"
	"k8s.io/api/core/v1"
)

type Metadata struct {
	Key     string `json:"key"`
	Builder BuilderFunction
}

type Simulatable interface {
	ListSimulations() ([]*Metadata, error)
}

type BuilderFunction func() (*Run, error)

type Run struct {
	Graph *graph.Model
}

func (r *Run) Add(t int, clusterState *target.ClusterStats, actual *v1.PodSpec) {
	if r.Graph == nil {
		r.Graph = &graph.Model{}
	}

	x := float64(t)

	if clusterState != nil {
		{
			v := clusterState.NodeSumAllocatable[v1.ResourceCPU]
			r.Graph.GetSeries("cluster-cores").AddXYPoint(x, float64(v.MilliValue())/1000.0)
		}
		{
			v := clusterState.NodeSumAllocatable[v1.ResourceMemory]
			r.Graph.GetSeries("cluster-mb").AddXYPoint(x, float64(v.Value())/(1024.0*1024.0*1024.0))
		}
		r.Graph.GetSeries("cluster-node-count").AddXYPoint(x, float64(clusterState.NodeCount))
	}

	if actual != nil {
		graph.AddPodDataPoints(r.Graph, "", x, actual)
	}
}
