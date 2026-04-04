package orchestrator

import (
	"context"
	"time"

	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/adrianozp/gaardrail/pkg/metrics"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type Consumer interface {
	Consume() (string, error)
	Size() (int64, error)
}

type Orchestrator struct {
	limiter  *rate.Limiter
	consumer Consumer
	done     chan struct{}
	ctx      context.Context
	workers  int
}

func NewOrchestrator(c Consumer, cfg config.Config) *Orchestrator {
	return &Orchestrator{
		consumer: c,
		limiter:  rate.NewLimiter(rate.Limit(cfg.Orchestrator.Rate), cfg.Orchestrator.Burst),
		done:     make(chan struct{}),
		workers:  cfg.Orchestrator.Workers,
	}
}

func (o *Orchestrator) Limiter() *rate.Limiter {
	return o.limiter
}

func (o *Orchestrator) Done() <-chan struct{} {
	return o.done
}

func (o *Orchestrator) SetDrainRate(drainRate float64) error {
	log.Debug().Float64("drain_rate", drainRate).Msg("orchestrator: updated drain rate")
	o.limiter.SetLimit(rate.Limit(drainRate))
	metrics.Gauge(map[string]float64{"drain_rate": drainRate})
	return nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	o.ctx = ctx
	for i := 0; i < o.workers; i++ {
		go o.worker()
	}
	go o.lagPoller()
	return nil
}

func (o *Orchestrator) worker() {
	for {
		select {
		case <-o.ctx.Done():
			log.Panic().Msg("orchestrator: context canceled, shutting down")
			return
		default:
		}

		if o.Limiter().Limit() == 0 {
			log.Info().Msg("orchestrator: limiter not set waiting")
			time.Sleep(1 * time.Second)
			continue
		}

		if err := o.limiter.Wait(context.Background()); err != nil {
			log.Warn().Err(err).Msg("orchestrator: rate limiter error, retrying")
			continue
		}

		if _, err := o.consumer.Consume(); err != nil {
			log.Error().Err(err).Msg("orchestrator: error consuming message")
		}
	}
}

func (o *Orchestrator) lagPoller() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-t.C:
			if lag, err := o.consumer.Size(); err == nil {
				metrics.Gauge(map[string]float64{"queue_lag": float64(lag)})
			}
		}
	}
}
