package httpserver

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
		log.Printf("method=%s path=%s duration=%s", c.Request.Method, c.Request.URL.Path, time.Since(start))
	}
}
