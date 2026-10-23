package orchestrator

import (
	"context"
	"sync"
	"time"

	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type Consumer interface {
	Consume(context.Context) (string, error)
	Size() (int64, error)
}

type Orchestrator struct {
	limiter  *rate.Limiter
	consumer Consumer
	workers  int
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

const idleWait = time.Second

func NewOrchestrator(c Consumer, cfg config.Config) *Orchestrator {
	return &Orchestrator{
		consumer: c,
		limiter:  rate.NewLimiter(rate.Limit(cfg.Orchestrator.Rate), cfg.Orchestrator.Burst),
		workers:  cfg.Orchestrator.Workers,
		cancel:   func() {},
	}
}

func (o *Orchestrator) Limiter() *rate.Limiter {
	return o.limiter
}

func (o *Orchestrator) SetDrainRate(drainRate float64) error {
	log.Debug().Float64("drain_rate", drainRate).Msg("orchestrator: updated drain rate")
	o.limiter.SetLimit(rate.Limit(drainRate))
	metrics.Gauge(map[string]float64{"drain_rate": drainRate})
	return nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	ctx, o.cancel = context.WithCancel(ctx)
	for range o.workers {
		o.wg.Go(func() { o.worker(ctx) })
	}
	o.wg.Go(func() { o.lagPoller(ctx) })
	return nil
}

func (o *Orchestrator) Stop(ctx context.Context) error {
	o.cancel()
	done := make(chan struct{})
	go func() {
		o.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (o *Orchestrator) worker(ctx context.Context) {
	for ctx.Err() == nil {
		if o.awaitToken(ctx) {
			o.consume(ctx)
		}
	}
	log.Info().Msg("orchestrator: worker stopped")
}

func (o *Orchestrator) awaitToken(ctx context.Context) bool {
	if o.limiter.Limit() == 0 {
		log.Debug().Msg("orchestrator: drain rate is 0, waiting")
		sleep(ctx, idleWait)
		return false
	}
	r := o.limiter.Reserve()
	if !r.OK() {
		sleep(ctx, idleWait)
		return false
	}
	if r.Delay() > idleWait || !sleep(ctx, r.Delay()) {
		r.Cancel()
		sleep(ctx, idleWait)
		return false
	}
	return true
}

func (o *Orchestrator) consume(ctx context.Context) {
	_, err := o.consumer.Consume(ctx)
	if err != nil && ctx.Err() == nil {
		log.Error().Err(err).Msg("orchestrator: error consuming message")
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func (o *Orchestrator) lagPoller(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			o.publishLag()
		}
	}
}

func (o *Orchestrator) publishLag() {
	if lag, err := o.consumer.Size(); err == nil {
		metrics.Gauge(map[string]float64{"queue_lag": float64(lag)})
	}
}
