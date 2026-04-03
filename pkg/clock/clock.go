package clock

import (
	"time"
)

var Clock func() time.Time

func Now() time.Time {
	if Clock != nil {
		return Clock()
	}
	return time.Now()
}

func With(f func() time.Time) {
	Clock = f
}

func WithTime(t time.Time) {
	Clock = func() time.Time { return t }
}
