# Cluster-Proportional Vertical Pod Scaler for Kubernetes

The CPVPS sets resource requirements on a target resource (Deployment / DamonSet / ReplicaSet).
As compared to alternatives it is intended to be API driven, based on simple deterministic
functions of cluster state (rather than on metrics), and to have no external dependencies so it
can be used for bootstrap cluster components (like kube-dns).

## The approach

To compute the resources for a target:

* We observe various inputs (number of nodes, sum of node cores, sum of node memory etc)
* ScalingPolicies explicitly transform those inputs into an "optimum" value for resource requests & limits
  for daemonsets / deployments / replicasets.  For example, cpu might be modelled as `200m + (coreCount * 10m)`.
* We can optionally apply Smoothing, so that oscillations (for example during a rolling-update) won't result in
  the targeted resources "flapping".
* We can optionally apply Quantization, so that the final value will be a human sensible-value. i.e. 256m, not 253m.

We repeat this operation at a regular interval (e.g. every 10 seconds) to compute the target values.  We run
a second periodic task which applies the updated resources whenever they are out of date.  Running two loops allows
for high-resolution collection of input data, without forcing pods to potentially restart at the same frequency.

TODO: Allow per-pod intervals?

## The ScalingPolicy schema

The ScalingPolicy schema reflects this approach, and is also supposed to feel similar to the PodSpec schema.

A ScalingPolicy object targets a single deployment / replicaset / daemonset.

There is a list of containers, each of which can have resource limits & requests.  Where Pods have 
resources directly specified in a map, a ScalingPolicy has a list of resource rules, which specify an
input value and apply a function to produce the target value for the resource.

It is possible to have multiple rules targeting the same container resource.  The values will be added.

TODO: The intent here is to allow for growth based on e.g. endpoints _and_ pods.  Is this useful?  Should we add
or do max?

Each resource can be quantized with a quantization policy.  Quantization is primarily to "snap" values
to more convenient multiples for humans (e.g. steps of 50m CPU), but also helps to avoid flapping.

The smoothing policy is specified at ScalingPolicy level.  If not specified, the default is no smoothing, so the resulting
value will be applied to the pod whenever the target differs from the actual value, which can result in rapid pod bouncing.
Quantization helps, but there are still transitional values where the output will flap.

An alternative smoothing policy is supported, which tracks the target value over a sliding window.  The smoothing policy
can specify a target percentile value (default 80%), along with a tolerable range (defaults to 70%-90%).  The target resources
will only be updated when the actual value is outside of the tolerable range.

// TODO: Need better names for the computed target value vs the actual resources of the target.

// TODO: Make sliding window time configurable.  Other policies?

// TODO: Publish prometheus metrics so these calculations are very visible and it is easy for operators to determine the correct policy

## Example:

```
apiVersion: scalingpolicy.kope.io/v1alpha1
kind: ScalingPolicy
metadata:
  name: kube-dns
  namespace: kube-system
spec:
  scaleTargetRef:
    kind: deployment
    name: kube-dns
  containers:
  - name: dns
    resources:
      limits:
      - resource: cpu
        base: 200m
        input: cores
        step: 10m
      requests:
      - resource: cpu
        base: 100m
    quantization:
    - resource: cpu
      step: 10m
      stepRatio: 2.0
      maxStep: 1000m
  smoothing:
    percentile:
      target: 0.80
      lowThreshold: 0.70
      highThreshold: 0.90
```

# Operator configurations

We expect that system add-ons will ship with a default ScalingPolicy.  We also expect that they will
not be optimal for every scenario, and that operators will want to modify them.

Inspired by bgrant's proposal for addon/configuration management, we adopt the idea of "overlays" / "kubectl apply"
style configuration merging.  At some stage we expect this will be done by a future addon manager, but until
then we aim to adopt the same process & combination/override rules that will be part of the addon manager,
but implemented instead by the scaler itself.

We allow multiple ScalingPolicies to target the same resource (e.g. deployment).  We produce a merged ScalingPolicy
for each target.  We do a union of all rules, except that a rule with the same (container, limit vs request, resource, name)
will replace the existing rule.  This allows for rules to be additive but also allows easy reconfiguration of
formula.

Names on rules are optional, but we expect system add-ins will define names, will not change the names of their rulesets,
and will likely develop a convention for names.

We also define a "priority" field, which determines the order in which policies will be merged.

// TODO: Examples of merging

// TODO: Implement this!