package handlers

import (
	"net/http"
	"strconv"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const maxQuantity = 10000

//go:generate mockery --name=CreateMessageUseCase --output=mocks --outpkg=mocks
type CreateMessageUseCase interface {
	Create(entities.Message) (string, error)
}

type FloodMessageHandler struct {
	usecase CreateMessageUseCase
}

func NewFloodMessageHandler(usecase CreateMessageUseCase) *FloodMessageHandler {
	return &FloodMessageHandler{usecase: usecase}
}

func RegisterFloodMessageRoutes(router *gin.Engine, h *FloodMessageHandler) {
	router.POST("/messages/flood", h.Handle)
}

func (h *FloodMessageHandler) Handle(c *gin.Context) {
	quantityStr := c.DefaultQuery("quantity", "1")
	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be at least 1"})
		return
	}
	if quantity > maxQuantity {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity exceeds max (10000)"})
		return
	}

	var req floodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"queued": quantity})

	go func() {
		for i := 0; i < quantity; i++ {
			if _, err := h.usecase.Create(req.toMessage()); err != nil {
				log.Error().Err(err).Msg("flood: failed to enqueue message")
			}
		}
	}()
}
