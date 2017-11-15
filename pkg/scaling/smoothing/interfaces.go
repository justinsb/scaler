package smoothing

import (
	"k8s.io/api/core/v1"
	"github.com/justinsb/scaler/pkg/http"
)

// Smoothing allows a hysteresis strategy to be plugged in
// The problem we are solving:
//  the inputs may change rapidly or oscillate - for example during a rolling update we might expect to see N / N+1 nodes oscillating.
// Quantization makes it less likely that this will result in target value oscillation, but if we are unlucky with N
// we will still oscillate.  (Quantization is more so we have human-readable values).
type Smoothing interface {
	UpdateTarget(podSpec *v1.PodSpec)

	ComputeChange(parentPath string, current *v1.PodSpec) (bool, *v1.PodSpec)

	Query() *http.Info
}
