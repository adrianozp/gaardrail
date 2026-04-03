package http

import "github.com/adrianozp/gaardrail/app/entities"

type model struct {
	ID string `json:"id"`
}

func FromMesssage(m entities.Message) model {
	return model{
		ID: m.ID,
	}
}
