package pollmetrics

import (
	"context"
	"time"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/rs/zerolog/log"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type MetricsReader interface {
	Read(ctx context.Context) (entities.Metrics, error)
}

// ProcessMetrics is the port for forwarding collected metrics into the PID pipeline.
type ProcessMetrics interface {
	Process(m entities.Metrics) error
}

// PollingHandler ticks on a configurable interval, reads metrics from a
// MetricsReader, and forwards them to ProcessMetrics. Errors from Read are
// logged and skipped — the PID controller retains its last state until the
// next successful read.
type PollingHandler struct {
	reader         MetricsReader
	processMetrics ProcessMetrics
	interval       time.Duration
	done           chan struct{}
	cancel         context.CancelFunc
	started        bool
}

func New(reader MetricsReader, pm ProcessMetrics, cfg config.Config) *PollingHandler {
	interval := time.Duration(cfg.MetricsPoller.IntervalMs) * time.Millisecond
	return &PollingHandler{
		reader:         reader,
		processMetrics: pm,
		interval:       interval,
		done:           make(chan struct{}),
		cancel:         func() {},
	}
}

// Done returns a channel that is closed when the run loop exits.
// Useful in tests to assert clean shutdown.
func (h *PollingHandler) Done() <-chan struct{} {
	return h.done
}

// Start launches the polling goroutine. Must be called at most once.
func (h *PollingHandler) Start(ctx context.Context) error {
	ctx, h.cancel = context.WithCancel(ctx)
	h.started = true
	go h.run(ctx)
	return nil
}

func (h *PollingHandler) Stop(ctx context.Context) error {
	h.cancel()
	if !h.started {
		return nil
	}
	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *PollingHandler) run(ctx context.Context) {
	defer close(h.done)
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("pollmetrics: shutting down")
			return
		case <-ticker.C:
			h.poll(ctx)
		}
	}
}

func (h *PollingHandler) poll(ctx context.Context) {
	metrics, err := h.reader.Read(ctx)
	if err != nil {
		log.Error().Err(err).Msg("pollmetrics: read error")
		return
	}
	if err := h.processMetrics.Process(metrics); err != nil {
		log.Error().Err(err).Msg("pollmetrics: process error")
	}
}
