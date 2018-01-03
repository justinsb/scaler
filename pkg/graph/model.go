package graph

import (
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Model struct {
	XAxis  Axis      `json:"xAxis"`
	YAxis  Axis      `json:"yAxis"`
	Series []*Series `json:"series"`
}

type BuilderFunction func() (*Model, error)

type Metadata struct {
	Key     string `json:"key"`
	Builder BuilderFunction
}

type Graphable interface {
	ListGraphs() ([]*Metadata, error)
}

func (g *Model) GetSeries(key string, options *Series) *Series {
	for _, s := range g.Series {
		if s.Key == key {
			return s
		}
	}
	s := &Series{}
	*s = *options
	s.Key = key
	g.Series = append(g.Series, s)
	return s
}

type Series struct {
	Key    string  `json:"key"`
	Units  string  `json:"units"`
	Values []Value `json:"values"`

	StrokeWidth float32 `json:"strokeWidth,omitempty"`
	Classed     string  `json:"classed,omitempty"`

	// Area will fill the area under the line
	Area bool `json:"area,omitempty"`
}

type Axis struct {
	Label string `json:"label"`
}

func (s *Series) AddXYPoint(x float64, y float64) {
	s.Values = append(s.Values, Value{
		X: x,
		Y: y,
	})
}

type Value struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func AddPodDataPoints(graph *Model, prefix string, x float64, podSpec *v1.PodSpec, options *Series) {
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]

		for k, q := range container.Resources.Limits {
			v, units := resourceToFloat(k, q)

			label := prefix + string(k) + "_limits_" + container.Name
			s := graph.GetSeries(label, options)
			s.AddXYPoint(x, v)
			s.Units = units
		}

		for k, q := range container.Resources.Requests {
			v, units := resourceToFloat(k, q)

			label := prefix + string(k) + "_requests_" + container.Name
			s := graph.GetSeries(label, options)
			s.AddXYPoint(x, v)
			s.Units = units
		}
	}
}

func resourceToFloat(k v1.ResourceName, q resource.Quantity) (float64, string) {
	var v float64
	var units string
	switch k {
	case v1.ResourceCPU:
		v = float64(q.MilliValue()) / 1000.0
		units = "CPU cores"
	case v1.ResourceMemory:
		v = float64(q.Value())
		units = "bytes"

	default:
		glog.Warningf("unhandled resource type in statz %s", k)
		v = float64(q.Value())
		units = ""
	}

	return v, units
}
