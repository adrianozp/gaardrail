package processmetrics

import (
	"errors"
	"fmt"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/adrianozp/gaardrail/pkg/control/filter"
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
	pvFilter     *filter.Signal
}

func NewProcessMetricsUseCase(c Controller, o Orchestrator, cfg config.Config) (ProcessMetricsUseCase, error) {
	spec := filter.Spec{Kind: filter.Kind(cfg.MetricsPoller.FilterType), Size: cfg.MetricsPoller.FilterSize}
	pvFilter, err := filter.NewMeasurement(spec)
	if err != nil {
		return ProcessMetricsUseCase{}, fmt.Errorf("processmetrics: %w", err)
	}
	return ProcessMetricsUseCase{
		controller:   c,
		orchestrator: o,
		pvFilter:     pvFilter,
	}, nil
}

func (u ProcessMetricsUseCase) Process(m entities.Metrics) error {
	cpuPercentage, ok := m.Metrics["cpu"]
	if !ok {
		log.Error().Msg("cpu metric not found")
		return errors.New("cpu metric not found")
	}

	cpuPercentage = u.pvFilter.Filter(cpuPercentage)

	metrics.Gauge(map[string]float64{"measured_cpu": cpuPercentage})

	drainRate, err := u.controller.Compute(cpuPercentage, m.MeasureTime)
	if err != nil {
		log.Error().Msg("computing drain rate")
		return err
	}

	return u.orchestrator.SetDrainRate(drainRate)
}
