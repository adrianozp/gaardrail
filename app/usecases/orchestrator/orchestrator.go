package orchestrator

import (
	"context"

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
}

func NewOrchestrator(c Consumer) *Orchestrator {
	return &Orchestrator{
		consumer: c,
		// rate 0 pauses the orchestrator at startup; the PID controller sets the real
		// drain rate via SetDrainRate once metrics arrive.
		limiter: rate.NewLimiter(0, 1),
		done:    make(chan struct{}),
	}
}

// Limiter exposes the rate.Limiter for testing purposes.
func (o *Orchestrator) Limiter() *rate.Limiter {
	return o.limiter
}

// Done returns a channel that is closed when the run loop exits.
func (o *Orchestrator) Done() <-chan struct{} {
	return o.done
}

func (o *Orchestrator) SetDrainRate(drainRate float64) error {
	o.limiter.SetLimit(rate.Limit(drainRate))
	return nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	go o.run(ctx)
	return nil
}

func (o *Orchestrator) run(ctx context.Context) {
	defer close(o.done)

	for {
		if err := o.limiter.Wait(ctx); err != nil {
			log.Info().Msg("orchestrator: shutting down")
			return
		}

		queueLen, err := o.consumer.Size()
		if err != nil {
			log.Error().Err(err).Msg("orchestrator: error getting queue length")
			continue
		}

		if queueLen == 0 {
			continue
		}

		messageID, err := o.consumer.Consume()
		if err != nil {
			log.Error().Err(err).Msg("orchestrator: error consuming message")
			continue
		}

		log.Info().Str("message_id", messageID).Msg("orchestrator: consumed message")
	}
}
