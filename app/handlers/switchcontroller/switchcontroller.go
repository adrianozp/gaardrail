package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:generate mockery --name=SwitchControllerUseCase --output=mocks --outpkg=mocks
type SwitchControllerUseCase interface {
	Switch(t string) error
	Current() string
}

type SwitchControllerHandler struct {
	usecase SwitchControllerUseCase
}

func New(usecase SwitchControllerUseCase) *SwitchControllerHandler {
	return &SwitchControllerHandler{usecase: usecase}
}

func RegisterRoutes(router *gin.Engine, h *SwitchControllerHandler) {
	router.PUT("/controller/type", h.HandleSet)
}

func (h *SwitchControllerHandler) HandleSet(c *gin.Context) {
	var req switchControllerRequest
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
