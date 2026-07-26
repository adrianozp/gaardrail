package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adrianozp/gaardrail/app/entities"
	handlers "github.com/adrianozp/gaardrail/app/handlers/controllerparams"
	"github.com/adrianozp/gaardrail/app/handlers/controllerparams/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func strptr(s string) *string { return &s }
func intptr(i int) *int       { return &i }

func setupRouter(h *handlers.ControllerParamsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handlers.RegisterRoutes(r, h)
	return r
}

func patch(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPatch, "/pid", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func get(r *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/pid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandleUpdate_SetpointFilterFieldsReachUseCase(t *testing.T) {
	uc := mocks.NewControllerParamsUseCase(t)
	uc.On("Update", entities.ControllerParams{SetpointFilterType: strptr("moving_average"), SetpointFilterSize: intptr(4)}).Return(nil)

	r := setupRouter(handlers.New(uc))
	w := patch(r, `{"setpoint_filter_type":"moving_average","setpoint_filter_size":4}`)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandleGet_SerializesSetpointFilterFieldsAndOmitsFilterSize(t *testing.T) {
	uc := mocks.NewControllerParamsUseCase(t)
	uc.On("Get").Return(entities.ControllerParams{SetpointFilterType: strptr("exponential"), SetpointFilterSize: intptr(2)})
	uc.On("CurrentType").Return("pid")

	r := setupRouter(handlers.New(uc))
	w := get(r)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.True(t, strings.Contains(body, `"setpoint_filter_type":"exponential"`))
	assert.True(t, strings.Contains(body, `"setpoint_filter_size":2`))
	assert.False(t, strings.Contains(body, `"filter_size"`))
}
