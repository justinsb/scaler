package static

import (
	"github.com/justinsb/scaler/pkg/factors"
)

type staticFactors struct {
	values map[string]float64
}

var _ factors.Interface = &staticFactors{}

type staticFactorsSnapshot struct {
	values map[string]float64
}

var _ factors.Snapshot = &staticFactorsSnapshot{}

func NewStaticFactors(values map[string]float64) factors.Interface {
	p := &staticFactors{
		values: values,
	}
	return p
}

func (k *staticFactors) Snapshot() (factors.Snapshot, error) {
	return &staticFactorsSnapshot{
		values: k.values,
	}, nil
}

func (s *staticFactorsSnapshot) Get(key string) (float64, bool, error) {
	v, found := s.values[key]
	return v, found, nil
}
