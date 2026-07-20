package handlers

import "github.com/adrianozp/gaardrail/app/entities"

type updatePIDParamsRequest struct {
	Kp       *float64 `json:"kp"`
	Ki       *float64 `json:"ki"`
	Kd       *float64 `json:"kd"`
	Min      *float64 `json:"min"`
	Max      *float64 `json:"max"`
	IClamp   *float64 `json:"i_clamp"`
	Setpoint *float64 `json:"setpoint"`
	FfGain   *float64 `json:"ff_gain"`

	// Smith predictor internal FOPDT model (ignored by other controllers).
	ModelK        *float64 `json:"model_k"`
	ModelTau      *float64 `json:"model_tau"`
	ModelTheta    *float64 `json:"model_theta"`
	SampleSeconds *float64 `json:"sample_seconds"`
}

func (r updatePIDParamsRequest) toPIDParams() entities.ControllerParams {
	return entities.ControllerParams{
		Kp:            r.Kp,
		Ki:            r.Ki,
		Kd:            r.Kd,
		Min:           r.Min,
		Max:           r.Max,
		IClamp:        r.IClamp,
		Setpoint:      r.Setpoint,
		FfGain:        r.FfGain,
		ModelK:        r.ModelK,
		ModelTau:      r.ModelTau,
		ModelTheta:    r.ModelTheta,
		SampleSeconds: r.SampleSeconds,
	}
}

type pidParamsResponse struct {
	Type       string  `json:"type"`
	Kp         float64 `json:"kp"`
	Ki         float64 `json:"ki"`
	Kd         float64 `json:"kd"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	IClamp     float64 `json:"i_clamp"`
	Setpoint   float64 `json:"setpoint"`
	FilterSize int     `json:"filter_size"`
	FfGain     float64 `json:"ff_gain"`

	// Smith predictor internal FOPDT model (omitted for other controllers).
	ModelK        *float64 `json:"model_k,omitempty"`
	ModelTau      *float64 `json:"model_tau,omitempty"`
	ModelTheta    *float64 `json:"model_theta,omitempty"`
	SampleSeconds *float64 `json:"sample_seconds,omitempty"`
}

func pidParamsFromEntity(p entities.ControllerParams) pidParamsResponse {
	r := pidParamsResponse{}
	if p.Kp != nil {
		r.Kp = *p.Kp
	}
	if p.Ki != nil {
		r.Ki = *p.Ki
	}
	if p.Kd != nil {
		r.Kd = *p.Kd
	}
	if p.Min != nil {
		r.Min = *p.Min
	}
	if p.Max != nil {
		r.Max = *p.Max
	}
	if p.IClamp != nil {
		r.IClamp = *p.IClamp
	}
	if p.Setpoint != nil {
		r.Setpoint = *p.Setpoint
	}
	if p.FilterSize != nil {
		r.FilterSize = *p.FilterSize
	}
	if p.FfGain != nil {
		r.FfGain = *p.FfGain
	}
	r.ModelK = p.ModelK
	r.ModelTau = p.ModelTau
	r.ModelTheta = p.ModelTheta
	r.SampleSeconds = p.SampleSeconds
	return r
}
