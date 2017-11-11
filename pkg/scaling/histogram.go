package scaling

//
//import (
//	"k8s.io/apimachinery/pkg/api/resource"
//	"sync"
//	"github.com/golang/glog"
//	"sort"
//)
//
//type Histogram struct {
//	limit int
//
//	mutex      sync.Mutex
//	values     []int64
//	pos        int
//	lastFormat resource.Format
//}
//
//func (h *Histogram) Add(t int64, q resource.Quantity) {
//	qv, ok := q.AsInt64()
//	if !ok {
//		glog.Warningf("ignoring out-of-range value %q", q)
//		return
//	}
//
//	h.mutex.Lock()
//	defer h.mutex.Unlock()
//
//	h.lastFormat = q.Format
//
//	if len(h.values) < h.limit {
//		h.values = append(h.values, qv)
//	} else {
//		h.values[h.pos] = qv
//		h.pos++
//		if h.pos >= h.limit {
//			h.pos = 0
//		}
//	}
//}
//
//func (h *Histogram) Percentile(ratio float32) (resource.Quantity, bool) {
//	h.mutex.Lock()
//	defer h.mutex.Unlock()
//
//	if len(h.values) < minObservationsForPercentile {
//		return resource.Quantity{}, false
//	}
//
//	sorted := make([]int64, len(h.values))
//	copy(sorted, h.values)
//	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
//
//	index := int(float32(len(h.values)) * ratio)
//	if index >= len(h.values) {
//		index--
//	}
//
//	v := sorted[index]
//	return *resource.NewQuantity(v, h.lastFormat), true
//}
//
//func (h *Histogram) EstimatePercentile(value *resource.Quantity) (float32, bool) {
//	qv, ok := value.AsInt64()
//	if !ok {
//		glog.Warningf("ignoring out-of-range value %q", value)
//		return 0, false
//	}
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
//		if h.values[i] <= qv {
//			x++
//		}
//	}
//
//	return float32(x) / float32(n), true
//}
