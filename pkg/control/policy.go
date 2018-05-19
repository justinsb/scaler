package control

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/control/target"
	"github.com/justinsb/scaler/pkg/factors"
	"github.com/justinsb/scaler/pkg/scaling"
)

// PolicyState is the state around a single scaling policy
type PolicyState struct {
	target  target.Interface
	options *options.AutoScalerConfig

	mutex  sync.Mutex
	parent *State
	policy *scalingpolicy.ScalingPolicy

	evaluator *scaling.ScalingPolicyEvaluator
}

func NewPolicyState(parent *State, policy *scalingpolicy.ScalingPolicy) *PolicyState {
	s := &PolicyState{
		target:  parent.target,
		options: parent.options,
		parent:  parent,
		policy:  policy,
	}

	s.evaluator = scaling.NewScalingPolicyEvaluator(parent.clock, policy)

	return s
}

func (s *PolicyState) updatePolicy(o *scalingpolicy.ScalingPolicy) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.policy = o
	s.evaluator.UpdatePolicy(o)
}

// addObservation is called whenever we observe a set of input values
func (s *PolicyState) addObservation(snapshot factors.Snapshot) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	policy := s.policy

	kind := policy.Spec.ScaleTargetRef.Kind
	namespace := policy.Namespace
	name := policy.Spec.ScaleTargetRef.Name

	path := fmt.Sprintf("%s/%s/%s", kind, namespace, name)

	glog.V(4).Infof("adding observation for %s", path)

	s.evaluator.AddObservation(snapshot)
}

func (s *PolicyState) updateValues() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	policy := s.policy

	kind := policy.Spec.ScaleTargetRef.Kind
	namespace := policy.Namespace
	name := policy.Spec.ScaleTargetRef.Name

	path := fmt.Sprintf("%s/%s/%s", kind, namespace, name)

	actual, err := s.target.Read(kind, namespace, name)
	if err != nil {
		// TODO: Emit event?
		return err
	}

	changes, err := s.evaluator.ComputeResources(path, actual)
	if err != nil {
		return err
	}

	if changes != nil {
		if err := s.target.UpdateResources(kind, namespace, name, changes, s.options.DryRun); err != nil {
			glog.Warningf("failed to update %q: %v", kind, err)
		} else {
			glog.V(4).Infof("applied update to %s", path)
		}
	} else {
		glog.V(4).Infof("no change needed for %s", path)
	}

	return nil
}
