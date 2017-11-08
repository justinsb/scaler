package scaling

import (
	"fmt"

	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"gopkg.in/inf.v0"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ComputeResources computes a list of resource quantities based on the input state and the specified policy
// It returns a partial PodSpec with the resources we should apply
func ComputeResources(inputs map[string]int64, policy *scalingpolicy.ScalingPolicySpec) (*v1.PodSpec, error) {
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
func buildResourceRequirements(inputs map[string]int64, rules []scalingpolicy.ResourceScalingRule) (v1.ResourceList, error) {
	accumulators := make(map[v1.ResourceName]*resourceAccumulator)
	for i := range rules {
		rule := &rules[i]

		input := inputs[rule.Input]

		accumulator := accumulators[rule.Resource]
		if accumulator == nil {
			accumulator = new(resourceAccumulator)
			accumulators[rule.Resource] = accumulator
		}

		var v inf.Dec

		if !rule.Base.IsZero() {
			accumulator.mergeFormat(&rule.Base)

			v.Add(&v, rule.Base.AsDec())
		}

		if !rule.Step.IsZero() {
			accumulator.mergeFormat(&rule.Step)

			var step inf.Dec
			step.Mul(rule.Step.AsDec(), inf.NewDec(input, 0))

			v.Add(&v, &step)
		}

		accumulator.accumulateValue(&v)
	}

	resourceList := make(v1.ResourceList)
	for k, v := range accumulators {
		r, err := v.asQuantity()
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
	value  inf.Dec
}

// asQuantity builds the resource.Quantity we have computed
func (a *resourceAccumulator) asQuantity() (*resource.Quantity, error) {
	unscaled, ok := a.value.Unscaled()
	if !ok {
		return nil, fmt.Errorf("cannot represent value %v", a.value)
	}
	var r resource.Quantity
	// Note that v.Scale() for inf.Dec is the negative value of Scale() for resource.Quantity
	r.SetScaled(unscaled, resource.Scale(-a.value.Scale()))
	r.Format = a.format

	return &r, nil
}

// mergeFormat incorporates the format from the resource.Quantity, if we don't already have one
func (a *resourceAccumulator) mergeFormat(q *resource.Quantity) {
	if a.format == "" {
		a.format = q.Format
	}
}

// accumulateValue adds the provided value to the accumulated resource quantity
func (a *resourceAccumulator) accumulateValue(v *inf.Dec) {
	a.value.Add(&a.value, v)
}
