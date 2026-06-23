package handlers

import "github.com/adrianozp/gaardrail/app/disturbance"

type setDisturbanceRequest struct {
	Query           string  `json:"query" binding:"required"`
	Rate            float64 `json:"rate"`
	DurationSeconds float64 `json:"duration_seconds"`
}

type disturbanceResponse struct {
	Query           string  `json:"query"`
	Rate            float64 `json:"rate"`
	DurationSeconds float64 `json:"duration_seconds"`
}

func disturbanceResponseFromState(s disturbance.State) disturbanceResponse {
	return disturbanceResponse{
		Query:           s.Query,
		Rate:            s.Rate,
		DurationSeconds: s.Duration.Seconds(),
	}
}
