package orchestrator

import (
	"context"

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
}

func NewOrchestrator(c Consumer, cfg config.Config) *Orchestrator {
	return &Orchestrator{
		consumer: c,
		// rate 0 pauses the orchestrator at startup; the PID controller sets the real
		// drain rate via SetDrainRate once metrics arrive.
		limiter: rate.NewLimiter(rate.Limit(cfg.Orchestrator.Rate), cfg.Orchestrator.Burst),
		done:    make(chan struct{}),
	}
}

func (o *Orchestrator) Limiter() *rate.Limiter {
	return o.limiter
}

func (o *Orchestrator) Done() <-chan struct{} {
	return o.done
}

func (o *Orchestrator) SetDrainRate(drainRate float64) error {
	log.Info().Float64("drain_rate", drainRate).Msg("orchestrator: updated drain rate")
	o.limiter.SetLimit(rate.Limit(drainRate))
	metrics.Gauge(map[string]float64{"drain_rate": drainRate})
	return nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	o.ctx = ctx
	go o.run()
	return nil
}

func (o *Orchestrator) run() {
	defer close(o.done)

	for {
		select {
		case <-o.ctx.Done():
			log.Panic().Msg("orchestrator: context canceled, shutting down")
			return
		default:
		}

		ctx := context.Background()
		if err := o.limiter.Wait(ctx); err != nil {
			log.Warn().Err(err).Msg("orchestrator: rate limiter error, retrying")
			continue
		}

		go func() {
			_, err := o.consumer.Consume()
			if err != nil {
				log.Error().Err(err).Msg("orchestrator: error consuming message")
			}
		}()

		if lag, err := o.consumer.Size(); err == nil {
			metrics.Gauge(map[string]float64{"queue_lag": float64(lag)})
		}
	}
}
