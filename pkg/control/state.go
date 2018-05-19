package control

import (
	"sync"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/control/target"
	"github.com/justinsb/scaler/pkg/factors"
	k8sfactors "github.com/justinsb/scaler/pkg/factors/kubernetes"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
)

// State holds the current parent and state around applying them
type State struct {
	clock   clock.Clock
	target  target.Interface
	options *options.AutoScalerConfig
	factors factors.Interface

	mutex    sync.Mutex
	policies map[types.NamespacedName]*PolicyState
}

func NewState(clock clock.Clock, target target.Interface, options *options.AutoScalerConfig) (*State, error) {
	p := &State{
		clock:    clock,
		target:   target,
		options:  options,
		policies: make(map[types.NamespacedName]*PolicyState),
	}

	p.factors = k8sfactors.NewPollingKubernetesFactors(clock, target)

	return p, nil
}

func (c *State) Run(stopCh <-chan struct{}) {
	go wait.Until(func() {
		err := c.makeObservation()
		if err != nil {
			// TODO: Report as event
			glog.Warningf("error observing cluster values: %v", err)
		}
	}, c.options.PollPeriod, stopCh)

	go wait.Until(func() {
		err := c.applyPolicies()
		if err != nil {
			// TODO: Report as event
			glog.Warningf("error applying policy values: %v", err)
		}
	}, c.options.UpdatePeriod, stopCh)
}

func (c *State) remove(namespace, name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := types.NamespacedName{Namespace: namespace, Name: name}
	policyState := c.policies[key]
	if policyState != nil {
		delete(c.policies, key)
	}
}

func (c *State) upsert(o *scalingpolicy.ScalingPolicy) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// TODO: Should we invalidate the histogram for a fast response to policy changes

	key := types.NamespacedName{Namespace: o.Namespace, Name: o.Name}
	policyState := c.policies[key]
	if policyState == nil {
		policyState = NewPolicyState(c, o)
		c.policies[key] = policyState
	} else {
		policyState.updatePolicy(o)
	}
}

func (c *State) makeObservation() error {
	snapshot, err := c.factors.Snapshot()
	if err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, p := range c.policies {
		p.addObservation(snapshot)
	}

	return nil
}

func (c *State) applyPolicies() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for k, p := range c.policies {
		if err := p.updateValues(); err != nil {
			glog.Warningf("error updating target values for %s: %v", k, err)
			continue
		}
	}

	return nil
}
