package modules

import (
	queuerepo "github.com/adrianozp/gaardrail/app/repositories/queue"
	"go.uber.org/fx"
)

func QueueFactories() fx.Option {
	return fx.Provide(queuerepo.New)
}
