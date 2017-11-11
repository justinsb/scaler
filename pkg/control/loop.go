package control

import (
	"sync"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/control/k8sclient"
	"github.com/justinsb/scaler/pkg/factors"
	k8sfactors "github.com/justinsb/scaler/pkg/factors/kubernetes"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type policies struct {
	client  kubernetes.Interface
	patcher k8sclient.ResourcePatcher
	options *options.AutoScalerConfig
	factors factors.Interface

	mutex    sync.Mutex
	policies map[types.NamespacedName]*PolicyState
}

func newPolicies(client kubernetes.Interface, options *options.AutoScalerConfig) (*policies, error) {
	p := &policies{
		client:   client,
		options:  options,
		policies: make(map[types.NamespacedName]*PolicyState),
	}

	var err error
	p.patcher, err = k8sclient.NewKubernetesPatcher(client)
	if err != nil {
		return nil, err
	}

	p.factors = k8sfactors.NewPollingKubernetesFactors(client)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (c *policies) Run(stopCh <-chan struct{}) {
	go wait.Until(func() {
		err := c.computeTargetValues()
		if err != nil {
			// TODO: Report as event
			glog.Warningf("error computing target values: %v", err)
		}
	}, c.options.PollPeriod, stopCh)

	go wait.Until(func() {
		err := c.updateValues()
		if err != nil {
			// TODO: Report as event
			glog.Warningf("error computing target values: %v", err)
		}
	}, c.options.PollPeriod, stopCh)
}

func (c *policies) remove(namespace, name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := types.NamespacedName{Namespace: namespace, Name: name}
	policyState := c.policies[key]
	if policyState != nil {
		delete(c.policies, key)
	}
}

func (c *policies) upsert(o *scalingpolicy.ScalingPolicy) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	key := types.NamespacedName{Namespace: o.Namespace, Name: o.Name}
	policyState := c.policies[key]
	if policyState == nil {
		policyState = NewPolicyState(c, o)
		c.policies[key] = policyState
	}
}

func (c *policies) computeTargetValues() error {
	snapshot, err := c.factors.Snapshot()
	if err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for k, p := range c.policies {
		if err := p.computeTargetValues(snapshot); err != nil {
			glog.Warningf("error computing target values for %s: %v", k, err)
			continue
		}
	}

	return nil
}

func (c *policies) updateValues() error {
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