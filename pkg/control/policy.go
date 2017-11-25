package control

import (
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/factors"
	staticfactors "github.com/justinsb/scaler/pkg/factors/static"
	"github.com/justinsb/scaler/pkg/graph"
	"github.com/justinsb/scaler/pkg/http"
	"github.com/justinsb/scaler/pkg/scaling"
	"github.com/justinsb/scaler/pkg/scaling/smoothing"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// PolicyState is the state around a single scaling policy
type PolicyState struct {
	kubeClient kubernetes.Interface
	options    *options.AutoScalerConfig

	mutex     sync.Mutex
	parent    *State
	policy    *scalingpolicy.ScalingPolicy
	smoothing smoothing.Smoothing
}

func NewPolicyState(parent *State, policy *scalingpolicy.ScalingPolicy) *PolicyState {
	s := &PolicyState{
		kubeClient: parent.client,
		options:    parent.options,
		parent:     parent,
		policy:     policy,
	}

	s.smoothing = smoothing.New(&policy.Spec.Smoothing)

	return s
}

func (s *PolicyState) updatePolicy(o *scalingpolicy.ScalingPolicy) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.policy = o
	s.smoothing = smoothing.UpdateRule(s.smoothing, &s.policy.Spec.Smoothing)
}

func (s *PolicyState) computeTargetValues(snapshot factors.Snapshot) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	glog.V(4).Infof("computing target values")

	return s.smoothing.UpdateTarget(snapshot, &s.policy.Spec)
}

func (s *PolicyState) updateValues() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	policy := s.policy
	client := s.kubeClient
	patcher := s.parent.patcher

	kind := policy.Spec.ScaleTargetRef.Kind
	namespace := policy.Namespace
	name := policy.Spec.ScaleTargetRef.Name

	path := fmt.Sprintf("%s/%s/%s", kind, namespace, name)

	changed := false
	var updates *v1.PodSpec
	switch strings.ToLower(kind) {
	case "replicaset":
		{
			kind = "ReplicaSet"
			o, err := client.ExtensionsV1beta1().ReplicaSets(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				// TODO: Emit event?
				return err
			}

			changed, updates = s.smoothing.ComputeChange(path, &o.Spec.Template.Spec)
		}

	case "daemonset":
		{
			kind = "DaemonSet"
			o, err := client.ExtensionsV1beta1().DaemonSets(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				// TODO: Emit event?
				return err
			}

			changed, updates = s.smoothing.ComputeChange(path, &o.Spec.Template.Spec)
		}

	case "deployment":
		{
			kind = "Deployment"
			o, err := client.AppsV1beta1().Deployments(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				// TODO: Emit event?
				return err
			}

			changed, updates = s.smoothing.ComputeChange(path, &o.Spec.Template.Spec)
		}

	default:
		return fmt.Errorf("unhandled kind: %q", kind)
	}

	if changed {
		if err := patcher.UpdateResources(kind, namespace, name, updates, s.options.DryRun); err != nil {
			glog.Warningf("failed to update %q: %v", kind, err)
		} else {
			glog.V(4).Infof("applied update to %s", path)
		}
	} else {
		glog.V(4).Infof("no change needed for %s", path)
	}

	return nil
}

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
			if r.Input != "" {
				inputs[r.Input] = true
			}
		}
		for _, r := range c.Resources.Requests {
			if r.Input != "" {
				inputs[r.Input] = true
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
	graph := &graph.Model{}
	for x := 1; x < 100; x++ {
		factors := make(map[string]float64)
		factors[factor] = float64(x)

		graph.XAxis.Label = factor

		static := staticfactors.NewStaticFactors(factors)
		snapshot, err := static.Snapshot()
		if err != nil {
			// Shouldn't happen...
			glog.Warningf("error taking snapshot of static factors: %v", err)
			continue
		}

		podSpec, err := scaling.ComputeResources(snapshot, &s.policy.Spec)
		if err != nil {
			glog.Warningf("error computing resources: %v", err)
			continue
		}

		addDataPoints(graph, "", float64(x), podSpec)

		if s.policy.Spec.Smoothing.ScaleDownShift != nil {
			scaleDownPodSpec, err := scaling.ComputeResourcesShifted(snapshot, &s.policy.Spec, s.policy.Spec.Smoothing.ScaleDownShift)
			if err != nil {
				glog.Warningf("error computing shifted resources: %v", err)
				continue
			}

			addDataPoints(graph, "scaledown_", float64(x), scaleDownPodSpec)
		}
	}

	return graph, nil
}

func addDataPoints(graph *graph.Model, prefix string, x float64, podSpec *v1.PodSpec) {
	for i := range podSpec.Containers {
		container := &podSpec.Containers[i]

		for k, q := range container.Resources.Limits {
			v, units := resourceToFloat(k, q)

			label := prefix + string(k) + "_limits_" + container.Name
			s := graph.GetSeries(label)
			s.AddXYPoint(x, v)
			s.Units = units
		}

		for k, q := range container.Resources.Requests {
			v, units := resourceToFloat(k, q)

			label := prefix + string(k) + "_requests_" + container.Name
			s := graph.GetSeries(label)
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

func (s *PolicyState) Query() *PolicyInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	info := &PolicyInfo{
		Policy: s.policy,
		State:  s.smoothing.Query(),
	}
	return info
}
