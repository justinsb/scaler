

## The approach

The model has 3 main steps:

* We observe various inputs (number of nodes, sum of node cores, sum of node memory etc)
* ScalingPolicies explicitly transform those inputs into an "optimum" value for resource requests & limits
  for daemonsets / deployments / replicasets.  For example, cpu might be modelled as `200m + (coreCount * 10m)`.
* We can optionally apply Smoothing, so that oscillations (for example during a rolling-update) won't result in
  the targeted resources constantly being restarted.
* We can optionally apply Quantization, so that the final value will be a human sensible-value. i.e. 256m, not 253m.
