package handlers

import (
	"net/http"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/gin-gonic/gin"
)

//go:generate mockery --name=CreateMessageUseCase --output=mocks --outpkg=mocks
type CreateMessageUseCase interface {
	Create(entities.Message) (string, error)
}

type CreateMessageHandler struct {
	usecase CreateMessageUseCase
}

func NewCreateMessageHandler(usecase CreateMessageUseCase) *CreateMessageHandler {
	return &CreateMessageHandler{usecase: usecase}
}

func RegisterCreateMessageRoutes(router *gin.Engine, h *CreateMessageHandler) {
	router.POST("/messages", h.Handle)
}

func (h *CreateMessageHandler) Handle(c *gin.Context) {
	var req createMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg := req.toMessage()
	id, err := h.usecase.Create(msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func RegisterCreateMessageRoute(router *gin.Engine, handler CreateMessageHandler) {
	router.POST("/message", handler.Handle)
}
