package processmetrics

import (
	"errors"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/rs/zerolog/log"
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
		log.Error().Msg("cpu metric not found")
		return errors.New("cpu metric not found")
	}

	// measured_cpu is the controller-agnostic process variable fed to the
	// controller. Emitted here (not in the controller) so the dashboard shows
	// it regardless of which controller is active.
	metrics.Gauge(map[string]float64{"measured_cpu": cpuPercentage})

	drainRate, err := u.controller.Compute(cpuPercentage, m.MeasureTime)
	if err != nil {
		log.Error().Msg("computing drain rate")
		return err
	}

	return u.orchestrator.SetDrainRate(drainRate)
}
