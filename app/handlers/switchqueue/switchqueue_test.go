package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handlers "github.com/adrianozp/gaardrail/app/handlers/switchqueue"
	"github.com/adrianozp/gaardrail/app/handlers/switchqueue/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter(h *handlers.SwitchQueueHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers.RegisterRoutes(r, h)
	return r
}

func put(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/queue/type", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func get(r *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/queue/type", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandleGet_ReturnsTypeAndAvailable(t *testing.T) {
	uc := mocks.NewSwitchQueueUseCase(t)
	uc.On("Current").Return("inmemory")
	uc.On("Available").Return([]string{"inmemory", "constant"})

	r := setupRouter(handlers.New(uc))
	w := get(r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"type":"inmemory"`)
	assert.Contains(t, w.Body.String(), `"available":["inmemory","constant"]`)
}

func TestHandleSet_ValidRequestReturns204(t *testing.T) {
	uc := mocks.NewSwitchQueueUseCase(t)
	uc.On("Switch", "constant").Return(nil)

	r := setupRouter(handlers.New(uc))
	w := put(r, `{"type":"constant"}`)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandleSet_InvalidJSONReturns400(t *testing.T) {
	uc := mocks.NewSwitchQueueUseCase(t)

	r := setupRouter(handlers.New(uc))
	w := put(r, `{"type":`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "Switch")
}

func TestHandleSet_MissingTypeReturns400(t *testing.T) {
	uc := mocks.NewSwitchQueueUseCase(t)

	r := setupRouter(handlers.New(uc))
	w := put(r, `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "Switch")
}

func TestHandleSet_UnknownTypeReturns422(t *testing.T) {
	uc := mocks.NewSwitchQueueUseCase(t)
	uc.On("Switch", "banana").Return(errors.New(`queue: unknown type "banana"`))

	r := setupRouter(handlers.New(uc))
	w := put(r, `{"type":"banana"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "unknown type"))
}
