package smoothing

import (
	"sync"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/debug"
	"github.com/justinsb/scaler/pkg/factors"
	"github.com/justinsb/scaler/pkg/http"
	"github.com/justinsb/scaler/pkg/scaling"
	"github.com/justinsb/scaler/pkg/timeutil"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const minObservationsForPercentile = 3

type resourceStatus struct {
	histogram Histogram
}

type containerStatus struct {
	limits   resourceStatusMap
	requests resourceStatusMap
}

type resourceStatusMap struct {
	m map[v1.ResourceName]*resourceStatus
}

func (r *resourceStatusMap) Get(key v1.ResourceName) *resourceStatus {
	if r.m == nil {
		r.m = make(map[v1.ResourceName]*resourceStatus)
	}
	v := r.m[key]
	if v == nil {
		v = &resourceStatus{}

		// TODO: Make limit configurable
		v.histogram.Scale = resource.Milli
		v.histogram.Limit = 30

		r.m[key] = v
	}
	return v
}

// PercentileSmoothing prevents rapid changing of the configured values, even as the modelled optimum value changes rapidly.
// It tracks a sliding-window of recent target values, and will only change the smoothed value when the current
// value is not in the 70-90% range.  When the current value is out of range, we will set it to the 80% optimum value.
type PercentileSmoothing struct {
	clock *timeutil.MonotonicClock

	mutex sync.Mutex

	containers map[string]*containerStatus

	latestTarget             *v1.PodSpec
	latestScaleUpThreshold   *v1.PodSpec
	latestScaleDownThreshold *v1.PodSpec

	latestActual *v1.PodSpec

	rule v1alpha1.PercentileSmoothing
}

func NewPercentileSmoothing(clock *timeutil.MonotonicClock, rule *v1alpha1.PercentileSmoothing) Smoothing {
	s := &PercentileSmoothing{
		clock: clock,

		containers: make(map[string]*containerStatus),
	}
	s.updateRule(rule)
	return s
}

func (s *PercentileSmoothing) UpdateTarget(snapshot factors.Snapshot, policy *v1alpha1.ScalingPolicySpec) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	podSpec, err := scaling.ComputeResources(snapshot, policy)
	if err != nil {
		return err
	}

	glog.V(4).Infof("computed target values: %s", debug.Print(podSpec))

	s.latestTarget = podSpec

	reltime := s.clock.Nanos()

	for _, container := range podSpec.Containers {
		cs := s.containers[container.Name]
		if cs == nil {
			cs = &containerStatus{}
			s.containers[container.Name] = cs
		}

		for k, v := range container.Resources.Limits {
			rs := cs.limits.Get(k)
			rs.histogram.Add(reltime, v)
		}

		for k, v := range container.Resources.Requests {
			rs := cs.requests.Get(k)
			rs.histogram.Add(reltime, v)
		}
	}

	// TODO: GC old values

	return nil
}

func (s *PercentileSmoothing) updateRule(update *v1alpha1.PercentileSmoothing) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	r := *update

	if r.Target == 0.0 {
		r.Target = 0.80
	}

	if r.HighThreshold == 0.0 {
		// TODO: Assume normal distribution to compute defaults ?
		r.HighThreshold = r.Target + 0.10
		if r.HighThreshold > 1.0 {
			r.HighThreshold = 1.0
		}
	}

	if r.LowThreshold == 0.0 {
		// TODO: Assume normal distribution to compute defaults ?
		r.LowThreshold = r.Target - 0.10
		if r.LowThreshold < 0.0 {
			r.LowThreshold = 0.0
		}
	}

	s.rule = r
}

func (s *PercentileSmoothing) ComputeChange(parentPath string, current *v1.PodSpec) (bool, *v1.PodSpec) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.latestActual = current

	podChanged := false
	podChanges := new(v1.PodSpec)

	if s.latestTarget == nil {
		glog.Infof("no data for %s", parentPath)
		return podChanged, podChanges
	}

	scaleDownThreshold := &v1.PodSpec{}
	scaleUpThreshold := &v1.PodSpec{}

	for i := range s.latestTarget.Containers {
		containerTarget := &s.latestTarget.Containers[i]
		containerName := containerTarget.Name

		status := s.containers[containerName]
		if status == nil {
			glog.Infof("insufficient data to compute target value for %s", containerName)
			continue
		}

		currentContainer := findContainerByName(current.Containers, containerName)
		if currentContainer == nil {
			glog.Warningf("ignoring target for non-existent container %q", containerName)
			continue
		}

		scaleDownThresholdContainer := &v1.Container{Name: containerName}
		scaleUpThresholdContainer := &v1.Container{Name: containerName}

		if changed, changes := s.updateContainer(containerName, currentContainer, containerTarget, scaleDownThresholdContainer, scaleUpThresholdContainer, status); changed {
			podChanges.Containers = append(podChanges.Containers, *changes)
			podChanged = true
		}

		scaleDownThreshold.Containers = append(scaleDownThreshold.Containers, *scaleDownThresholdContainer)
		scaleUpThreshold.Containers = append(scaleUpThreshold.Containers, *scaleUpThresholdContainer)
	}

	s.latestScaleUpThreshold = scaleUpThreshold
	s.latestScaleDownThreshold = scaleDownThreshold
	return podChanged, podChanges

}

