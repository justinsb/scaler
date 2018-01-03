/*
Copyright 2018 The Kubernetes Authors.

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
	v1alpha1 "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	scheme "github.com/justinsb/scaler/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ScalingPoliciesGetter has a method to return a ScalingPolicyInterface.
// A group's client should implement this interface.
type ScalingPoliciesGetter interface {
	ScalingPolicies(namespace string) ScalingPolicyInterface
}

// ScalingPolicyInterface has methods to work with ScalingPolicy resources.
type ScalingPolicyInterface interface {
	Create(*v1alpha1.ScalingPolicy) (*v1alpha1.ScalingPolicy, error)
	Update(*v1alpha1.ScalingPolicy) (*v1alpha1.ScalingPolicy, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.ScalingPolicy, error)
	List(opts v1.ListOptions) (*v1alpha1.ScalingPolicyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ScalingPolicy, err error)
	ScalingPolicyExpansion
}

// scalingPolicies implements ScalingPolicyInterface
type scalingPolicies struct {
	client rest.Interface
	ns     string
}

// newScalingPolicies returns a ScalingPolicies
func newScalingPolicies(c *ScalingpolicyV1alpha1Client, namespace string) *scalingPolicies {
	return &scalingPolicies{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the scalingPolicy, and returns the corresponding scalingPolicy object, and an error if there is any.
func (c *scalingPolicies) Get(name string, options v1.GetOptions) (result *v1alpha1.ScalingPolicy, err error) {
	result = &v1alpha1.ScalingPolicy{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("scalingpolicies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ScalingPolicies that match those selectors.
func (c *scalingPolicies) List(opts v1.ListOptions) (result *v1alpha1.ScalingPolicyList, err error) {
	result = &v1alpha1.ScalingPolicyList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("scalingpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested scalingPolicies.
func (c *scalingPolicies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("scalingpolicies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a scalingPolicy and creates it.  Returns the server's representation of the scalingPolicy, and an error, if there is any.
func (c *scalingPolicies) Create(scalingPolicy *v1alpha1.ScalingPolicy) (result *v1alpha1.ScalingPolicy, err error) {
	result = &v1alpha1.ScalingPolicy{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("scalingpolicies").
		Body(scalingPolicy).
		Do().
		Into(result)
	return
}

// Update takes the representation of a scalingPolicy and updates it. Returns the server's representation of the scalingPolicy, and an error, if there is any.
func (c *scalingPolicies) Update(scalingPolicy *v1alpha1.ScalingPolicy) (result *v1alpha1.ScalingPolicy, err error) {
	result = &v1alpha1.ScalingPolicy{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("scalingpolicies").
		Name(scalingPolicy.Name).
		Body(scalingPolicy).
		Do().
		Into(result)
	return
}

// Delete takes name of the scalingPolicy and deletes it. Returns an error if one occurs.
func (c *scalingPolicies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("scalingpolicies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *scalingPolicies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("scalingpolicies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched scalingPolicy.
func (c *scalingPolicies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ScalingPolicy, err error) {
	result = &v1alpha1.ScalingPolicy{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("scalingpolicies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
