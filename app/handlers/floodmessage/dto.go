package handlers

import (
	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/clock"
)

type floodRequest struct {
	Payload string `json:"payload" binding:"required"`
}

func (r floodRequest) toMessage() entities.Message {
	return entities.Message{
		Body:      []byte(r.Payload),
		CreatedAt: clock.Now(),
	}
}
