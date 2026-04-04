package handlers

import (
	"net/http"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/gin-gonic/gin"
)

//go:generate mockery --name=UpdatePIDParamsUseCase --output=mocks --outpkg=mocks
type UpdatePIDParamsUseCase interface {
	Update(p entities.PIDParams) error
}

type UpdatePIDParamsHandler struct {
	usecase UpdatePIDParamsUseCase
}

func NewUpdatePIDParamsHandler(usecase UpdatePIDParamsUseCase) *UpdatePIDParamsHandler {
	return &UpdatePIDParamsHandler{usecase: usecase}
}

func RegisterUpdatePIDParamsRoutes(router *gin.Engine, h *UpdatePIDParamsHandler) {
	router.PATCH("/pid", h.Handle)
}

func (h *UpdatePIDParamsHandler) Handle(c *gin.Context) {
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
