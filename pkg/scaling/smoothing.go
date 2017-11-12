package scaling

import (
	"sync"
	"time"

	"github.com/golang/glog"
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

type Smoother interface {
	UpdateTarget(podSpec *v1.PodSpec)
	ComputeChange(parentPath string, current *v1.PodSpec) (bool, *v1.PodSpec)

	Query() *Info
}

type HistogramSmoother struct {
	baseTime time.Time

	mutex sync.Mutex

	containers map[string]*containerStatus

	latestTarget *v1.PodSpec
	latestActual *v1.PodSpec
}

func NewHistogramSmoothing() Smoother {
	s := &HistogramSmoother{
		baseTime: time.Now(),

		containers: make(map[string]*containerStatus),
	}
	return s
}

func (e *HistogramSmoother) UpdateTarget(podSpec *v1.PodSpec) {
	t := time.Now()

	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.latestTarget = podSpec

	reltime := t.Sub(e.baseTime).Nanoseconds()

	for _, container := range podSpec.Containers {
		s := e.containers[container.Name]
		if s == nil {
			s = &containerStatus{}
			e.containers[container.Name] = s
		}

		for k, v := range container.Resources.Limits {
			rs := s.limits.Get(k)
			rs.histogram.Add(reltime, v)
		}

		for k, v := range container.Resources.Requests {
			rs := s.requests.Get(k)
			rs.histogram.Add(reltime, v)
		}
	}

	// TODO: GC old values
}

func (e *HistogramSmoother) ComputeChange(parentPath string, current *v1.PodSpec) (bool, *v1.PodSpec) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.latestActual = current

	podChanged := false
	podChanges := new(v1.PodSpec)

	if e.latestTarget == nil {
		glog.Infof("no data for %s", parentPath)
		return podChanged, podChanges
	}

	for i := range e.latestTarget.Containers {
		containerTarget := &e.latestTarget.Containers[i]
		containerName := containerTarget.Name

		status := e.containers[containerName]
		if status == nil {
			glog.Infof("insufficient data to compute target value for %s", containerName)
			continue
		}

		var currentContainer *v1.Container
		for i := range current.Containers {
			c := &current.Containers[i]
			if c.Name == containerName {
				currentContainer = c
				break
			}
		}

		if currentContainer == nil {
			glog.Warningf("ignoring target for non-existent container %q", containerName)
			continue
		}

		if changed, changes := e.updateContainer(containerName, currentContainer, containerTarget, status); changed {
			podChanges.Containers = append(podChanges.Containers, *changes)
			podChanged = true
		}
	}

	return podChanged, podChanges

}

func (e *HistogramSmoother) updateContainer(path string, currentContainer *v1.Container, target *v1.Container, status *containerStatus) (bool, *v1.Container) {
	containerChanged := false
	containerChanges := new(v1.Container)

	containerChanges.Name = target.Name

	if changed, changes := e.updateResourceList(path+".Limits", currentContainer.Resources.Limits, target.Resources.Limits, &status.limits); changed {
		containerChanges.Resources.Limits = changes
		containerChanged = true
	}

	if changed, changes := e.updateResourceList(path+".Requests", currentContainer.Resources.Requests, target.Resources.Requests, &status.requests); changed {
		containerChanges.Resources.Requests = changes
		containerChanged = true
	}

	return containerChanged, containerChanges
}

func (e *HistogramSmoother) updateResourceList(parentPath string, currentResources v1.ResourceList, target v1.ResourceList, status *resourceStatusMap) (bool, v1.ResourceList) {
	changed := false
	var changes v1.ResourceList

	for resource := range target {
		path := parentPath + "." + string(resource)

		rs := status.Get(resource)

		currentQuantity, found := currentResources[resource]
		if found {
			p70, ok := rs.histogram.Percentile(0.70)
			if !ok {
				glog.Infof("insufficient data to compute percentile value for %s @ 70%", path)
				continue
			}
			p90, ok := rs.histogram.Percentile(0.90)
			if !ok {
				glog.Infof("insufficient data to compute percentile value for %s @ 90%", path)
				continue
			}

			if currentQuantity.Cmp(p70) >= 0 && currentQuantity.Cmp(p90) <= 0 {
				// Value in tolerable range
				glog.V(4).Infof("value for %s (%s) is tolerable: (%s-%s)", path, currentQuantity.String(), p70.String(), p90.String())
				continue
			}
		}

		estimated, ok := rs.histogram.Percentile(0.8)
		// TODO: quantization?
		if !ok {
			glog.Infof("insufficient data to compute target 80% value for %s", path)
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

type Info struct {
	LatestTarget *v1.PodSpec `json:"latestTarget"`
	LatestActual *v1.PodSpec `json:"latestActual"`

	Histograms map[string]*HistogramInfo `json:"histograms"`
}

func (e *HistogramSmoother) Query() *Info {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var actual *v1.PodSpec
	if e.latestActual != nil {
		actual = &v1.PodSpec{}
		for _, c := range e.latestActual.Containers {
			actual.Containers = append(actual.Containers, v1.Container{
				Name:      c.Name,
				Resources: c.Resources,
			})
		}
	}
	info := &Info{
		LatestTarget: e.latestTarget,
		LatestActual: actual,
		Histograms:   make(map[string]*HistogramInfo),
	}

	for k, container := range e.containers {
		for m, metric := range container.requests.m {
			info.Histograms[k+".requests."+string(m)] = metric.histogram.Query(e.baseTime)
		}
		for m, metric := range container.limits.m {
			info.Histograms[k+".limits."+string(m)] = metric.histogram.Query(e.baseTime)
		}
	}
	return info
}
