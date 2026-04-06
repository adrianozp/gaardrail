package httpserver

import (
	"bytes"
	"io/fs"
	"net/http"
	"time"

	"github.com/adrianozp/gaardrail/pkg/config"
	"github.com/adrianozp/gaardrail/web"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func New(cfg config.Config) *gin.Engine {
	router := gin.New()
	router.Use(metricsMiddleware())

	registerWeb(router, cfg)

	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return router
}

func registerWeb(router *gin.Engine, cfg config.Config) {
	html := bytes.ReplaceAll(web.IndexHTML, []byte("{{GRAFANA_URL}}"), []byte(cfg.Grafana.URL))

	router.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", html)
	})

	favicons, _ := fs.Sub(web.Favicons, "favicon")
	router.StaticFS("/favicon", http.FS(favicons))
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Debug().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Dur("duration_ms", time.Since(start)).
			Msg("request")
	}
}
