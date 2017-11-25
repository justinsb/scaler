package graph

type Model struct {
	XAxis  Axis      `json:"xAxis"`
	YAxis  Axis      `json:"yAxis"`
	Series []*Series `json:"series"`
}

type Graphable interface {
	BuildGraph() (*Model, error)
}

func (g *Model) GetSeries(key string) *Series {
	for _, s := range g.Series {
		if s.Key == key {
			return s
		}
	}
	s := &Series{
		Key: key,
	}
	g.Series = append(g.Series, s)
	return s
}

type Series struct {
	Key    string  `json:"key"`
	Units  string  `json:"units"`
	Values []Value `json:"values"`
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
