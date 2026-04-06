package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	handlers "github.com/adrianozp/gaardrail/app/handlers/floodmessage"
	"github.com/adrianozp/gaardrail/app/handlers/floodmessage/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupRouter(h *handlers.FloodMessageHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers.RegisterFloodMessageRoutes(r, h)
	return r
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func TestHandle_ValidRequest_Returns202WithQueuedCount(t *testing.T) {
	done := make(chan struct{}, 3)
	uc := mocks.NewCreateMessageUseCase(t)
	uc.On("Create", mock.AnythingOfType("entities.Message")).
		Run(func(_ mock.Arguments) { done <- struct{}{} }).
		Return("id", nil)

	r := setupRouter(handlers.NewFloodMessageHandler(uc))
	req := httptest.NewRequest(http.MethodPost, "/messages/flood?quantity=3",
		jsonBody(t, map[string]string{"payload": "hello"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]int
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 3, resp["queued"])

	for i := 0; i < 3; i++ {
		select {
		case <-done:
		case <-t.Context().Done():
			t.Fatal("goroutine did not call Create 3 times")
		}
	}
}

func TestHandle_DefaultQuantityIsOne(t *testing.T) {
	done := make(chan struct{}, 1)
	uc := mocks.NewCreateMessageUseCase(t)
	uc.On("Create", mock.AnythingOfType("entities.Message")).
		Run(func(_ mock.Arguments) { done <- struct{}{} }).
		Return("id", nil)

	r := setupRouter(handlers.NewFloodMessageHandler(uc))
	req := httptest.NewRequest(http.MethodPost, "/messages/flood",
		jsonBody(t, map[string]string{"payload": "x"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var resp map[string]int
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 1, resp["queued"])

	select {
	case <-done:
	case <-t.Context().Done():
		t.Fatal("goroutine did not call Create")
	}
}

func TestHandle_QuantityExceedsMax_Returns400(t *testing.T) {
	uc := mocks.NewCreateMessageUseCase(t)
	r := setupRouter(handlers.NewFloodMessageHandler(uc))

	req := httptest.NewRequest(http.MethodPost, "/messages/flood?quantity=10001",
		jsonBody(t, map[string]string{"payload": "x"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "quantity exceeds max (10000)", resp["error"])
}

func TestHandle_QuantityBelowOne_Returns400(t *testing.T) {
	uc := mocks.NewCreateMessageUseCase(t)
	r := setupRouter(handlers.NewFloodMessageHandler(uc))

	req := httptest.NewRequest(http.MethodPost, "/messages/flood?quantity=0",
		jsonBody(t, map[string]string{"payload": "x"}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "quantity must be at least 1", resp["error"])
}

func TestHandle_MissingPayload_Returns400(t *testing.T) {
	uc := mocks.NewCreateMessageUseCase(t)
	r := setupRouter(handlers.NewFloodMessageHandler(uc))

	req := httptest.NewRequest(http.MethodPost, "/messages/flood?quantity=1",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
