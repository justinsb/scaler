## Avoiding repeated scaling (flapping)

Currently in kubernetes, whenever we change the resources limits/requests on a resource, the underlying
pods are restarted.  Thus there is a non-trivial cost to each change.  However, clusters can change in size
rapidly, for example during a rolling update where each node in turn will likely go NotReady for a short period.
We would like to avoid repeatedly restarting pods during these operations.

## Strategies

We currently have various strategies, and we are trying to evaluate their relative utility, importance
and simplicity during the alpha period:

* Each resource function can define `segments`, so that we will only change the target value every N units of input.
  This allows you to express the idea that at small cluster sizes it is import not to over-allocate scarce resources, but
  for bigger clusters we might trade-off some resource under-utilization to avoid repeated restarts.
  This can still lead to flapping at the boundaries.
* We can define a hysteresis offset on the input values, such that we only scale down if we are "out of range" by more than
  that input value.
* We can define a simple statistical smoothing function.  The model here is that we treat the target value as an estimate of
  the true underlying resource requirements, which actually varies continuously, and we can never practically model
  it exactly.  Instead we estimate the resource limits periodically, and we use that as an estimate of the true optimal limits.
  When we estimate that the applied limits are not likely to be correct, we update the limits.
* We can define quantization on the output values (we could round each memory value to the nearest 32MiB, for example).
  This can still lead to flapping at the boundaries.  It does avoid values that are awkward for humans, like 107MiB.

## Segment model

For each input function, we define directly how often we want to scale based on the input values.

```
container:
  name: kube-dns
  ...
  requests:
    resource: cpu
    functions:
      input: Cores
      base: 100m
      slope: 10m # per core
      segments:
      - at: 0
        every: 1
      - at: 16
        every: 4
```

The idea here is that we have modelled cpu as being `(100m + 10m * cores)`.  We want to scale accurately for smaller
clusters, thus from 0 we scale `every: 1` core.  But at bigger cluster sizes, we can tolerate some resource over-allocation in return
for not scaling as often, so `at: 16` cores we only scale `every: 4` cores.  Thus at 18 cores, we round up to 20 cores,
and we allocate `100m + 10m * 20` = `300m` 

This has the advantage of being very simple to understand, and explicit.  It does still exhibit flapping
around boundaries (20 vs 21 cores), so it is probably not sufficient on its own.

## Hysteresis offset

The hysteresis offset allows us to define a tolerance or delay before scaling-up or scaling-down.  We express it
in terms of the input values.

```
  smoothing:
    delayScaleDown:
      inputs:
        cores: 20
```

This means that we delay scaling-down until our calculated value is more than 20 cores from the current value.  So in practice
we recompute the target value at `cores + 20`,  and we scale down only when we go under that value.

A worked example:

* Suppose our model is `10m * cores`
* The cluster has been steady-state at 100 cores, so we have set the resources as 1000m
* We reduce the cluster size to 90 cores, so the computed target is now 900m.  But the offset value is `10m * (90 cores + 20 core offset) == 1100m`,
  which is greater than the current value, and we don't scale down yet.
* If our cluster goes back up to 95 cores, our computed target is 950m, the offset value is 1150m, we still don't scale down
  and we've successfully avoided a re-scaling.
* If our cluster now reduces to 75 cores, the computed target is now 750m.  The offset value is 950m, which is less than the value of 1000m,
  and so we scale down to 750m.

We currently never delay scaling up (though we could easily add that).

The biggest challenge here is that if we reduce from 100 to 90 cores above, we will never scale down to the new target value.
We likely need to introduce some time decay function.  And that is likely going to be the sliding window model / percentage
estimation model...

## Percentage estimation model

The percentage estimation model in practice "just works", but it can be difficult to think through.

We assume that there is some true resource setting that varies within each time quantum.  The time quantum is on the
order of milliseconds, and depends on a large number of factors - some of which are outside the cluster - so we can't expect to perfectly model it, particularly with a simple model that is based on aggregate metrics like nodes & core counts.

