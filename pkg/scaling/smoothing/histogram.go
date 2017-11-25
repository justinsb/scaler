package smoothing

import (
	"sort"
	"sync"
	"time"

	"github.com/justinsb/scaler/pkg/http"
	"github.com/justinsb/scaler/pkg/timeutil"
	"k8s.io/apimachinery/pkg/api/resource"
)

type dataPoint struct {
	time  time.Duration
	value int64
}

type Histogram struct {
	Limit int
	Scale resource.Scale

	mutex      sync.Mutex
	values     []dataPoint
	pos        int
	lastFormat resource.Format
}

func (h *Histogram) Add(t time.Duration, q resource.Quantity) {
	qv := q.ScaledValue(h.Scale)

	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.lastFormat = q.Format

	if len(h.values) < h.Limit {
		h.values = append(h.values, dataPoint{time: t, value: qv})
	} else {
		h.values[h.pos] = dataPoint{time: t, value: qv}
		h.pos++
		if h.pos >= h.Limit {
			h.pos = 0
		}
	}
}

// Percentile returns the value at the given percentile
// TODO: Multi-fetch, or store in a way that this isn't expensive
func (h *Histogram) Percentile(ratio float32) (resource.Quantity, bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if len(h.values) < minObservationsForPercentile {
		return resource.Quantity{}, false
	}

	sorted := make([]int64, len(h.values))
	for i := range h.values {
		sorted[i] = h.values[i].value
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	index := int(float32(len(h.values)) * ratio)
	if index >= len(h.values) {
		index--
	}

	v := sorted[index]
	r := resource.NewScaledQuantity(v, h.Scale)
	r.Format = h.lastFormat
	return *r, true
}

//func (h *Percentile) EstimatePercentile(value *resource.Quantity) (float32, bool) {
//	qv := value.ScaledValue(h.Scale)
//
//	h.mutex.Lock()
//	defer h.mutex.Unlock()
//
//	if len(h.values) < minObservationsForPercentile {
//		return 0, false
//	}
//
//	n := len(h.values)
//	if n == 0 {
//		return 0, false
//	}
//
//	x := 0
//	for i := 0; i < n; i++ {
//		if h.values[i].value <= qv {
//			x++
//		}
//	}
//
//	return float32(x) / float32(n), true
//}

func (h *Histogram) Query(clock *timeutil.MonotonicClock) *http.HistogramInfo {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	info := &http.HistogramInfo{}

	for i := h.pos; i < len(h.values); i++ {
		info.Data = append(info.Data, http.HistogramDataPoint{
			Time:  clock.ToUnix(h.values[i].time),
			Value: h.values[i].value,
		})
	}
	for i := 0; i < h.pos; i++ {
		info.Data = append(info.Data, http.HistogramDataPoint{
			Time:  clock.ToUnix(h.values[i].time),
			Value: h.values[i].value,
		})
	}
	return info
}
