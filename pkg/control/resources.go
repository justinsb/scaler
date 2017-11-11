package control

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/justinsb/scaler/cmd/scaler/options"
	scalingpolicy "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/debug"
	"github.com/justinsb/scaler/pkg/factors"
	"github.com/justinsb/scaler/pkg/scaling"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PolicyState struct {
	kubeClient kubernetes.Interface
	options    *options.AutoScalerConfig

	policies  *policies
	policy    *scalingpolicy.ScalingPolicy
	smoothing *scaling.Unsmoothed
}

func NewPolicyState(policies *policies, policy *scalingpolicy.ScalingPolicy) *PolicyState {
	s := &PolicyState{
		kubeClient: policies.client,
		options:    policies.options,
		policies:   policies,
		policy:     policy,
	}

	s.smoothing = scaling.NewUnsmoothed()
	return s
}

func (c *PolicyState) updatePolicy(o *scalingpolicy.ScalingPolicy) {
	c.policy = o
}

func (c *PolicyState) computeTargetValues(snapshot factors.Snapshot) error {
	glog.V(4).Infof("computing target values")

	podSpec, err := scaling.ComputeResources(snapshot, &c.policy.Spec)
	if err != nil {
		return err
	}

	glog.V(4).Infof("updated target values: %s", debug.Print(podSpec))

	c.smoothing.UpdateTarget(podSpec)
	return nil
}

func (c *PolicyState) updateValues() error {
	policy := c.policy
	client := c.kubeClient
	patcher := c.policies.patcher

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

			changed, updates = c.smoothing.ComputeChange(path, &o.Spec.Template.Spec)
		}

	case "daemonset":
		{
			kind = "DaemonSet"
			o, err := client.ExtensionsV1beta1().DaemonSets(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				// TODO: Emit event?
				return err
			}

			changed, updates = c.smoothing.ComputeChange(path, &o.Spec.Template.Spec)
		}

	case "deployment":
		{
			kind = "Deployment"
			o, err := client.AppsV1beta1().Deployments(namespace).Get(name, meta_v1.GetOptions{})
			if err != nil {
				// TODO: Emit event?
				return err
			}

			changed, updates = c.smoothing.ComputeChange(path, &o.Spec.Template.Spec)
		}

	default:
		return fmt.Errorf("unhandled kind: %q", kind)
	}

	if changed {
		if err := patcher.UpdateResources(kind, namespace, name, updates, c.options.DryRun); err != nil {
			glog.Warningf("failed to update %q: %v", kind, err)
		} else {
			glog.V(4).Infof("applied update to %s", path)
		}
	}

	return nil
}
