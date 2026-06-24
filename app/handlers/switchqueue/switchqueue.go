package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:generate mockery --name=SwitchQueueUseCase --output=mocks --outpkg=mocks
type SwitchQueueUseCase interface {
	Switch(t string) error
	Current() string
	Available() []string
}

type SwitchQueueHandler struct {
	usecase SwitchQueueUseCase
}

func New(usecase SwitchQueueUseCase) *SwitchQueueHandler {
	return &SwitchQueueHandler{usecase: usecase}
}

func RegisterRoutes(router *gin.Engine, h *SwitchQueueHandler) {
	router.GET("/queue/type", h.HandleGet)
	router.PUT("/queue/type", h.HandleSet)
}

func (h *SwitchQueueHandler) HandleGet(c *gin.Context) {
	c.JSON(http.StatusOK, switchQueueResponse{
		Type:      h.usecase.Current(),
		Available: h.usecase.Available(),
	})
}

func (h *SwitchQueueHandler) HandleSet(c *gin.Context) {
	var req switchQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Switch(req.Type); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