func (s *PercentileSmoothing) updateContainer(
	path string,
	currentContainer *v1.Container,
	target *v1.Container,
	scaleDownThreshold *v1.Container,
	scaleUpThreshold *v1.Container,
	status *containerStatus) (bool, *v1.Container) {
	containerChanged := false
	containerChanges := new(v1.Container)

	containerChanges.Name = target.Name

	scaleDownThreshold.Resources.Limits = make(v1.ResourceList)
	scaleUpThreshold.Resources.Limits = make(v1.ResourceList)
	scaleDownThreshold.Resources.Requests = make(v1.ResourceList)
	scaleUpThreshold.Resources.Requests = make(v1.ResourceList)

	if changed, changes := s.updateResourceList(path+".Limits", currentContainer.Resources.Limits, target.Resources.Limits, scaleDownThreshold.Resources.Limits, scaleUpThreshold.Resources.Limits, &status.limits); changed {
		containerChanges.Resources.Limits = changes
		containerChanged = true
	}

	if changed, changes := s.updateResourceList(path+".Requests", currentContainer.Resources.Requests, target.Resources.Requests, scaleDownThreshold.Resources.Requests, scaleUpThreshold.Resources.Requests, &status.requests); changed {
		containerChanges.Resources.Requests = changes
		containerChanged = true
	}

	return containerChanged, containerChanges
}

func (s *PercentileSmoothing) updateResourceList(
	parentPath string,
	currentResources v1.ResourceList,
	target v1.ResourceList,
	scaleDownThreshold v1.ResourceList,
	scaleUpThreshold v1.ResourceList,
	status *resourceStatusMap) (bool, v1.ResourceList) {

	changed := false
	var changes v1.ResourceList

	for resource := range target {
		path := parentPath + "." + string(resource)

		rs := status.Get(resource)

		currentQuantity, found := currentResources[resource]
		if found {
			pLow, ok := rs.histogram.Percentile(s.rule.LowThreshold)
			if !ok {
				glog.Infof("insufficient data to compute percentile value for %s @ %f", path, s.rule.LowThreshold)
				continue
			}
			scaleDownThreshold[resource] = pLow

			pHigh, ok := rs.histogram.Percentile(s.rule.HighThreshold)
			if !ok {
				glog.Infof("insufficient data to compute percentile value for %s @ %f", path, s.rule.HighThreshold)
				continue
			}
			scaleUpThreshold[resource] = pHigh

			if currentQuantity.Cmp(pLow) >= 0 && currentQuantity.Cmp(pHigh) <= 0 {
				// Value in tolerable range
				glog.V(4).Infof("value for %s (%s) is in-range: (%s-%s)", path, currentQuantity.String(), pLow.String(), pHigh.String())
				continue
			}
		}

		estimated, ok := rs.histogram.Percentile(s.rule.Target)
		// TODO: quantization?
		if !ok {
			glog.Infof("insufficient data to compute target %f value for %s", s.rule.Target, path)
			continue
		}

		changed = true
		if changes == nil {
			changes = make(v1.ResourceList)
		}
		glog.V(4).Infof("current value for %s (%s) is out of range; will use %s", path, currentQuantity.String(), estimated.String())
		changes[resource] = estimated
	}

	return changed, changes
}

func (s *PercentileSmoothing) Query() *http.Info {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	info := &http.Info{
		LatestTarget:       s.latestTarget,
		ScaleDownThreshold: s.latestScaleDownThreshold,
		ScaleUpThreshold:   s.latestScaleUpThreshold,
		LatestActual:       s.latestActual,
		Histograms:         make(map[string]*http.HistogramInfo),
	}

	for k, container := range s.containers {
		for m, metric := range container.requests.m {
			info.Histograms[k+".requests."+string(m)] = metric.histogram.Query(s.clock)
		}
		for m, metric := range container.limits.m {
			info.Histograms[k+".limits."+string(m)] = metric.histogram.Query(s.clock)
		}
	}
	return info
}
