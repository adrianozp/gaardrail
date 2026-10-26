package handlers

import (
	"errors"
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
	quantity, err := parseQuantity(c.DefaultQuery("quantity", "1"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req floodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"queued": quantity})
	go h.flood(req, quantity)
}

func parseQuantity(raw string) (int, error) {
	quantity, err := strconv.Atoi(raw)
	if err != nil || quantity < 1 {
		return 0, errors.New("quantity must be at least 1")
	}
	if quantity > maxQuantity {
		return 0, errors.New("quantity exceeds max (10000)")
	}
	return quantity, nil
}

func (h *FloodMessageHandler) flood(req floodRequest, quantity int) {
	for i := 0; i < quantity; i++ {
		h.enqueue(req)
	}
}

func (h *FloodMessageHandler) enqueue(req floodRequest) {
	if _, err := h.usecase.Create(req.toMessage()); err != nil {
		log.Error().Err(err).Msg("flood: failed to enqueue message")
	}
}
