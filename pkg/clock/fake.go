package clock

import "time"

type FakeClock struct {
	elapsed time.Duration
}

var _ Clock = &FakeClock{}

func (c *FakeClock) Nanos() time.Duration {
	return c.elapsed
}
