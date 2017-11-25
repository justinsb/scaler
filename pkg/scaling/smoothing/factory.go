package smoothing

import (
	api "github.com/justinsb/scaler/pkg/apis/scalingpolicy/v1alpha1"
)

func New(p *api.SmoothingRule) Smoothing {
	return UpdateRule(nil, p)
}

func UpdateRule(s Smoothing, p *api.SmoothingRule) Smoothing {
	if p.Percentile != nil {
		ps, ok := s.(*PercentileSmoothing)
		if ok {
			ps.updateRule(p.Percentile)
			return ps
		}
		return NewPercentileSmoothing(p.Percentile)
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
