package scaling

import (
	"sync"

	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/factors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/clock"
)

type ScalingPolicyEvaluator struct {
	mutex sync.Mutex
	clock clock.Clock
	rule  *scalingpolicy.ScalingPolicy

	containers map[string]*containerScalingRuleEvaluator
}

func NewScalingPolicyEvaluator(clock clock.Clock, rule *scalingpolicy.ScalingPolicy) *ScalingPolicyEvaluator {
	e := &ScalingPolicyEvaluator{
		rule:       rule,
		clock:      clock,
		containers: make(map[string]*containerScalingRuleEvaluator),
	}

	e.UpdatePolicy(rule)

	return e
}

func (e *ScalingPolicyEvaluator) UpdatePolicy(rule *scalingpolicy.ScalingPolicy) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	marked := make(map[string]bool)
	for i := range rule.Spec.Containers {
		r := &rule.Spec.Containers[i]
		ce := e.containers[r.Name]
		if ce == nil {
			ce = newContainerScalingRuleEvaluator(r, e.clock)
			e.containers[r.Name] = ce
		} else {
			ce.updatePolicy(r)
		}
		marked[r.Name] = true
	}
	for k := range e.containers {
		if !marked[k] {
			delete(e.containers, k)
		}
	}
}

// ComputeResources computes a list of resource quantities based on the input state and the specified policy
// It returns a partial PodSpec with the resources we should apply
func (e *ScalingPolicyEvaluator) ComputeResources(parentPath string, currentPod *v1.PodSpec) (*v1.PodSpec, error) {
	pod := &v1.PodSpec{}

	for k, ce := range e.containers {
		var current *v1.Container
		for i := range currentPod.Containers {
			if currentPod.Containers[i].Name == k {
				current = &currentPod.Containers[i]
			}
		}

		c, err := ce.computeResources(parentPath+"["+k+"]", current)
		if err != nil {
			return nil, err
		}
		if c != nil {
			pod.Containers = append(pod.Containers, *c)
		}
	}

	if len(pod.Containers) == 0 {
		return nil, nil
	}

	return pod, nil
}

// AddObservation is called whenever we observe input values
func (e *ScalingPolicyEvaluator) AddObservation(inputs factors.Snapshot) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, ce := range e.containers {
		ce.addObservation(inputs)
	}
}
