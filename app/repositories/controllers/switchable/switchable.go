package switchable

import (
	"fmt"
	"sync"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/repositories/controllers"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/pid"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/smith"
	"github.com/adrianozp/gaardrail/app/repositories/controllers/step"
	"github.com/adrianozp/gaardrail/pkg/config"
)

type Controller struct {
	mu         sync.RWMutex
	active     controllers.Controller
	activeType string
	available  map[string]controllers.Controller
}

type factory func(config.Config) (controllers.Controller, error)

var factories = map[string]factory{
	"pid":   func(cfg config.Config) (controllers.Controller, error) { return pid.New(cfg, false) },
	"pidff": func(cfg config.Config) (controllers.Controller, error) { return pid.New(cfg, true) },
	"step":  func(cfg config.Config) (controllers.Controller, error) { return step.New(cfg) },
	"smith": func(cfg config.Config) (controllers.Controller, error) { return smith.New(cfg) },
}

func New(cfg config.Config) (*Controller, error) {
	available, err := buildAll(cfg)
	if err != nil {
		return nil, err
	}
	active, ok := available[cfg.Controller.Type]
	if !ok {
		return nil, fmt.Errorf("controller: unknown type %q", cfg.Controller.Type)
	}
	return &Controller{active: active, activeType: cfg.Controller.Type, available: available}, nil
}

func buildAll(cfg config.Config) (map[string]controllers.Controller, error) {
	available := make(map[string]controllers.Controller, len(factories))
	for name, build := range factories {
		c, err := build(cfg)
		if err != nil {
			return nil, fmt.Errorf("controller %s: %w", name, err)
		}
		available[name] = c
	}
	return available, nil
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

func (c *Controller) Reset() {
	c.current().Reset()
}

func (c *Controller) Type() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activeType
}

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
