package control

import (
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/graph"
	"github.com/justinsb/scaler/pkg/http"
	"github.com/justinsb/scaler/pkg/simulate"
)

type PolicyInfo struct {
	Policy *scalingpolicy.ScalingPolicy `json:"policy"`
	State  *http.Info                   `json:"state"`
}

var _ graph.Graphable = &PolicyState{}

func (s *PolicyState) ListGraphs() ([]*graph.Metadata, error) {
	var metadata []*graph.Metadata

	inputs := make(map[string]bool)
	for _, c := range s.policy.Spec.Containers {
		for _, r := range c.Resources.Limits {
			fn := r.Function
			if fn.Input != "" {
				inputs[fn.Input] = true
			}
		}
		for _, r := range c.Resources.Requests {
			fn := r.Function
			if fn.Input != "" {
				inputs[fn.Input] = true
			}
		}
	}

	for input := range inputs {
		{
			g := &graph.Metadata{}
			g.Key = input
			g.Builder = func() (*graph.Model, error) { return s.buildGraph(input) }
			metadata = append(metadata, g)
		}
	}

	return metadata, nil
}

func (s *PolicyState) buildGraph(factor string) (*graph.Model, error) {
	g := &graph.Model{}
	for x := 1; x < 100; x++ {
		//factors := make(map[string]float64)
		//factors[factor] = float64(x)
		//
		//g.XAxis.Label = factor
		//
		//baseTime := time.Now()
		//clock := clock.NewFakeClock(baseTime)
		//static := staticfactors.NewStaticFactors(clock, factors)
		//snapshot, err := static.Snapshot()
		//if err != nil {
		//	// Shouldn't happen...
		//	glog.Warningf("error taking snapshot of static factors: %v", err)
		//	continue
		//}
		//
		//podSpec, err := scaling.ComputeChanges(snapshot, &s.policy.Spec)
		//if err != nil {
		//	glog.Warningf("error computing resources: %v", err)
		//	continue
		//}
		//
		//graph.AddPodDataPoints(g, "", float64(x), podSpec, &graph.Series{})
		//
		//if s.policy.Spec.Smoothing.DelayScaleDown != nil {
		//	scaleDownPodSpec, err := scaling.ComputeResourcesShifted(snapshot, &s.policy.Spec, s.policy.Spec.Smoothing.DelayScaleDown)
		//	if err != nil {
		//		glog.Warningf("error computing shifted resources: %v", err)
		//		continue
		//	}
		//
		//	graph.AddPodDataPoints(g, "scaledown_", float64(x), scaleDownPodSpec, &graph.Series{})
		//}
	}

	return g, nil
}

func (s *PolicyState) Query() *PolicyInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	info := &PolicyInfo{
		Policy: s.policy,
		//State:  s.evaluator.Query(),
	}
	return info
}

var _ simulate.Simulatable = &PolicyState{}

func (s *PolicyState) ListSimulations() ([]*simulate.Metadata, error) {
	var metadata []*simulate.Metadata

	{
		g := &simulate.Metadata{}
		g.Key = "default"
		g.Builder = func() (*simulate.Run, error) {
			return RunSimulation(s.policy, s.options)
		}
		metadata = append(metadata, g)
	}

	return metadata, nil
}

type StateInfo struct {
	Policies map[string]*PolicyInfo `json:"policies"`
}

// Query returns the current state, for reporting e.g. via the /statz endpoint
func (c *State) Query() interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	info := &StateInfo{
		Policies: make(map[string]*PolicyInfo),
	}
	for k, v := range c.policies {
		info.Policies[k.String()] = v.Query()
	}
	return info
}

var _ graph.Graphable = &State{}

func (c *State) ListGraphs() ([]*graph.Metadata, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var metadata []*graph.Metadata

	for k, v := range c.policies {
		graphs, err := v.ListGraphs()
		if err != nil {
			return nil, err
		}
		for _, g := range graphs {
			g.Key = k.Namespace + "/" + k.Name + "/" + g.Key
			metadata = append(metadata, g)
		}
	}
	return metadata, nil
}

var _ simulate.Simulatable = &State{}

func (c *State) ListSimulations() ([]*simulate.Metadata, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var metadata []*simulate.Metadata
	for k, v := range c.policies {
		simulations, err := v.ListSimulations()
		if err != nil {
			return nil, err
		}
		for _, s := range simulations {
			s.Key = k.Namespace + "/" + k.Name + "/" + s.Key
			metadata = append(metadata, s)
		}
	}
	return metadata, nil
}
