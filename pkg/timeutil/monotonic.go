package timeutil

import (
	"time"

	"k8s.io/apimachinery/pkg/util/clock"
)

type MonotonicClock struct {
	baseTime time.Time
	clock    clock.Clock
}

func NewMonotonicClock(clock clock.Clock) *MonotonicClock {
	c := &MonotonicClock{
		baseTime: clock.Now(),
		clock:    clock,
	}
	return c
}

func (m *MonotonicClock) Nanos() time.Duration {
	t := m.clock.Now()
	return t.Sub(m.baseTime)
}

func (m *MonotonicClock) ToUnix(nanos time.Duration) int64 {
	return m.baseTime.Add(nanos).Unix()
}
