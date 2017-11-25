package clock

import "time"

type WallClock struct {
	baseTime time.Time
}

var _ Clock = &WallClock{}

func (c *WallClock) Nanos() time.Duration {
	now := time.Now()
	return now.Sub(c.baseTime)
}
