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
	ScaleTargetRef autoscaling.CrossVersionObjectReference `json:"scaleTargetRef""`

	Containers []ContainerScalingRule `json:"containers" patchStrategy:"merge"`

	// TODO: Should this be at the ContainerScalingRule level?
	Smoothing SmoothingRule `json:"smoothing,omitempty"`
}

type QuantizationRule struct {
	Resource v1.ResourceName `json:"resource"`

	Base      resource.Quantity `json:"base,omitempty"`
	Step      resource.Quantity `json:"step,omitempty"`
	StepRatio float32           `json:"stepRatio,omitempty"`
	MaxStep   resource.Quantity `json:"maxStep,omitempty"`
}

type PercentileSmoothing struct {
	// TODO: How should we represent percentages?

	Target        float32 `json"target,omitempty"`
	LowThreshold  float32 `json"lowThreshold,omitempty"`
	HighThreshold float32 `json"highThreshold,omitempty"`
}

type ShiftSmoothing struct {
	Inputs map[string]float64 `json:"inputs,omitempty"`
}

type SmoothingRule struct {
	Percentile     *PercentileSmoothing `json:"percentile,omitempty"`
	ScaleDownShift *ShiftSmoothing      `json:"scaleDownShift,omitempty"`
}

// ScalingRule defines how container resources are scaled
type ContainerScalingRule struct {
	// Name of the container specified as a DNS_LABEL.
	// Each container in a pod must have a unique name (DNS_LABEL).
	// Cannot be updated.
	Name string `json:"name"`

	// Compute Resources required by this container.
	// cf Container resources
	// +optional
	Resources ResourceRequirements `json:"resources,omitempty"`

	Quantization []QuantizationRule `json:"quantization,omitempty"`
}

// ResourceScaling configures
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
	// Input is the source value to use as the input to scaling
	Input string `json:"input,omitempty"`

	Resource v1.ResourceName `json:"resource"`

	Base  resource.Quantity `json:"base,omitempty"`
	Slope resource.Quantity `json:"slope,omitempty"`
	Max   resource.Quantity `json:"max,omitempty"`

	Segments []ResourceScalingSegment `json:"segments,omitempty"`
}

// ResourceScalingSegment describes a segment of input values and the rounding policy we apply to it
type ResourceScalingSegment struct {
	// The segment applies to values greater than or equal to at.  The "closest" segment is selected
	At float64 `json:"at,omitempty"`

	// RoundTo specifies the granularity to which we round.  We always round up to the next multiple of roundTo.
	RoundTo float64 `json:"roundTo,omitempty"`
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
