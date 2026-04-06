package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adrianozp/gaardrail/internal/httpserver"
	"github.com/adrianozp/gaardrail/pkg/config"
)

func TestGetIndex(t *testing.T) {
	cfg := config.Config{
		Grafana: config.Grafana{URL: "http://localhost:3000/d/test/test?kiosk"},
	}
	router := httpserver.New(cfg)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}

	if !strings.Contains(w.Body.String(), "<html") {
		t.Error("expected response body to contain <html")
	}
}
