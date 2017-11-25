package clock

import "time"

type Clock interface {
	Nanos() time.Duration
}
