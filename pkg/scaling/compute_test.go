package scaling

import (
	"testing"

	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/debug"
	"github.com/justinsb/scaler/pkg/factors/static"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestComputeResources(t *testing.T) {
	grid := []struct {
		Name     string
		Inputs   map[string]float64
		Policy   *scalingpolicy.ScalingPolicySpec
		Expected *v1.PodSpec
	}{
		{
			Name:     "Empty policy",
			Policy:   &scalingpolicy.ScalingPolicySpec{},
			Expected: &v1.PodSpec{},
		},
		{
			Name: "Trivial no-scale policy",
			Policy: &scalingpolicy.ScalingPolicySpec{
				Containers: []scalingpolicy.ContainerScalingRule{
					{
						Name: "container1",
						Resources: scalingpolicy.ResourceRequirements{
							Limits: []scalingpolicy.ResourceScalingRule{
								{
									Resource: v1.ResourceCPU,
									Base:     resource.MustParse("2000m"),
								},
							},
						},
					},
				},
			},
			Expected: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container1",
						Resources: v1.ResourceRequirements{
							Limits: v1.ResourceList{
								v1.ResourceCPU: resource.MustParse("2000m"),
							},
						},
					},
				},
			},
		},
		{
			Name: "Simple linear scale policy",
			Inputs: map[string]float64{
				"pods": 10,
			},
			Policy: &scalingpolicy.ScalingPolicySpec{
				Containers: []scalingpolicy.ContainerScalingRule{
					{
						Name: "container1",
						Resources: scalingpolicy.ResourceRequirements{
							Requests: []scalingpolicy.ResourceScalingRule{
								{
									Input:    "pods",
									Resource: v1.ResourceMemory,
									Base:     resource.MustParse("100Mi"),
									Step:     resource.MustParse("10Mi"),
								},
							},
						},
					},
				},
			},
			Expected: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container1",
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceMemory: resource.MustParse("200Mi"),
							},
						},
					},
				},
			},
		},
		{
			Name: "Policy allows negative coefficients",
			Inputs: map[string]float64{
				"pods": 5,
			},
			Policy: &scalingpolicy.ScalingPolicySpec{
				Containers: []scalingpolicy.ContainerScalingRule{
					{
						Name: "container1",
						Resources: scalingpolicy.ResourceRequirements{
							Requests: []scalingpolicy.ResourceScalingRule{
								{
									Input:    "pods",
									Resource: v1.ResourceMemory,
									Base:     resource.MustParse("100Mi"),
									Step:     resource.MustParse("-10Mi"),
								},
							},
						},
					},
				},
			},
			Expected: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container1",
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceMemory: resource.MustParse("50Mi"),
							},
						},
					},
				},
			},
		},
		{
			Name: "Mixed units work",
			Inputs: map[string]float64{
				"pods": 5,
			},
			Policy: &scalingpolicy.ScalingPolicySpec{
				Containers: []scalingpolicy.ContainerScalingRule{
					{
						Name: "container1",
						Resources: scalingpolicy.ResourceRequirements{
							Requests: []scalingpolicy.ResourceScalingRule{
								{
									Input:    "pods",
									Resource: v1.ResourceMemory,
									Base:     resource.MustParse("10Mi"),
									Step:     resource.MustParse("1M"),
								},
							},
						},
					},
				},
			},
			Expected: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container1",
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceMemory: *resource.NewScaledQuantity((1024*1024*10)+(1000*1000*5), 0),
							},
						},
					},
				},
			},
		},
		{
			Name: "Multiple rules on same resource",
			Inputs: map[string]float64{
				"nodes": 2,
				"pods":  4,
			},
			Policy: &scalingpolicy.ScalingPolicySpec{
				Containers: []scalingpolicy.ContainerScalingRule{
					{
						Name: "container1",
						Resources: scalingpolicy.ResourceRequirements{
							Requests: []scalingpolicy.ResourceScalingRule{
								{
									Input:    "pods",
									Resource: v1.ResourceMemory,
									Base:     resource.MustParse("100Mi"),
									Step:     resource.MustParse("10Mi"),
								},
								{
									Input:    "nodes",
									Resource: v1.ResourceMemory,
									Step:     resource.MustParse("20Mi"),
								},
							},
						},
					},
				},
			},
			Expected: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container1",
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceMemory: resource.MustParse("180Mi"),
							},
						},
					},
				},
			},
		},
		{
			Name: "Multiple resources",
			Inputs: map[string]float64{
				"nodes": 2,
				"pods":  4,
			},
			Policy: &scalingpolicy.ScalingPolicySpec{
				Containers: []scalingpolicy.ContainerScalingRule{
					{
						Name: "container1",
						Resources: scalingpolicy.ResourceRequirements{
							Requests: []scalingpolicy.ResourceScalingRule{
								{
									Input:    "pods",
									Resource: v1.ResourceMemory,
									Base:     resource.MustParse("200Mi"),
									Step:     resource.MustParse("7Mi"),
								},
								{
									Input:    "nodes",
									Resource: v1.ResourceCPU,
									Base:     resource.MustParse("100m"),
									Step:     resource.MustParse("23m"),
								},
							},
						},
					},
				},
			},
			Expected: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Name: "container1",
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceMemory: resource.MustParse("228Mi"),
								v1.ResourceCPU:    resource.MustParse("146m"),
							},
						},
					},
				},
			},
		},
	}

	for _, g := range grid {
		factors := static.NewStaticFactors(g.Inputs)
		snaphot, err := factors.Snapshot()
		if err != nil {
			t.Errorf("snapshot failed: %v", err)
		}
		actual, err := ComputeResources(snaphot, g.Policy)
		if err != nil {
			t.Errorf("unexpected error from test\npolicy=%v\nerror=%v", debug.Print(g.Policy), err)
			continue
		}
		if !equality.Semantic.DeepEqual(actual, g.Expected) {
			t.Errorf("test failure\nname=%s\npolicy=%v\n  actual=%v\nexpected=%v", g.Name, debug.Print(g.Policy), debug.Print(actual), debug.Print(g.Expected))
			continue
		}
	}
}
