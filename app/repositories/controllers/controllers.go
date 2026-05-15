package controllers

import (
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
)

type Controller interface {
	Compute(measured float64, measureTime time.Time) (float64, error)
	GetParams() entities.ControllerParams
	SetParams(p entities.ControllerParams) error
	SetSetpoint(setpoint float64) error
	Reset()
}
