package control

import (
	"sync"
	"time"
)

type Step struct {
	mu     sync.RWMutex
	output float64
}

func NewStep(output float64) (*Step, error) {
	if !isFinite(output) {
		return nil, ErrInvalidOutput
	}
	return &Step{output: output}, nil
}

func (s *Step) Compute(measured float64, _ time.Time) (float64, Terms, error) {
	out := s.Output()
	return out, Terms{Measured: measured, Output: out}, nil
}

func (s *Step) Output() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.output
}

func (s *Step) SetOutput(output float64) error {
	if !isFinite(output) {
		return ErrInvalidOutput
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.output = output
	return nil
}

func (s *Step) SetSetpoint(float64) error {
	return nil
}

func (s *Step) Reset() {}
