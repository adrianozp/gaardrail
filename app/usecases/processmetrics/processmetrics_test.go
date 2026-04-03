package processmetrics_test

import (
	"errors"
	"testing"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics"
	"github.com/adrianozp/gaardrail/app/usecases/processmetrics/mocks"
	"github.com/stretchr/testify/assert"
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
	mockOrchestrator.On("UpdateDrainRate", 1.5).Return(nil)

	uc := processmetrics.NewProcessMetricsUseCase(mockController, mockOrchestrator)
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

	uc := processmetrics.NewProcessMetricsUseCase(mockController, mockOrchestrator)
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

	uc := processmetrics.NewProcessMetricsUseCase(mockController, mockOrchestrator)
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
	mockOrchestrator.On("UpdateDrainRate", 2.0).Return(errors.New("orchestrator error"))

	uc := processmetrics.NewProcessMetricsUseCase(mockController, mockOrchestrator)
	err := uc.Process(metrics)

	assert.EqualError(t, err, "orchestrator error")
}
