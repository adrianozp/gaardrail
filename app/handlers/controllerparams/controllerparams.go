package handlers

import (
	"net/http"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/gin-gonic/gin"
)

//go:generate mockery --name=ControllerParamsUseCase --output=mocks --outpkg=mocks
type ControllerParamsUseCase interface {
	Update(p entities.ControllerParams) error
	Get() entities.ControllerParams
}

type ControllerParamsHandler struct {
	usecase        ControllerParamsUseCase
	controllerType string
}

func New(usecase ControllerParamsUseCase, cfg config.Config) *ControllerParamsHandler {
	return &ControllerParamsHandler{usecase: usecase, controllerType: cfg.Controller.Type}
}

func RegisterRoutes(router *gin.Engine, h *ControllerParamsHandler) {
	router.GET("/pid", h.HandleGet)
	router.PATCH("/pid", h.HandleUpdate)
}

func (h *ControllerParamsHandler) HandleGet(c *gin.Context) {
	resp := pidParamsFromEntity(h.usecase.Get())
	resp.Type = h.controllerType
	c.JSON(http.StatusOK, resp)
}

func (h *ControllerParamsHandler) HandleUpdate(c *gin.Context) {
	var req updatePIDParamsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.usecase.Update(req.toPIDParams()); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
