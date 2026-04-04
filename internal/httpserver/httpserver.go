package httpserver

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func New() *gin.Engine {
	router := gin.New()
	router.Use(metricsMiddleware())

	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return router
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Dur("duration_ms", time.Since(start)).
			Msg("request")
	}
}
