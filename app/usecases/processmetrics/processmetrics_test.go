package processmetrics_test

import (
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics/mocks"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProcess_Success(t *testing.T) {
	now := time.Now()
	metrics := entities.Metrics{
		MeasureTime: now,
		Metrics:     map[string]float64{"cpu": 0.75},
	}
	mockController := mocks.NewController(t)
	mockOrchestrator := mocks.NewOrchestrator(t)
	mockController.On("Compute", 0.75, now).Return(1.5, nil)
	mockOrchestrator.On("SetDrainRate", 1.5).Return(nil)

	uc := processmetrics.NewProcessMetricsUseCase(mockController, mockOrchestrator, config.Config{})
	err := uc.Process(metrics)

	assert.NoError(t, err)
}

func TestProcess_MissingCPUMetric(t *testing.T) {
	metrics := entities.Metrics{
		MeasureTime: time.Now(),
		Metrics:     map[string]float64{"memory": 0.5},
	}
	mockController := mocks.NewController(t)
	mockOrchestrator := mocks.NewOrchestrator(t)

	uc := processmetrics.NewProcessMetricsUseCase(mockController, mockOrchestrator, config.Config{})
	err := uc.Process(metrics)

	assert.EqualError(t, err, "cpu metric not found")
}

func TestProcess_ControllerError(t *testing.T) {
	now := time.Now()
	metrics := entities.Metrics{
		MeasureTime: now,
		Metrics:     map[string]float64{"cpu": 0.9},
	}
	mockController := mocks.NewController(t)
	mockOrchestrator := mocks.NewOrchestrator(t)
	mockController.On("Compute", 0.9, now).Return(0.0, errors.New("controller error"))

	uc := processmetrics.NewProcessMetricsUseCase(mockController, mockOrchestrator, config.Config{})
	err := uc.Process(metrics)

	assert.EqualError(t, err, "controller error")
}

func TestProcess_OrchestratorError(t *testing.T) {
	now := time.Now()
	metrics := entities.Metrics{
		MeasureTime: now,
		Metrics:     map[string]float64{"cpu": 0.6},
	}
	mockController := mocks.NewController(t)
	mockOrchestrator := mocks.NewOrchestrator(t)
	mockController.On("Compute", 0.6, now).Return(2.0, nil)
	mockOrchestrator.On("SetDrainRate", 2.0).Return(errors.New("orchestrator error"))

	uc := processmetrics.NewProcessMetricsUseCase(mockController, mockOrchestrator, config.Config{})
	err := uc.Process(metrics)

	assert.EqualError(t, err, "orchestrator error")
}

func TestProcessAplicaFiltroGeralNaPV(t *testing.T) {
	cfg := config.Config{MetricsPoller: config.MetricsPoller{FilterType: "moving_average", FilterSize: 2}}
	mockController := mocks.NewController(t)
	mockOrchestrator := mocks.NewOrchestrator(t)
	uc := processmetrics.NewProcessMetricsUseCase(mockController, mockOrchestrator, cfg)

	mockController.On("Compute", 40.0, mock.Anything).Return(10.0, nil)
	mockOrchestrator.On("SetDrainRate", 10.0).Return(nil)
	err := uc.Process(entities.Metrics{Metrics: map[string]float64{"cpu": 40}, MeasureTime: time.Now()})
	assert.NoError(t, err)

	mockController.On("Compute", 60.0, mock.Anything).Return(10.0, nil)
	err = uc.Process(entities.Metrics{Metrics: map[string]float64{"cpu": 80}, MeasureTime: time.Now()})
	assert.NoError(t, err)
}
