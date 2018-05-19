package scaling

import (
	"sync"

	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/factors"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/clock"
)

type containerScalingRuleEvaluator struct {
	mutex sync.Mutex
	clock clock.Clock
	rule  *scalingpolicy.ContainerScalingRule

	limits   map[v1.ResourceName]*resourceScalingRuleEvaluator
	requests map[v1.ResourceName]*resourceScalingRuleEvaluator
}

func newContainerScalingRuleEvaluator(rule *scalingpolicy.ContainerScalingRule, clock clock.Clock) *containerScalingRuleEvaluator {
	e := &containerScalingRuleEvaluator{
		rule:     rule,
		clock:    clock,
		limits:   make(map[v1.ResourceName]*resourceScalingRuleEvaluator),
		requests: make(map[v1.ResourceName]*resourceScalingRuleEvaluator),
	}

	e.updatePolicy(rule)

	return e
}

func (e *containerScalingRuleEvaluator) updatePolicy(rule *scalingpolicy.ContainerScalingRule) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.updateResourceMap(rule.Resources.Limits, e.limits)
	e.updateResourceMap(rule.Resources.Requests, e.requests)
}

func (e *containerScalingRuleEvaluator) updateResourceMap(rules []scalingpolicy.ResourceScalingRule, evaluators map[v1.ResourceName]*resourceScalingRuleEvaluator) {
	marked := make(map[v1.ResourceName]bool)
	for i := range rules {
		r := &rules[i]
		re := evaluators[r.Resource]
		if re == nil {
			re = &resourceScalingRuleEvaluator{clock: e.clock}
			evaluators[r.Resource] = re
		}
		re.updatePolicy(r)
		marked[r.Resource] = true
	}
	for k := range evaluators {
		if !marked[k] {
			delete(evaluators, k)
		}
	}
}

// ComputeResources computes a list of resource quantities based on the input state and the specified policy
// It returns a partial PodSpec with the resources we should apply
func (e *containerScalingRuleEvaluator) computeResources(parentPath string, currentParent *v1.Container) (*v1.Container, error) {
	container := &v1.Container{
		Name: e.rule.Name,
	}

	if currentParent == nil {
		currentParent = &v1.Container{}
	}

	for k, re := range e.limits {
		current := currentParent.Resources.Limits[k]
		r, err := re.computeResources(parentPath+".limits."+string(k), current)
		if err != nil {
			return nil, err
		}
		if r == nil {
			continue
		}
		if container.Resources.Limits == nil {
			container.Resources.Limits = make(v1.ResourceList)
		}
		container.Resources.Limits[k] = *r
	}

	for k, re := range e.requests {
		current := currentParent.Resources.Requests[k]
		r, err := re.computeResources(parentPath+".requests."+string(k), current)
		if err != nil {
			return nil, err
		}
		if r == nil {
			continue
		}
		if container.Resources.Requests == nil {
			container.Resources.Requests = make(v1.ResourceList)
		}
		container.Resources.Requests[k] = *r
	}

	if len(container.Resources.Requests) == 0 && len(container.Resources.Limits) == 0 {
		return nil, nil
	}

	return container, nil
}

// AddObservation is called whenever we observe input values
func (e *containerScalingRuleEvaluator) addObservation(inputs factors.Snapshot) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, re := range e.limits {
		re.addObservation(inputs)
	}
	for _, re := range e.requests {
		re.addObservation(inputs)
	}
}
