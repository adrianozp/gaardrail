package receivemetrics

import "github.com/adrianozp/gaardrail/app/entities"

type receiveMetricsRequest struct {
	MeasureTime string             `json:"measureTime"`
	Metrics     map[string]float64 `json:"metrics"`
}

func (r receiveMetricsRequest) toMetrics() entities.Metrics {
	return entities.Metrics{
		MeasureTime: r.toMetrics().MeasureTime,
		Metrics:     r.Metrics,
	}
}
