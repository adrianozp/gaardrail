package control_test

import (
	"sync"
	"testing"

	"github.com/adrianozp/gaardrail/pkg/control"
)

func hammer(c control.Controller, retune func(i int)) {
	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() { _, _, _ = c.Compute(10, at(float64(i))) })
		wg.Go(func() { _ = c.SetSetpoint(float64(i)) })
		wg.Go(func() { retune(i) })
		wg.Go(c.Reset)
	}
	wg.Wait()
}

func TestPIDConcurrentUse(t *testing.T) {
	c := newPID(t, pidParams(1, 1, 1, 0, 100, 20, 15))
	hammer(c, func(i int) {
		p := c.Params()
		p.Kp = float64(i)
		_ = c.SetParams(p)
	})
}

func TestSmithConcurrentUse(t *testing.T) {
	c := newSmith(t, smithParams(1, 1, 0, 100, 20, 15, oneSampleDelay(), 10))
	hammer(c, func(i int) {
		p := c.Params()
		p.Model.Theta = float64(i % 3)
		_ = c.SetParams(p)
	})
}

func TestStepConcurrentUse(t *testing.T) {
	s := newStep(t, 5)
	hammer(s, func(i int) { _ = s.SetOutput(float64(i)) })
}
