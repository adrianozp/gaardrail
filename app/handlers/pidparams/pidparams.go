package handlers

import (
	"net/http"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/gin-gonic/gin"
)

//go:generate mockery --name=PIDParamsUseCase --output=mocks --outpkg=mocks
type PIDParamsUseCase interface {
	Update(p entities.PIDParams) error
	Get() entities.PIDParams
}

type PIDParamsHandler struct {
	usecase PIDParamsUseCase
}

func New(usecase PIDParamsUseCase) *PIDParamsHandler {
	return &PIDParamsHandler{usecase: usecase}
}

func RegisterRoutes(router *gin.Engine, h *PIDParamsHandler) {
	router.GET("/pid", h.HandleGet)
	router.PATCH("/pid", h.HandleUpdate)
}

func (h *PIDParamsHandler) HandleGet(c *gin.Context) {
	c.JSON(http.StatusOK, pidParamsFromEntity(h.usecase.Get()))
}

func (h *PIDParamsHandler) HandleUpdate(c *gin.Context) {
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
