package scaling

import (
	"testing"

	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/debug"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestQuantization(t *testing.T) {
	grid := []struct {
		Name     string
		Input    string
		Rule     *scalingpolicy.QuantizationRule
		Expected string
	}{
		{
			Name:     "Empty quantization",
			Input:    "200M",
			Rule:     &scalingpolicy.QuantizationRule{},
			Expected: "200M",
		},
		{
			Name:  "Simple step with exact match",
			Input: "200M",
			Rule: &scalingpolicy.QuantizationRule{
				Base: resource.MustParse("100M"),
				Step: resource.MustParse("100M"),
			},
			Expected: "200M",
		},
		{
			Name:  "Simple step with straddle",
			Input: "210M",
			Rule: &scalingpolicy.QuantizationRule{
				Base: resource.MustParse("100M"),
				Step: resource.MustParse("100M"),
			},
			Expected: "300M",
		},
		{
			Name:  "Step with no base",
			Input: "210M",
			Rule: &scalingpolicy.QuantizationRule{
				Step: resource.MustParse("100M"),
			},
			Expected: "300M",
		},
		{
			Name:  "Quadratic step",
			Input: "100M",
			Rule: &scalingpolicy.QuantizationRule{
				Base:      resource.MustParse("5M"),
				Step:      resource.MustParse("10M"),
				StepRatio: 2.0,
			},
			// 5, +10 = 15, +20 = 35, +40 = 75, +80 = 155
			Expected: "155M",
		},
		{
			Name:  "Quadratic step with max",
			Input: "100M",
			Rule: &scalingpolicy.QuantizationRule{
				Base:      resource.MustParse("5M"),
				Step:      resource.MustParse("10M"),
				StepRatio: 2.0,
				MaxStep:   resource.MustParse("30M"),
			},
			// 5, +10 = 15, +20 = 35, +30 = 65, +30 = 95, +30 = 125
			Expected: "125M",
		},
	}

	for _, g := range grid {
		q := resource.MustParse(g.Input)
		actual := Quantize(q, g.Rule)

		if actual.String() != g.Expected {
			t.Errorf("test failure\npolicy=%v\n  actual=%v\nexpected=%v", debug.Print(g.Rule), actual.String(), g.Expected)
			continue
		}
	}
}
