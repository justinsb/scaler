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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
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
	ScaleTargetRef autoscaling.CrossVersionObjectReference `json:"scaleTargetRef" protobuf:"bytes,1,opt,name=scaleTargetRef"`

	Foo            string `json:"foo"`
	Bar            bool   `json:"bar"`
	DeploymentName string `json:"deploymentName"`
	Replicas       *int32 `json:"replicas"`
}

// ScalingPolicyStatus is the status for an ScalingPolicy resource
type ScalingPolicyStatus struct {
	State             ScalingPolicyState `json:"state,omitempty"`
	Message           string             `json:"message,omitempty"`
	AvailableReplicas int32              `json:"availableReplicas"`
}

type ScalingPolicyState string

const (
	ScalingPolicyStateCreated   ScalingPolicyState = "Created"
	ScalingPolicyStateProcessed ScalingPolicyState = "Processed"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ScalingPolicyList is a list of ScalingPolicy resources
type ScalingPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ScalingPolicy `json:"items"`
}
