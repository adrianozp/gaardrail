package receivemetrics

import (
	"net/http"

	"github.com/adrianozp/gaardrail/app/entities"
	"github.com/gin-gonic/gin"
)

//go:generate mockery --all --output=mocks --outpkg=mocks
type ProcessMetricsUseCase interface {
	Process(m entities.Metrics) error
}

type ReceiveMetricsHandler struct {
	useCase ProcessMetricsUseCase
}

func NewReceiveMetricsHandler(u ProcessMetricsUseCase) ReceiveMetricsHandler {
	return ReceiveMetricsHandler{
		useCase: u,
	}
}

func (h ReceiveMetricsHandler) Handle(c *gin.Context) {
	var req receiveMetricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metrics := req.toMetrics()
	err := h.useCase.Process(metrics)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusAccepted)
}
