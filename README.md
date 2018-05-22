# Cluster-Proportional Vertical Pod Scaler for Kubernetes

The CPVPS sets resource requirements on a target resource (Deployment / DamonSet / ReplicaSet).
As compared to alternatives it is intended to be API driven, based on simple deterministic
functions of cluster state (rather than on metrics), and to have no external dependencies so it
can be used for bootstrap cluster components (like kube-dns).

## The approach

To compute the resources for a target:

* We observe various inputs (number of nodes, sum of node cores, sum of node memory etc)
* ScalingPolicies explicitly transform those inputs into an "optimum" value for resource requests & limits
  for daemonsets / deployments / replicasets.  For example, cpu might be modelled as `200m + (cores * 10m)`.
  We don't support expressions via a DSL, but rather via fields in the API (e.g. `base: 200m`, `input: cores`, `slope: 10m`).
* When we must restart the target pods when we change the resources, so we try to avoid scaling them for every input change.
  We would like to avoid oscillations (for example during a rolling-update) resulting in
  rapid-rescaling the targeted resources.
* The first way we avoid rapid-rescaling is by defining `segments`; which round the input value up to a multiple
  of an interval called `every`.  Each segment applies to a different range of input values, and this lets
  us express the idea that at small scale we care about every core, but at larger scale we probably value
  avoiding restarts more than having the optimal value.
* The second way we avoid rapid-rescaling is that we delay scaling down - either by enforcing
  a time delay before scaling down, or by tolerating resource values that are higher than our computed
  values - or both.  We currently always scale up immediately.

We repeat this operation at a regular interval (e.g. every 10 seconds) to compute the target values.  We run
a second periodic task which applies the updated resources whenever they are out of date.  Running two loops allows
for high-resolution collection of input data, without forcing pods to potentially restart at the same frequency.

TODO: Allow per-pod intervals?

## The ScalingPolicy schema

The ScalingPolicy schema reflects this approach, and is also supposed to feel similar to the PodSpec schema.

A ScalingPolicy object targets a single deployment / replicaset / daemonset.

There is a list of containers, each of which can have resource limits & requests.  Where Pods have 
resources directly specified in a map, a ScalingPolicy has a list of resource rules, which specify an
the target resource and the input function which produce the target value for the resource from an input
- currently `cores` `memory` or `nodes`.

The scaling function is defined by a `base` value, and then a `slope` which multiples an `input` value.
So `200m + (cores * 10m)` maps to `base: 200m`, `input: cores`, `slope: 10m`.  To allow for a slope
of less than 1m per input value, we also define a field `per` which divides the `input`.

We have input `segments` which start `at` a particular input value, and then round
the input value to the next multiple of `every`.

We also have a `delayScaleDown` block which lets us specify the `delaySeconds` we will delay before scaling down,
and the `max` input skew we tolerate in the output value.  As an example, with our
function of `200m + (cores * 10m)` the target would be 280m, so if the resource on the target was more than 280m
we would scale down - possibly after `delaySeconds`.  If the `max` was 2, we would only scale down if the target was
more than 300m (`200m + ((8 + 2) * 10m)`).

// TODO: At & Every don't work for values like 2G for total memory - they're both integers.  Nor does Per.  Make them resources?  Define memory in MB?

// TODO: Need better names for the computed target value vs the actual resources of the target.

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
        function:
          base: 200m
          input: cores
          slope: 10m
          segments:
          - at: 4
            every: 4
          - at: 32
            every: 8
          - at: 256
            every: 64
      requests:
      - resource: cpu
        function:
          base: 100m
          delayScaling:
            delaySeconds: 30
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

// TODO: Implement this - or just use kustomize!
