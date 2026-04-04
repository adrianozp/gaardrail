package repositories

import "context"

//go:generate mockery --all --output=mocks --outpkg=mocks

// MetricsReader is the port for pulling metrics from an external source.
// Each adapter maps source field names to domain names before returning.
type MetricsReader interface {
	Read(ctx context.Context) (map[string]float64, error)
}
