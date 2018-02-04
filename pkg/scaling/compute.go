package scaling

import (
	"fmt"
	"math"

	"github.com/golang/glog"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/factors"
	"k8s.io/apimachinery/pkg/api/resource"
)

const internalScale = resource.Milli

func computeValue(fn *scalingpolicy.ResourceScalingFunction, inputs factors.Snapshot, shift float64) (float64, error) {
	var v float64
	if !fn.Base.IsZero() {
		v = float64(fn.Base.ScaledValue(internalScale))
	}

	if fn.Input != "" {
		input, found, err := inputs.Get(fn.Input)
		if err != nil {
			return 0, fmt.Errorf("error reading %q: %v", fn.Input, err)
		}

		if !found {
			glog.Warningf("value %q not found", fn.Input)
			// We still continue, we just apply the base value
		} else if !fn.Slope.IsZero() {
			input += shift

			roundedInput := roundInput(fn, input)

			increment := float64(fn.Slope.ScaledValue(internalScale)) * roundedInput
			if fn.Per > 1 {
				increment /= float64(fn.Per)
			}
			v += increment
		}
	}

	return v, nil
}

// findSegment returns the segment of the rule, closest to the input value
func findSegment(fn *scalingpolicy.ResourceScalingFunction, input float64) *scalingpolicy.ResourceScalingSegment {
	var closest *scalingpolicy.ResourceScalingSegment
	for i := range fn.Segments {
		segment := &fn.Segments[i]
		if float64(segment.At) > input {
			continue
		}
		if closest == nil || closest.At < segment.At {
			closest = segment
		}
	}
	return closest
}

// roundInput returns the input rounded based on the closest segment
func roundInput(fn *scalingpolicy.ResourceScalingFunction, input float64) float64 {
	segment := findSegment(fn, input)
	if segment == nil {
		return input
	}
	return math.Ceil((input/float64(segment.Every))-0.001) * float64(segment.Every)
}
