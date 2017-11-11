package scaling

import (
	"fmt"

	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"github.com/justinsb/scaler/pkg/factors"
	"github.com/golang/glog"
)

// ComputeResources computes a list of resource quantities based on the input state and the specified policy
// It returns a partial PodSpec with the resources we should apply
func ComputeResources(inputs factors.Snapshot, policy *scalingpolicy.ScalingPolicySpec) (*v1.PodSpec, error) {
	var err error

	podSpec := &v1.PodSpec{}

	for _, containerPolicy := range policy.Containers {
		container := v1.Container{
			Name: containerPolicy.Name,
		}

		container.Resources.Limits, err = buildResourceRequirements(inputs, containerPolicy.Resources.Limits)
		if err != nil {
			return nil, err
		}
		container.Resources.Requests, err = buildResourceRequirements(inputs, containerPolicy.Resources.Requests)
		if err != nil {
			return nil, err
		}

		podSpec.Containers = append(podSpec.Containers, container)
	}

	return podSpec, nil
}

// buildResourceRequirements applies the list of rules to the current input state to compute a list of resource quantities
func buildResourceRequirements(inputs factors.Snapshot, rules []scalingpolicy.ResourceScalingRule) (v1.ResourceList, error) {
	// TODO: Scale isn't really exposed by resource.Quantity??
	scale := resource.Milli

	accumulators := make(map[v1.ResourceName]*resourceAccumulator)
	for i := range rules {
		rule := &rules[i]

		input, found, err := inputs.Get(rule.Input)
		if err != nil {
			return nil, fmt.Errorf("error reading %q: %v", rule.Input, err)
		}

		if !found {
			glog.Warningf("value %q not found", rule.Input)
			// We still continue, we just apply the base value
		}

		accumulator := accumulators[rule.Resource]
		if accumulator == nil {
			accumulator = new(resourceAccumulator)
			accumulators[rule.Resource] = accumulator
		}

		var v int64
		if !rule.Base.IsZero() {
			accumulator.mergeFormat(&rule.Base)

			v = rule.Base.ScaledValue(scale)
		}

		if found && !rule.Step.IsZero() {
			accumulator.mergeFormat(&rule.Step)

			step := float64(rule.Step.ScaledValue(scale)) * input
			v += int64(step)
		}

		accumulator.accumulateValue(v)
	}

	resourceList := make(v1.ResourceList)
	for k, v := range accumulators {
		r, err := v.asQuantity(scale)
		if err != nil {
			return nil, err
		}
		resourceList[k] = *r
	}

	return resourceList, nil
}

// resourceAccumulator holds the state of a resource.Quantity as we are building it
type resourceAccumulator struct {
	format resource.Format
	value  int64
}

// asQuantity builds the resource.Quantity we have computed
func (a *resourceAccumulator) asQuantity(scale resource.Scale) (*resource.Quantity, error) {
	q := resource.NewScaledQuantity(a.value, scale)
	q.Format = a.format
	return q, nil
}

// mergeFormat incorporates the format from the resource.Quantity, if we don't already have one
func (a *resourceAccumulator) mergeFormat(q *resource.Quantity) {
	if a.format == "" {
		a.format = q.Format
	}
}

// accumulateValue adds the provided value to the accumulated resource quantity
func (a *resourceAccumulator) accumulateValue(v int64) {
	a.value += v
}
