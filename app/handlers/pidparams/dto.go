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
}

func (r updatePIDParamsRequest) toPIDParams() entities.PIDParams {
	return entities.PIDParams{
		Kp:       r.Kp,
		Ki:       r.Ki,
		Kd:       r.Kd,
		Min:      r.Min,
		Max:      r.Max,
		IClamp:   r.IClamp,
		Setpoint: r.Setpoint,
	}
}

type pidParamsResponse struct {
	Kp         float64 `json:"kp"`
	Ki         float64 `json:"ki"`
	Kd         float64 `json:"kd"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	IClamp     float64 `json:"i_clamp"`
	Setpoint   float64 `json:"setpoint"`
	FilterSize int     `json:"filter_size"`
}

func pidParamsFromEntity(p entities.PIDParams) pidParamsResponse {
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
	return r
}
