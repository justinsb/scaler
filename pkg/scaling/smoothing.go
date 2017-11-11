package scaling

//
//import (
//	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
//	"k8s.io/api/core/v1"
//	"sync"
//	"sort"
//	"time"
//	"github.com/golang/glog"
//	"k8s.io/apimachinery/pkg/api/resource"
//)
//
//const minObservationsForPercentile = 3
//
//type Histogram struct {
//	limit int
//
//	mutex      sync.Mutex
//	values     []int64
//	pos        int
//	lastFormat resource.Format
//}
//
//func (h *Histogram) Add(t int64, q resource.Quantity) {
//	qv, ok := q.AsInt64()
//	if !ok {
//		glog.Warningf("ignoring out-of-range value %q", q)
//		return
//	}
//
//	h.mutex.Lock()
//	defer h.mutex.Unlock()
//
//	h.lastFormat = q.Format
//
//	if len(h.values) < h.limit {
//		h.values = append(h.values, qv)
//	} else {
//		h.values[h.pos] = qv
//		h.pos++
//		if h.pos >= h.limit {
//			h.pos = 0
//		}
//	}
//}
//
//func (h *Histogram) Percentile(ratio float32) (resource.Quantity, bool) {
//	h.mutex.Lock()
//	defer h.mutex.Unlock()
//
//	if len(h.values) < minObservationsForPercentile {
//		return resource.Quantity{}, false
//	}
//
//	sorted := make([]int64, len(h.values))
//	copy(sorted, h.values)
//	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
//
//	index := int(float32(len(h.values)) * ratio)
//	if index >= len(h.values) {
//		index--
//	}
//
//	v := sorted[index]
//	return *resource.NewQuantity(v, h.lastFormat), true
//}
//
//func (h *Histogram) EstimatePercentile(value *resource.Quantity) (float32, bool){
//	qv, ok := value.AsInt64()
//	if !ok {
//		glog.Warningf("ignoring out-of-range value %q", value)
//		return 0, false
//	}
//
//	h.mutex.Lock()
//	defer h.mutex.Unlock()
//
//
//	if len(h.values) < minObservationsForPercentile {
//		return 0, false
//	}
//
//	n := len(h.values)
//	if n == 0 {
//		return 0, false
//	}
//
//	x := 0
//	for i := 0; i < n; i++ {
//		if h.values[i] <= qv {
//			x++
//		}
//	}
//
//	return float32(x) / float32(n), true
//}
//
//type resourceStatus struct {
//	histogram Histogram
//}
//
//type containerStatus struct {
//	limits   resourceStatusMap
//	requests resourceStatusMap
//}
//
//type resourceStatusMap struct {
//	m map[v1.ResourceName]*resourceStatus
//}
//
//func (r *resourceStatusMap) Get(key v1.ResourceName) *resourceStatus {
//	if r.m == nil {
//		r.m = make(map[v1.ResourceName]*resourceStatus)
//	}
//	v := r.m[key]
//	if v == nil {
//		v = &resourceStatus{}
//		r.m[key] = v
//	}
//	return v
//}
//
//type Estimator struct {
//	baseTime time.Time
//
//	mutex sync.Mutex
//
//	policy     *scalingpolicy.ScalingPolicySpec
//	containers map[string]*containerStatus
//}
//
//func (e *Estimator) Update(t time.Time, podSpec *v1.PodSpec) {
//	e.mutex.Lock()
//	defer e.mutex.Unlock()
//
//	reltime := t.Sub(e.baseTime).Nanoseconds()
//
//	for _, container := range podSpec.Containers {
//		s := e.containers[container.Name]
//		if s == nil {
//			s = &containerStatus{
//			}
//			e.containers[container.Name] = s
//		}
//
//		for k, v := range container.Resources.Limits {
//			rs := s.limits.Get(k)
//			rs.histogram.Add(reltime, v)
//		}
//
//		for k, v := range container.Resources.Requests {
//			rs := s.requests.Get(k)
//			rs.histogram.Add(reltime, v)
//		}
//	}
//
//	// TODO: GC old values
//}
//
//func (e *Estimator) UpdateSpecs(current *v1.PodSpec) (bool, *v1.PodSpec) {
//	e.mutex.Lock()
//	defer e.mutex.Unlock()
//
//	podChanged := false
//	podChanges := new(v1.PodSpec)
//
//	for i := range e.policy.Containers {
//		containerPolicy := &e.policy.Containers[i]
//		containerName := containerPolicy.Name
//
//		status := e.containers[containerName]
//		if status == nil {
//			glog.Infof("insufficient data to compute target value for %s", containerName)
//			continue
//		}
//
//		var currentContainer *v1.Container
//		for i := range current.Containers {
//			c := &current.Containers[i]
//			if c.Name == containerName {
//				currentContainer = c
//				break
//			}
//		}
//
//		if currentContainer == nil {
//			glog.Warningf("ignoring policy for non-existent container %q", containerName)
//			continue
//		}
//
//		if changed, changes := e.updateContainer(containerName, currentContainer, containerPolicy, status); changed {
//			podChanges.Containers = append(podChanges.Containers, *changes)
//			podChanged = true
//		}
//	}
//
//	return podChanged, podChanges
//
//}
//
//func (e *Estimator) updateContainer(path string, currentContainer *v1.Container, containerPolicy *scalingpolicy.ContainerScalingRule, status *containerStatus) (bool, *v1.Container) {
//	containerChanged := false
//	containerChanges := new(v1.Container)
//
//	if changed, changes := e.updateResourceList(path+".Limits", currentContainer.Resources.Limits, containerPolicy.Resources.Limits, &status.limits); changed {
//		containerChanges.Resources.Limits = changes
//		containerChanged = true
//	}
//
//	if changed, changes := e.updateResourceList(path+".Requests", currentContainer.Resources.Requests, containerPolicy.Resources.Requests, &status.requests); changed {
//		containerChanges.Resources.Requests = changes
//		containerChanged = true
//	}
//
//	return containerChanged, containerChanges
//}
//
//func (e *Estimator) updateResourceList(parentPath string, currentResources v1.ResourceList, rules []scalingpolicy.ResourceScalingRule, status *resourceStatusMap) (bool, v1.ResourceList) {
//	changed := false
//	var changes v1.ResourceList
//
//	doneResources := make(map[v1.ResourceName]bool)
//	for _, rule := range rules {
//		if doneResources[rule.Resource] {
//			continue
//		}
//		doneResources[rule.Resource] = true
//
//		path := parentPath + "." + string(rule.Resource)
//
//		currentQuantity, found := currentResources[rule.Resource]
//		if found {
//			percentile, ok := status.Get(rule.Resource).histogram.EstimatePercentile(&currentQuantity)
//			if !ok {
//				glog.Infof("insufficient data to compute percentile value for %s", path)
//				continue
//			}
//			if percentile < 0.7 || percentile > 0.9 {
//				// Value in tolerable range
//				glog.V(8).Infof("value for %s (%s) is tolerable: %f", path, currentQuantity, percentile)
//				continue
//			}
//		}
//
//		estimated, ok := status.Get(rule.Resource).histogram.Percentile(0.8)
//		// TODO: quantization?
//		if !ok {
//			glog.Infof("insufficient data to compute target value for %s", path)
//			continue
//		}
//
//		changed = true
//		if changes == nil {
//			changes = make(v1.ResourceList)
//		}
//		changes[rule.Resource] = estimated
//	}
//
//	return changed, changes
//}