Instead, we embrace the idea that our computations are themselves just estimates of the true resource requirements, and use
this to avoid rapid re-scaling.

The current model estimates the underlying resource limits by keeping a 5 minute window and assuming that the statistical
distribution of values that we observe in that window _is_ the distribution of the underlying value.  This is technically
called non-parametric statistics (I believe).

In classical statistics we would typically argue that the values are normally distributed,
we would compute the mean and the variance of the values, and we would estimate the 95% confidence bound as mean +/- 2 sigma.
In non-parametric statistics, we avoid assigning a distribution entirely, and we simply say that if half our observations were
less than X and half greater than X, then X is our estimate of the median.  This can be extended for any probability value:
the threshold value for 8 of every 10 observations is our estimation of the 80% probability.

To use this in a ScalingPolicy must simply define our target range and an acceptable range, in terms of probability.  Consider this policy:

```
  smoothing:
    percentile:
      target: 0.80
      lowThreshold: 0.60
      highThreshold: 0.95
```

Our target is to apply resources at the 80% estimate - thus we expect that 80% of the time the resources will be greater than or equal
to the underlying true value, and only 20% of the time too low.  But the 80% value is still an estimate, and would still change
rapidly.  There is nothing particularly special about the 80% value, nor do we believe that the 80% estimate is itself
accurate: it is only an estimate.  So to avoid flapping we define the thresholds - in terms of confidence - where we will
apply changes.

With a highThreshold of 95%, if we have 95% confidence that the true value is less than our current value, we will scale down; in
practice this means that if we haven't seen a bigger estimate than our current value for at least 1 of every 20 observations, then we scale down.

With a lowThreshold of 60%, if we have 60% confidence that the true value is greater than our current value, we will scale
up.  In practice this means that if we've seen a bigger estimate than our current value for at least 8 of every 20 observations, then we scale up.

Aiming for 80% (rather than 50%) means we will err on the side of over-provisioning.  Setting the thresholds at 60% and 95%,
and not centered on 80%, means that we scale up faster than we scale down.

A more intuitive way (but less correct) way to express this is "we'll aim to be at a value that is big enough 80% of the time.  If we've been too low 40% of the time, scale up.  If we've been too high 95% of the time, scale down."

This has some nice properties:

* We don't assume anything about the underlying distribution - the normal distribution is often correct in nature, but
  can be badly incorrect for skewed distributions that we see in computer systems.
* It is computationally simple: we record a vector of the N most recent observations, sort and lookup the value for any
  desired fraction.
* It doesn't depend at all on the input method we are using; it is purely a function of the final estimates.  We can
  get those values from a neural network and still smooth them appropriately.
* It is similar to a moving average when plotted on a graph.
* It converges - if each observation is the same value X, the final estimate will be X.  For cluster proportional scaling,
  we expect this to be the case - hours and hours of boredom punctuated by minutes of rapid change.

I believe this model could replace both segments and the hysteresis offset.  The quantization would only be needed if we wanted
our humans to see "nice round values".

In practice, we could sensibly default lowThreshold & highThreshold and probably target also.

## Output quantization

```
container:
  name: kube-dns
  ...
  quantization:
    resource: cpu
    base: 100m
```

This quantization means that the final cpu resource values will always be rounded up to the nearest 0.1 cores.  This does help to avoid re-scaling (though it fails at the boundaries), but also means that no matter how complicated our functions, we won't ever end up with odd fractional values.
Kubernetes doesn't really care, but it can be hard for humans to read.

The odd "base" syntax is because we actually support non-constant rounding.  Thus you likely want to snap to every 0.1 core
when the value is less than 1 core, but once you're up to 10 cores, you probably can snap to every core.  Currently this uses exponentially growing steps, but if we want this we should switch this to use segments as defined on the function rules.

