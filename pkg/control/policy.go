package control

import (
	"fmt"
	"sync"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/control/target"
	"github.com/justinsb/scaler/pkg/factors"
	"github.com/justinsb/scaler/pkg/scaling/smoothing"
)

// PolicyState is the state around a single scaling policy
type PolicyState struct {
	target  target.Interface
	options *options.AutoScalerConfig

	mutex     sync.Mutex
	parent    *State
	policy    *scalingpolicy.ScalingPolicy
	smoothing smoothing.Smoothing
}

func NewPolicyState(parent *State, policy *scalingpolicy.ScalingPolicy) *PolicyState {
	s := &PolicyState{
		target:  parent.target,
		options: parent.options,
		parent:  parent,
		policy:  policy,
	}

	s.smoothing = smoothing.New(parent.clock, &policy.Spec.Smoothing)

	return s
}

func (s *PolicyState) updatePolicy(o *scalingpolicy.ScalingPolicy) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.policy = o
	s.smoothing = smoothing.UpdateRule(s.parent.clock, s.smoothing, &s.policy.Spec.Smoothing)
}

func (s *PolicyState) computeTargetValues(snapshot factors.Snapshot) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	policy := s.policy

	kind := policy.Spec.ScaleTargetRef.Kind
	namespace := policy.Namespace
	name := policy.Spec.ScaleTargetRef.Name

	path := fmt.Sprintf("%s/%s/%s", kind, namespace, name)

	glog.V(4).Infof("computing target values for %s", path)

	return s.smoothing.UpdateTarget(snapshot, &s.policy.Spec)
}

func (s *PolicyState) updateValues() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	policy := s.policy

	kind := policy.Spec.ScaleTargetRef.Kind
	namespace := policy.Namespace
	name := policy.Spec.ScaleTargetRef.Name

	path := fmt.Sprintf("%s/%s/%s", kind, namespace, name)

	spec, err := s.target.Read(kind, namespace, name)
	if err != nil {
		// TODO: Emit event?
		return err
	}

	changed, updates := s.smoothing.ComputeChange(path, spec)

	if changed {
		if err := s.target.UpdateResources(kind, namespace, name, updates, s.options.DryRun); err != nil {
			glog.Warningf("failed to update %q: %v", kind, err)
		} else {
			glog.V(4).Infof("applied update to %s", path)
		}
	} else {
		glog.V(4).Infof("no change needed for %s", path)
	}

	return nil
}
