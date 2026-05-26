// Package switchable wraps the available controllers and lets the active one be
// swapped at runtime. It implements controllers.Controller by delegating every
// call to the currently selected controller, so consumers never know the
// underlying implementation changed. The switch logic and all edge cases live
// here, keeping use cases and FX modules free of branching.
package switchable

import (
	"fmt"
	"sync"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/pid"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/step"
	"github.com/adrianozp/gaardrail/pkg/config"
)

// Controller delegates to whichever underlying controller is currently active.
// Each underlying instance stays alive across switches and keeps its own params.
type Controller struct {
	mu         sync.RWMutex
	active     controllers.Controller
	activeType string
	available  map[string]controllers.Controller
}

// New builds every available controller and selects the one named by
// cfg.Controller.Type. Returns an error for an unknown type.
func New(cfg config.Config) (*Controller, error) {
	available := map[string]controllers.Controller{
		"pid":  pid.New(cfg),
		"step": step.NewStep(cfg),
	}

	active, ok := available[cfg.Controller.Type]
	if !ok {
		return nil, fmt.Errorf("controller: unknown type %q", cfg.Controller.Type)
	}

	return &Controller{
		active:     active,
		activeType: cfg.Controller.Type,
		available:  available,
	}, nil
}

func (c *Controller) current() controllers.Controller {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.active
}

func (c *Controller) Compute(measured float64, measureTime time.Time) (float64, error) {
	return c.current().Compute(measured, measureTime)
}

func (c *Controller) GetParams() entities.ControllerParams {
	return c.current().GetParams()
}

func (c *Controller) SetParams(p entities.ControllerParams) error {
	return c.current().SetParams(p)
}

func (c *Controller) SetSetpoint(setpoint float64) error {
	return c.current().SetSetpoint(setpoint)
}

func (c *Controller) Reset() {
	c.current().Reset()
}

func (c *Controller) Type() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activeType
}

// SetType swaps the active controller. The newly activated controller is reset
// so stale integral/derivative state does not leak across a switch. Switching
// to the already-active type is a no-op. Unknown types return an error.
func (c *Controller) SetType(t string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if t == c.activeType {
		return nil
	}

	next, ok := c.available[t]
	if !ok {
		return fmt.Errorf("controller: unknown type %q", t)
	}

	next.Reset()
	c.active = next
	c.activeType = t
	return nil
}
