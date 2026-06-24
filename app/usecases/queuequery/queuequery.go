package queuequery

import "github.com/rs/zerolog/log"

//go:generate mockery --all --output=mocks --outpkg=mocks
type QueryHolder interface {
	SetQuery(string)
	Query() string
}

// ConfigStore persists runtime changes back to the config file so they survive
// a restart.
type ConfigStore interface {
	Set(updates map[string]any) error
}

type UseCase struct {
	holder QueryHolder
	store  ConfigStore
}

func New(h QueryHolder, s ConfigStore) UseCase {
	return UseCase{holder: h, store: s}
}

// Set applies the constant query at runtime and persists it so it is restored
// on restart. A failed persist is logged but does not undo the runtime change.
func (u UseCase) Set(query string) {
	u.holder.SetQuery(query)

	if err := u.store.Set(map[string]any{"queue.query": query}); err != nil {
		log.Warn().Err(err).Msg("queuequery: persist failed")
	}
}

func (u UseCase) Get() string {
	return u.holder.Query()
}
