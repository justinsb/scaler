package smoothing

import (
	api "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
	"github.com/justinsb/scaler/pkg/timeutil"
)

func New(clock *timeutil.MonotonicClock, p *api.SmoothingRule) Smoothing {
	return UpdateRule(clock, nil, p)
}

func UpdateRule(clock *timeutil.MonotonicClock, s Smoothing, p *api.SmoothingRule) Smoothing {
	if p.Percentile != nil {
		ps, ok := s.(*PercentileSmoothing)
		if ok {
			ps.updateRule(p.Percentile)
			return ps
		}
		return NewPercentileSmoothing(clock, p.Percentile)
	} else if p.ScaleDownShift != nil {
		rs, ok := s.(*ResourceShiftSmoothing)
		if ok {
			rs.updateRule(p.ScaleDownShift)
			return rs
		}
		return NewResourceShiftSmoothing(p.ScaleDownShift)
	} else {
		noop, ok := s.(*NoopSmoothing)
		if ok {
			// No update needed
			return noop
		}
		return NewNoop()
	}
}
