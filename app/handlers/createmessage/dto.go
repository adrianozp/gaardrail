package handlers

import "github.com/adrianozp/gaardrail/app/entities"

type createMessageRequest struct {
	ID string `json:"id" binding:"required"`
}

func (c createMessageRequest) toMessage() entities.Message {
	return entities.Message{
		ID: c.ID,
	}
}
