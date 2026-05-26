package handlers_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	handlers "github.com/adrianozp/gaardrail/app/handlers/switchcontroller"
	"github.com/adrianozp/gaardrail/app/handlers/switchcontroller/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter(h *handlers.SwitchControllerHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers.RegisterRoutes(r, h)
	return r
}

func do(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/controller/type", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandleSet_ValidRequestReturns204(t *testing.T) {
	uc := mocks.NewSwitchControllerUseCase(t)
	uc.On("Switch", "step").Return(nil)

	r := setupRouter(handlers.New(uc))
	w := do(r, `{"type":"step"}`)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandleSet_InvalidJSONReturns400(t *testing.T) {
	uc := mocks.NewSwitchControllerUseCase(t)

	r := setupRouter(handlers.New(uc))
	w := do(r, `{"type":`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "Switch")
}

func TestHandleSet_MissingTypeReturns400(t *testing.T) {
	uc := mocks.NewSwitchControllerUseCase(t)

	r := setupRouter(handlers.New(uc))
	w := do(r, `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	uc.AssertNotCalled(t, "Switch")
}

func TestHandleSet_UnknownTypeReturns422(t *testing.T) {
	uc := mocks.NewSwitchControllerUseCase(t)
	uc.On("Switch", "banana").Return(errors.New(`controller: unknown type "banana"`))

	r := setupRouter(handlers.New(uc))
	w := do(r, `{"type":"banana"}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.True(t, strings.Contains(w.Body.String(), "unknown type"))
}
