package scaling

import (
	"github.com/golang/glog"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Quantize(q resource.Quantity, rule *scalingpolicy.QuantizationRule) resource.Quantity {
	scale := resource.Milli

	qInt := q.ScaledValue(scale)

	var stepV float64
	if !rule.Step.IsZero() {
		stepInt := rule.Step.ScaledValue(scale)
		stepV = float64(stepInt)
	}
	var maxStep float64
	if !rule.MaxStep.IsZero() {
		maxStepInt := rule.MaxStep.ScaledValue(scale)
		maxStep = float64(maxStepInt)
	}

	current := rule.Base.ScaledValue(scale)

	iteration := 0
	maxIterationCount := 10000
	for {
		iteration++

		// TODO: optimize this, once we like our step function: jump to the approximate value
		if iteration > maxIterationCount {
			glog.Warningf("hit max iteration count in quantization")
			break
		}
		if current >= qInt {
			rq := *resource.NewScaledQuantity(current, scale)
			rq.Format = q.Format
			return rq
		}

		if int64(stepV) < 1 {
			// We aren't going to make progress
			break
		}
		current += int64(stepV)

		if rule.StepRatio != 0.0 && rule.StepRatio != 1.0 {
			stepV *= float64(rule.StepRatio)
			if maxStep != 0.0 && stepV >= maxStep {
				stepV = maxStep
			}
		}
	}

	glog.Warningf("quantization rule did not produce any results; returning original value")
	return q
}
