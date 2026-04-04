package handlers

import "github.com/adrianozp/gaardrail/app/entities"

type createMessageRequest struct {
	Payload string `json:"payload" binding:"required"`
}

func (c createMessageRequest) toMessage() entities.Message {
	return entities.Message{
		Body: []byte(c.Payload),
	}
}
