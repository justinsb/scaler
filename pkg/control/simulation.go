package control

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/control/target"
	"github.com/justinsb/scaler/pkg/simulate"
	"github.com/justinsb/scaler/pkg/timeutil"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/clock"
)

func RunSimulation(policy *scalingpolicy.ScalingPolicy, options *options.AutoScalerConfig) (*simulate.Run, error) {
	universe := target.NewSimulationTarget()

	nodeCount := 100
	updateClusterState(universe, nodeCount)

	universe.Current = buildMockPodSpec(policy)

	state, err := NewState(universe, options)
	if err != nil {
		return nil, err
	}
	baseTime := time.Now()
	fakeClock := clock.NewFakeClock(baseTime)
	state.clock = timeutil.NewMonotonicClock(fakeClock)

	policy = policy.DeepCopy()
	state.upsert(policy)

	pollPeriod := int(options.PollPeriod.Seconds())
	updatePeriod := int(options.UpdatePeriod.Seconds())

	var errors []error

	run := &simulate.Run{}

	for t := 0; t < 3600; t++ {
		nodeCount += int(rand.NormFloat64())
		updateClusterState(universe, nodeCount)

		timeNow := baseTime.Add(time.Duration(t) * time.Second)
		fakeClock.SetTime(timeNow)

		if (t % pollPeriod) == 0 {
			if err := state.computeTargetValues(); err != nil {
				errors = append(errors, err)
			}
		}

		if (t % updatePeriod) == 0 {
			if err := state.updateValues(); err != nil {
				errors = append(errors, err)
			}
		}

		si := state.Query().(*StateInfo)

		var pi *PolicyInfo
		for k := range si.Policies {
			pi = si.Policies[k]
		}

		var latestTarget *v1.PodSpec
		var scaleDownThreshold *v1.PodSpec
		var scaleUpThreshold *v1.PodSpec
		if pi != nil && pi.State != nil {
			latestTarget = pi.State.LatestTarget
			scaleDownThreshold = pi.State.ScaleDownThreshold
			scaleUpThreshold = pi.State.ScaleUpThreshold
		}

		run.Add(t, universe.ClusterState, universe.Current, latestTarget, scaleDownThreshold, scaleUpThreshold)
	}

	run.UpdateCount = universe.UpdateCount

	if len(errors) != 0 {
		glog.Warningf("%d errors in simulation.  first error=%v", len(errors), errors[0])
	}
	return run, nil
}

func updateClusterState(t *target.SimulationTarget, nodeCount int) {
	t.ClusterState = &target.ClusterStats{
		NodeCount:          nodeCount,
		NodeSumAllocatable: make(v1.ResourceList),
	}
	t.ClusterState.NodeSumAllocatable[v1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%d", nodeCount*4))
	t.ClusterState.NodeSumAllocatable[v1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dGi", nodeCount*32))
}

func buildMockPodSpec(policy *scalingpolicy.ScalingPolicy) *v1.PodSpec {
	ps := &v1.PodSpec{}
	for _, c := range policy.Spec.Containers {
		mc := v1.Container{
			Name: c.Name,
		}
		ps.Containers = append(ps.Containers, mc)
	}
	return ps
}
