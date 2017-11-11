package scaling

import (
	"github.com/golang/glog"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func asInt64(q resource.Quantity) (int64, bool) {
	return q.Value(), true
}

func Quantize(q resource.Quantity, rule *scalingpolicy.QuantizationRule) resource.Quantity {
	qInt, ok := asInt64(q)
	if !ok {
		glog.Warningf("ignoring out of range quantity %s", q)
		return q
	}

	var stepV float64
	if !rule.Step.IsZero() {
		stepInt, ok := asInt64(rule.Step)
		if !ok {
			// step will be treated as missing
			glog.Warningf("ignoring out of range step value %s", rule.Step)
		} else {
			stepV = float64(stepInt)
		}
	}
	var maxStep float64
	if !rule.MaxStep.IsZero() {
		maxStepInt, ok := asInt64(rule.MaxStep)
		if !ok {
			// MaxStep will be treated as missing
			glog.Warningf("ignoring out of range MaxStep value %s", rule.MaxStep)
		} else {
			maxStep = float64(maxStepInt)
		}
	}

	current, ok := asInt64(rule.Base)
	if !ok {
		// Base will be treated as missing, we will start from zero
		glog.Warningf("ignoring out of range base value %s", rule.Base)
		current = 0
	}

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
			return *resource.NewQuantity(current, q.Format)
		}

		if int64(stepV) < 1 {
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
