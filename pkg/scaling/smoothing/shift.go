package smoothing

import (
	"sync"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/debug"
	"github.com/justinsb/scaler/pkg/factors"
	"github.com/justinsb/scaler/pkg/http"
	"github.com/justinsb/scaler/pkg/scaling"
	"k8s.io/api/core/v1"
)

// ResourceShiftSmoothing avoids "flapping" of values by offsetting the value at which we scale down.
type ResourceShiftSmoothing struct {
	mutex sync.Mutex
	rule  v1alpha1.ShiftSmoothing

	latestTarget    *v1.PodSpec
	latestScaleDown *v1.PodSpec

	latestActual *v1.PodSpec
}

func NewResourceShiftSmoothing(rule *v1alpha1.ShiftSmoothing) Smoothing {
	s := &ResourceShiftSmoothing{}
	s.updateRule(rule)
	return s
}

func (s *ResourceShiftSmoothing) UpdateTarget(snapshot factors.Snapshot, policy *v1alpha1.ScalingPolicySpec) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	podSpec, err := scaling.ComputeResources(snapshot, policy)
	if err != nil {
		return err
	}
	glog.V(4).Infof("computed target values: %s", debug.Print(podSpec))

	scaleDownPodSpec, err := scaling.ComputeResourcesShifted(snapshot, policy, &s.rule)
	if err != nil {
		return err
	}
	glog.V(4).Infof("computed shifted values: %s", debug.Print(scaleDownPodSpec))

	s.latestTarget = podSpec
	s.latestScaleDown = scaleDownPodSpec

	return nil
}

func (s *ResourceShiftSmoothing) updateRule(update *v1alpha1.ShiftSmoothing) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	r := *update

	s.rule = r
}

func findContainerByName(containers []v1.Container, name string) *v1.Container {
	for i := range containers {
		c := &containers[i]
		if c.Name == name {
			return c
		}
	}
	return nil
}

func (s *ResourceShiftSmoothing) ComputeChange(parentPath string, current *v1.PodSpec) (bool, *v1.PodSpec) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.latestActual = current

	podChanged := false
	podChanges := new(v1.PodSpec)

	if s.latestTarget == nil {
		glog.Infof("no data for %s", parentPath)
		return podChanged, podChanges
	}

	for i := range s.latestTarget.Containers {
		targetContainer := &s.latestTarget.Containers[i]
		containerName := targetContainer.Name

		path := parentPath + "." + containerName

		currentContainer := findContainerByName(current.Containers, containerName)
		if currentContainer == nil {
			glog.Warningf("ignoring policy for non-existent container %q", path)
			continue
		}

		scaleDownContainer := findContainerByName(current.Containers, containerName)
		if scaleDownContainer == nil {
			scaleDownContainer = &v1.Container{}
		}

		if changed, changes := s.updateContainer(path, currentContainer, targetContainer, scaleDownContainer); changed {
			podChanges.Containers = append(podChanges.Containers, *changes)
			podChanged = true
		}
	}

	return podChanged, podChanges
}

func (s *ResourceShiftSmoothing) updateContainer(path string, currentContainer *v1.Container, target *v1.Container, scaleDown *v1.Container) (bool, *v1.Container) {
	containerChanged := false
	containerChanges := new(v1.Container)
	containerChanges.Name = target.Name

	if changed, changes := s.updateResourceList(path+".Limits", currentContainer.Resources.Limits, target.Resources.Limits, scaleDown.Resources.Limits); changed {
		containerChanges.Resources.Limits = changes
		containerChanged = true
	}

	if changed, changes := s.updateResourceList(path+".Requests", currentContainer.Resources.Requests, target.Resources.Requests, scaleDown.Resources.Requests); changed {
		containerChanges.Resources.Requests = changes
		containerChanged = true
	}

	return containerChanged, containerChanges
}

func (s *ResourceShiftSmoothing) updateResourceList(parentPath string, currentResources v1.ResourceList, target v1.ResourceList, scaleDown v1.ResourceList) (bool, v1.ResourceList) {
	changed := false
	var changes v1.ResourceList

	for k, v := range target {
		path := parentPath + "." + string(k)

		currentQuantity, found := currentResources[k]
		if !found {
			glog.V(8).Infof("value for %s not found; will treat as zero", path)
		}

		cmp := currentQuantity.Cmp(v)
		if cmp == 0 {
			glog.V(8).Infof("value for %s matches target: %s", path, v)
			continue
		}

		if cmp < 0 {
			// Scale down candidate, compare to scale-down threshold
			scaleDownV, found := scaleDown[k]
			if found && currentQuantity.Cmp(scaleDownV) >= 0 {
				glog.V(8).Infof("value for %s is below target (%s), but above scale-down threshold (%s)", path, v, scaleDownV)
				continue
			}
		}

		changed = true
		if changes == nil {
			changes = make(v1.ResourceList)
		}
		changes[k] = v
	}

	return changed, changes
}

func (s *ResourceShiftSmoothing) Query() *http.Info {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	info := &http.Info{
		LatestTarget:       s.latestTarget,
		ScaleDownThreshold: s.latestScaleDown,
	}
	return info
}
