package processmetrics

import (
	"errors"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type Controller interface {
	Compute(measured float64, measureTime time.Time) (float64, error)
}

type Orchestrator interface {
	SetDrainRate(float64) error
}

type ProcessMetricsUseCase struct {
	controller   Controller
	orchestrator Orchestrator
}

func NewProcessMetricsUseCase(c Controller, o Orchestrator) ProcessMetricsUseCase {
	return ProcessMetricsUseCase{
		controller:   c,
		orchestrator: o,
	}
}

func (u ProcessMetricsUseCase) Process(m entities.Metrics) error {
	cpuPercentage, ok := m.Metrics["cpu"]
	if !ok {
		return errors.New("cpu metric not found")
	}

	drainRate, err := u.controller.Compute(cpuPercentage, m.MeasureTime)
	if err != nil {
		return err
	}

	return u.orchestrator.SetDrainRate(drainRate)
}
