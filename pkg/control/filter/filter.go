package filter

import (
	"fmt"
	"math"
	"sync"
)

type Kind string

const (
	None          Kind = "none"
	MovingAverage Kind = "moving_average"
	Exponential   Kind = "exponential"
)

type Spec struct {
	Kind Kind
	Size int
}

func (s Spec) Normalize() (Spec, error) {
	if s.Kind == "" {
		s.Kind = None
	}
	if s.Kind != None && s.Kind != MovingAverage && s.Kind != Exponential {
		return Spec{}, fmt.Errorf("filter: unknown type %q", s.Kind)
	}
	s.Size = max(s.Size, 1)
	return s, nil
}

type Signal struct {
	mu            sync.Mutex
	spec          Spec
	seedFromFirst bool
	seeded        bool
	window        *window
	state         float64
}

func NewReference(spec Spec) (*Signal, error) {
	return newSignal(spec, false)
}

func NewMeasurement(spec Spec) (*Signal, error) {
	return newSignal(spec, true)
}

func newSignal(spec Spec, seedFromFirst bool) (*Signal, error) {
	spec, err := spec.Normalize()
	if err != nil {
		return nil, err
	}
	return &Signal{spec: spec, seedFromFirst: seedFromFirst, window: newWindow(spec.Size)}, nil
}

func (s *Signal) Spec() Spec {
	return s.spec
}

func (s *Signal) Filter(x float64) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.spec.Kind == None {
		return x
	}
	if s.seedFromFirst && !s.seeded {
		s.seed(x)
		return x
	}
	if s.spec.Kind == MovingAverage {
		return s.window.push(x)
	}
	return s.smooth(x)
}

func (s *Signal) smooth(x float64) float64 {
	a := math.Exp(-1.0 / float64(s.spec.Size))
	s.state = a*s.state + (1-a)*x
	s.seeded = true
	return s.state
}

func (s *Signal) Seed(v float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seed(v)
}

func (s *Signal) seed(v float64) {
	s.state = v
	s.seeded = true
	s.window.fill(v)
}

func (s *Signal) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = 0
	s.seeded = false
	s.window.reset()
}
