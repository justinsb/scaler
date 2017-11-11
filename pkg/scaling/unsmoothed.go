package scaling

import (
	"sync"

	"github.com/golang/glog"
	"k8s.io/api/core/v1"
)

type Unsmoothed struct {
	mutex  sync.Mutex
	target *v1.PodSpec
}

func NewUnsmoothed() *Unsmoothed {
	return &Unsmoothed{}
}

func (e *Unsmoothed) UpdateTarget(podSpec *v1.PodSpec) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	e.target = podSpec
}

func (e *Unsmoothed) ComputeChange(parentPath string, current *v1.PodSpec) (bool, *v1.PodSpec) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	podChanged := false
	podChanges := new(v1.PodSpec)

	if e.target == nil {
		glog.V(2).Infof("target value %s not computed", parentPath)
		return false, nil
	}

	for i := range e.target.Containers {
		targetContainer := &e.target.Containers[i]
		containerName := targetContainer.Name

		path := parentPath + "." + containerName

		var currentContainer *v1.Container
		for i := range current.Containers {
			c := &current.Containers[i]
			if c.Name == containerName {
				currentContainer = c
				break
			}
		}

		if currentContainer == nil {
			glog.Warningf("ignoring policy for non-existent container %q", path)
			continue
		}

		if changed, changes := e.updateContainer(path, currentContainer, targetContainer); changed {
			podChanges.Containers = append(podChanges.Containers, *changes)
			podChanged = true
		}
	}

	return podChanged, podChanges

}

func (e *Unsmoothed) updateContainer(path string, currentContainer *v1.Container, target *v1.Container) (bool, *v1.Container) {
	containerChanged := false
	containerChanges := new(v1.Container)
	containerChanges.Name = target.Name

	if changed, changes := e.updateResourceList(path+".Limits", currentContainer.Resources.Limits, target.Resources.Limits); changed {
		containerChanges.Resources.Limits = changes
		containerChanged = true
	}

	if changed, changes := e.updateResourceList(path+".Requests", currentContainer.Resources.Requests, target.Resources.Requests); changed {
		containerChanges.Resources.Requests = changes
		containerChanged = true
	}

	return containerChanged, containerChanges
}

func (e *Unsmoothed) updateResourceList(parentPath string, currentResources v1.ResourceList, target v1.ResourceList) (bool, v1.ResourceList) {
	changed := false
	var changes v1.ResourceList

	for k, v := range target {
		path := parentPath + "." + string(k)

		currentQuantity, found := currentResources[k]
		if found && currentQuantity.Cmp(v) == 0 {
			glog.V(8).Infof("value for %s matches: %s", path, v)
			continue
		}

		changed = true
		if changes == nil {
			changes = make(v1.ResourceList)
		}
		changes[k] = v
	}

	return changed, changes
}

type Info struct {
	Target *v1.PodSpec `json:"target"`
}

func (e *Unsmoothed) Query() *Info {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	info := &Info{
		Target: e.target,
	}
	return info
}
