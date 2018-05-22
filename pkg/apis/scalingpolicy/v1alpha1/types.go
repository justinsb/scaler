/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	autoscaling "k8s.io/api/autoscaling/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScalingPolicy is a specification for an ScalingPolicy resource
type ScalingPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ScalingPolicySpec   `json:"spec"`
	Status ScalingPolicyStatus `json:"status,omitempty"`
}

// ScalingPolicySpec is the spec for an ScalingPolicy resource
type ScalingPolicySpec struct {
	// This is what HPA uses but I don’t love it

	// reference to scaled resource; horizontal pod autoscaler will learn the current resource consumption
	// and will set the desired number of pods by using its Scale subresource.
	ScaleTargetRef autoscaling.CrossVersionObjectReference `json:"scaleTargetRef"`

	Containers []ContainerScalingRule `json:"containers" patchStrategy:"merge"`
}

type DelayScaling struct {
	// Max is the input value skew we tolerate in the output value
	Max float64 `json:"max,omitempty"`

	// DelaySeconds is the delay before we scale down
	DelaySeconds int32 `json:"delaySeconds,omitempty"`
}

// ContainerScalingRule defines how container resources are scaled
type ContainerScalingRule struct {
	// Name of the container specified as a DNS_LABEL.
	// Each container in a pod must have a unique name (DNS_LABEL).
	// Cannot be updated.
	Name string `json:"name"`

	// Compute Resources required by this container.
	// cf Container resources
	// +optional
	Resources ResourceRequirements `json:"resources,omitempty"`
}

// ResourceRequirements holds the functions for resource limits & requests
// TODO: Should we just embed this in the parent?
type ResourceRequirements struct {
	// Limits describes the maximum amount of compute resources allowed.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	// +optional
	Limits []ResourceScalingRule `json:"limits,omitempty"`
	// Requests describes the minimum amount of compute resources required.
	// If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
	// otherwise to an implementation-defined value.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	// +optional
	Requests []ResourceScalingRule `json:"requests,omitempty"`
}

type ResourceScalingRule struct {
	// Resource is the name of the resource we are scaling
	Resource v1.ResourceName `json:"resource"`

	// Function defines how the target resource usage
	// depends on a set of input values (such as cluster core count, number of nodes etc)
	Function ResourceScalingFunction `json:"function"`

	// Max limits the maximum computed value of the resource.
	// If the value computed is greater than Max, we will use Max instead
	Max resource.Quantity `json:"max,omitempty"`
}

type ResourceScalingFunction struct {
	// Input is the source value to use as the input to scaling: `cores`, `memory`, `nodes`
	Input string `json:"input,omitempty"`

	// Base is the constant resource value we use regardless of input, the y-axis intercept
	Base resource.Quantity `json:"base,omitempty"`

	// Slope determines how fast the resource usage changes per unit of input.
	// For each Input unit, we increase resources by Slope
	Slope resource.Quantity `json:"slope,omitempty"`

	// Per divides Input before multiplying by Slope, allowing us to specify slopes of < 1m per input unit
	Per int32 `json:"int,omitempty"`

	// Segments defines a set of segments of the resource line.
	// In each segment we define the interval with which we change values.
	// This is typically used so that we resize for every input unit for small cluster,
	// but for larger clusters we only resize for changes of N units or more.
	// Where it is not otherwise defined, we assume a first value of { at: 0, every: 1 }
	Segments []ResourceScalingSegment `json:"segments,omitempty"`

	DelayScaleDown *DelayScaling `json:"delayScaleDown,omitempty"`
}

// ResourceScalingSegment describes a segment of input values and the rounding policy we apply to it
type ResourceScalingSegment struct {
	// The segment applies to values greater than or equal to at.  The "closest" segment is selected
	At int64 `json:"at,omitempty"`

	// Every specifies the granularity to which we round.  We always round up to the next multiple of Every.
	Every int64 `json:"every,omitempty"`
}

// ScalingPolicyStatus is the status for an ScalingPolicy resource
type ScalingPolicyStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScalingPolicyList is a list of ScalingPolicy resources
type ScalingPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ScalingPolicy `json:"items"`
}
