package control

import (
	"fmt"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/debug"
	"github.com/justinsb/scaler/pkg/factors"
	"github.com/justinsb/scaler/pkg/scaling"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"github.com/justinsb/scaler/pkg/scaling/smoothing"
	"github.com/justinsb/scaler/pkg/http"
)

// PolicyState is the state around a single scaling policy
type PolicyState struct {
	kubeClient kubernetes.Interface
	options    *options.AutoScalerConfig

	mutex     sync.Mutex
	policies  *State
	policy    *scalingpolicy.ScalingPolicy
	smoothing smoothing.Smoothing
}

func NewPolicyState(policies *State, policy *scalingpolicy.ScalingPolicy) *PolicyState {
	s := &PolicyState{
		kubeClient: policies.client,
		options:    policies.options,
		policies:   policies,
		policy:     policy,
	}

	s.smoothing = smoothing.New(&policy.Spec.Smoothing)

	return s
}

func (s *PolicyState) updatePolicy(o *scalingpolicy.ScalingPolicy) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.policy = o
	s.smoothing = smoothing.UpdateRule(s.smoothing, &s.policy.Spec.Smoothing)
}

func (s *PolicyState) computeTargetValues(snapshot factors.Snapshot) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	glog.V(4).Infof("computing target values")

	podSpec, err := scaling.ComputeResources(snapshot, &s.policy.Spec)
	if err != nil {
		return err
	}

	glog.V(4).Infof("computed target values: %s", debug.Print(podSpec))

	s.smoothing.UpdateTarget(podSpec)
	return nil
}

func (s *PolicyState) updateValues() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	policy := s.policy
	client := s.kubeClient
	patcher := s.policies.patcher

	kind := policy.Spec.ScaleTargetRef.Kind
	namespace := policy.Namespace
	name := policy.Spec.ScaleTargetRef.Name

	path := fmt.Sprintf("%s/%s/%s", kind, namespace, name)

	changed := false
	var updates *v1.PodSpec
	switch strings.ToLower(kind) {
	case "replicaset":
		{
			kind = "ReplicaSet"
			o, err := client.ExtensionsV1beta1().ReplicaSets(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				// TODO: Emit event?
				return err
			}

			changed, updates = s.smoothing.ComputeChange(path, &o.Spec.Template.Spec)
		}

	case "daemonset":
		{
			kind = "DaemonSet"
			o, err := client.ExtensionsV1beta1().DaemonSets(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				// TODO: Emit event?
				return err
			}

			changed, updates = s.smoothing.ComputeChange(path, &o.Spec.Template.Spec)
		}

	case "deployment":
		{
			kind = "Deployment"
			o, err := client.AppsV1beta1().Deployments(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				// TODO: Emit event?
				return err
			}

			changed, updates = s.smoothing.ComputeChange(path, &o.Spec.Template.Spec)
		}

	default:
		return fmt.Errorf("unhandled kind: %q", kind)
	}

	if changed {
		if err := patcher.UpdateResources(kind, namespace, name, updates, s.options.DryRun); err != nil {
			glog.Warningf("failed to update %q: %v", kind, err)
		} else {
			glog.V(4).Infof("applied update to %s", path)
		}
	} else {
		glog.V(4).Infof("no change needed for %s", path)
	}

	return nil
}

type PolicyInfo struct {
	Policy *scalingpolicy.ScalingPolicy `json:"policy"`
	State  *http.Info                `json:"state"`
}

func (s *PolicyState) Query() *PolicyInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	info := &PolicyInfo{
		Policy: s.policy,
		State:  s.smoothing.Query(),
	}
	return info
}
