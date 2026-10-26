package handlers

import (
	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/ptr"
)

type updatePIDParamsRequest struct {
	Kp       *float64 `json:"kp"`
	Ki       *float64 `json:"ki"`
	Kd       *float64 `json:"kd"`
	Min      *float64 `json:"min"`
	Max      *float64 `json:"max"`
	IClamp   *float64 `json:"i_clamp"`
	Setpoint *float64 `json:"setpoint"`
	FfGain   *float64 `json:"ff_gain"`

	SetpointFilterType *string `json:"setpoint_filter_type"`
	SetpointFilterSize *int    `json:"setpoint_filter_size"`

	ModelK        *float64 `json:"model_k"`
	ModelTau      *float64 `json:"model_tau"`
	ModelTheta    *float64 `json:"model_theta"`
	SampleSeconds *float64 `json:"sample_seconds"`
}

func (r updatePIDParamsRequest) toPIDParams() entities.ControllerParams {
	return entities.ControllerParams{
		Kp:                 r.Kp,
		Ki:                 r.Ki,
		Kd:                 r.Kd,
		Min:                r.Min,
		Max:                r.Max,
		IClamp:             r.IClamp,
		Setpoint:           r.Setpoint,
		FfGain:             r.FfGain,
		SetpointFilterType: r.SetpointFilterType,
		SetpointFilterSize: r.SetpointFilterSize,
		ModelK:             r.ModelK,
		ModelTau:           r.ModelTau,
		ModelTheta:         r.ModelTheta,
		SampleSeconds:      r.SampleSeconds,
	}
}

type pidParamsResponse struct {
	Type     string  `json:"type"`
	Kp       float64 `json:"kp"`
	Ki       float64 `json:"ki"`
	Kd       float64 `json:"kd"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	IClamp   float64 `json:"i_clamp"`
	Setpoint float64 `json:"setpoint"`
	FfGain   float64 `json:"ff_gain"`

	SetpointFilterType string `json:"setpoint_filter_type"`
	SetpointFilterSize int    `json:"setpoint_filter_size"`

	ModelK        *float64 `json:"model_k,omitempty"`
	ModelTau      *float64 `json:"model_tau,omitempty"`
	ModelTheta    *float64 `json:"model_theta,omitempty"`
	SampleSeconds *float64 `json:"sample_seconds,omitempty"`
}

func pidParamsFromEntity(p entities.ControllerParams) pidParamsResponse {
	r := pidParamsResponse{
		ModelK:        p.ModelK,
		ModelTau:      p.ModelTau,
		ModelTheta:    p.ModelTheta,
		SampleSeconds: p.SampleSeconds,
	}
	ptr.Assign(&r.Kp, p.Kp)
	ptr.Assign(&r.Ki, p.Ki)
	ptr.Assign(&r.Kd, p.Kd)
	ptr.Assign(&r.Min, p.Min)
	ptr.Assign(&r.Max, p.Max)
	ptr.Assign(&r.IClamp, p.IClamp)
	ptr.Assign(&r.Setpoint, p.Setpoint)
	ptr.Assign(&r.FfGain, p.FfGain)
	ptr.Assign(&r.SetpointFilterType, p.SetpointFilterType)
	ptr.Assign(&r.SetpointFilterSize, p.SetpointFilterSize)
	return r
}
