package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:generate mockery --name=QueueQueryUseCase --output=mocks --outpkg=mocks
type QueueQueryUseCase interface {
	Set(query string)
	Get() string
}

type QueueQueryHandler struct {
	usecase QueueQueryUseCase
}

func New(usecase QueueQueryUseCase) *QueueQueryHandler {
	return &QueueQueryHandler{usecase: usecase}
}

func RegisterRoutes(router *gin.Engine, h *QueueQueryHandler) {
	router.GET("/queue/query", h.HandleGet)
	router.PUT("/queue/query", h.HandleSet)
}

func (h *QueueQueryHandler) HandleGet(c *gin.Context) {
	c.JSON(http.StatusOK, queueQueryResponse{Query: h.usecase.Get()})
}

func (h *QueueQueryHandler) HandleSet(c *gin.Context) {
	var req setQueueQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.usecase.Set(req.Query)
	c.Status(http.StatusNoContent)
}
