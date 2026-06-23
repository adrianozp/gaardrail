package setdisturbance

import (
	"time"

	"github.com/adrianozp/gaardrail/app/disturbance"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type Component interface {
	Set(query string, ratePerSecond float64, ttl time.Duration) error
	Get() disturbance.State
}

type UseCase struct {
	comp Component
}

func New(c Component) UseCase {
	return UseCase{comp: c}
}

func (u UseCase) Set(query string, ratePerSecond float64, ttl time.Duration) error {
	return u.comp.Set(query, ratePerSecond, ttl)
}

func (u UseCase) Get() disturbance.State {
	return u.comp.Get()
}
