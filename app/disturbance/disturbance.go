package disturbance

import (
	"context"
	"sync"
	"time"

	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type Executor interface {
	ExecContext(ctx context.Context, query string) (int64, error)
}

type State struct {
	Query    string
	Rate     float64
	Duration time.Duration
}

// Disturbance injects load against the SQL target by executing a query at a
// configured rate (queries/second). It publishes the configured rate as the
// disturbance_rate gauge, analogous to the orchestrator drain_rate.
type Disturbance struct {
	executor Executor

	mu     sync.Mutex
	state  State
	cancel context.CancelFunc
	timer  *time.Timer
	ctx    context.Context
}

func New(executor Executor) *Disturbance {
	return &Disturbance{executor: executor, ctx: context.Background()}
}

func (d *Disturbance) Start(ctx context.Context) error {
	d.mu.Lock()
	d.ctx = ctx
	d.mu.Unlock()
	return nil
}

func (d *Disturbance) Get() State {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.state
}

// Set (re)configures the disturbance. rate <= 0 stops it. A positive ttl turns
// it into a timed pulse that auto-stops; ttl == 0 keeps it running until stopped.
func (d *Disturbance) Set(query string, ratePerSecond float64, ttl time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.stopLocked()
	d.state = State{Query: query, Rate: ratePerSecond, Duration: ttl}

	if ratePerSecond <= 0 {
		metrics.Gauge(map[string]float64{"disturbance_rate": 0})
		return nil
	}
	if d.executor == nil {
		log.Warn().Msg("disturbance: no sql executor (target is not sql); ignoring")
		d.state.Rate = 0
		metrics.Gauge(map[string]float64{"disturbance_rate": 0})
		return nil
	}

	runCtx, cancel := context.WithCancel(d.ctx)
	d.cancel = cancel
	limiter := rate.NewLimiter(rate.Limit(ratePerSecond), 1)
	go d.run(runCtx, limiter, query)
	metrics.Gauge(map[string]float64{"disturbance_rate": ratePerSecond})

	if ttl > 0 {
		d.timer = time.AfterFunc(ttl, func() {
			d.mu.Lock()
			defer d.mu.Unlock()
			d.stopLocked()
			d.state.Rate = 0
			metrics.Gauge(map[string]float64{"disturbance_rate": 0})
		})
	}
	return nil
}

func (d *Disturbance) stopLocked() {
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}

func (d *Disturbance) run(ctx context.Context, limiter *rate.Limiter, query string) {
	for {
		if err := limiter.Wait(ctx); err != nil {
			return
		}
		go func() {
			if _, err := d.executor.ExecContext(ctx, query); err != nil && ctx.Err() == nil {
				log.Error().Err(err).Msg("disturbance: query failed")
			}
		}()
	}
}
