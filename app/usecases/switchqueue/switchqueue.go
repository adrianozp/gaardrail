package switchqueue

import (
	metrics "github.com/adrianozp/gaardrail/internal/metrics"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/rs/zerolog/log"
)

//go:generate mockery --all --output=mocks --outpkg=mocks

// Queue is the runtime-switchable queue. Satisfied by *switchable.Queue.
type Queue interface {
	SetType(t string) error
	Type() string
	Available() []string
}

// ConfigStore persists runtime changes back to the config file so they survive
// a restart.
type ConfigStore interface {
	Set(updates map[string]any) error
}

type UseCase struct {
	queue Queue
	store ConfigStore
}

func New(q Queue, s ConfigStore) UseCase {
	return UseCase{queue: q, store: s}
}

// Switch changes the active queue at runtime, updates the dashboard gauge so the
// switch is visible, and persists the choice so it is restored on restart. A
// failed persist is logged but does not undo the runtime switch.
func (u UseCase) Switch(t string) error {
	if err := u.queue.SetType(t); err != nil {
		return err
	}

	metrics.Gauge(map[string]float64{"queue_type": config.TypeCode(t)})

	if err := u.store.Set(map[string]any{"queue.protocol": t}); err != nil {
		log.Warn().Err(err).Msg("switchqueue: persist failed")
	}
	return nil
}

func (u UseCase) Current() string {
	return u.queue.Type()
}

func (u UseCase) Available() []string {
	return u.queue.Available()
}
