package handlers

import (
	"net/http"
	"time"

	"github.com/adrianozp/gaardrail/app/disturbance"
	"github.com/gin-gonic/gin"
)

//go:generate mockery --name=DisturbanceUseCase --output=mocks --outpkg=mocks
type DisturbanceUseCase interface {
	Set(query string, ratePerSecond float64, ttl time.Duration) error
	Get() disturbance.State
}

type DisturbanceHandler struct {
	usecase DisturbanceUseCase
}

func New(usecase DisturbanceUseCase) *DisturbanceHandler {
	return &DisturbanceHandler{usecase: usecase}
}

func RegisterRoutes(router *gin.Engine, h *DisturbanceHandler) {
	router.GET("/disturbance", h.HandleGet)
	router.POST("/disturbance", h.HandleSet)
}

func (h *DisturbanceHandler) HandleGet(c *gin.Context) {
	c.JSON(http.StatusOK, disturbanceResponseFromState(h.usecase.Get()))
}

func (h *DisturbanceHandler) HandleSet(c *gin.Context) {
	var req setDisturbanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ttl := time.Duration(req.DurationSeconds * float64(time.Second))
	if err := h.usecase.Set(req.Query, req.Rate, ttl); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
