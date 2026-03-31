package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenNSW/nsw/internal/auth"
	"github.com/OpenNSW/nsw/internal/config"
	"github.com/OpenNSW/nsw/internal/task/persistence"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHTTPHandler_HandleExecuteTask(t *testing.T) {
	t.Run("Invalid Method", func(t *testing.T) {
		tm, _, _, _ := setupTest(t)
		handler := NewHTTPHandler(tm, &config.AuthConfig{TraderGroup: "Trader", CHAGroup: "CHA"})
		req := httptest.NewRequest(http.MethodGet, "/execute", nil)

		userID := "cha-1"
		ctx := context.WithValue(req.Context(), auth.AuthContextKey, &auth.AuthContext{
			UserID: &userID, Groups: []string{"CHA"}, IsM2M: false,
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleExecuteTask(w, req)
		resp := w.Result()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("Invalid Body", func(t *testing.T) {
		tm, _, _, _ := setupTest(t)
		handler := NewHTTPHandler(tm, &config.AuthConfig{TraderGroup: "Trader", CHAGroup: "CHA"})
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBufferString("invalid json"))

		userID := "cha-1"
		ctx := context.WithValue(req.Context(), auth.AuthContextKey, &auth.AuthContext{
			UserID: &userID, Groups: []string{"CHA"}, IsM2M: false,
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		handler.HandleExecuteTask(w, req)
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestHTTPHandler_HandleGetTask(t *testing.T) {
	t.Run("Missing TaskID", func(t *testing.T) {
		tm, _, _, _ := setupTest(t)
		handler := NewHTTPHandler(tm, &config.AuthConfig{TraderGroup: "Trader", CHAGroup: "CHA"})
		req := httptest.NewRequest(http.MethodGet, "/tasks/", nil)

		userID := "trader-1"
		ctx := context.WithValue(req.Context(), auth.AuthContextKey, &auth.AuthContext{
			UserID: &userID, Groups: []string{"Trader"}, IsM2M: false,
		})
		req = req.WithContext(ctx)

		// No path value set
		w := httptest.NewRecorder()
		handler.HandleGetTask(w, req)
		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Invalid TaskID string", func(t *testing.T) {
		tm, _, mockStore, _ := setupTest(t)
		handler := NewHTTPHandler(tm, &config.AuthConfig{TraderGroup: "Trader", CHAGroup: "CHA"})
		req := httptest.NewRequest(http.MethodGet, "/tasks/invalid", nil)
		req.SetPathValue("id", "invalid")

		userID := "trader-1"
		ctx := context.WithValue(req.Context(), auth.AuthContextKey, &auth.AuthContext{
			UserID: &userID, Groups: []string{"Trader"}, IsM2M: false,
		})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		mockStore.On("GetByID", "invalid").Return(nil, errors.New("not found")).Once()

		handler.HandleGetTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
func withM2MContext(ctx context.Context, clientID string) context.Context {
	authCtx := &auth.AuthContext{
		ClientID: clientID,
		IsM2M:    true,
	}
	return context.WithValue(ctx, auth.AuthContextKey, authCtx)
}

func TestHTTPHandler_M2M_Bypass(t *testing.T) {
	tm, mockFactory, mockStore, mockPlugin := setupTest(t)
	handler := NewHTTPHandler(tm, &config.AuthConfig{TraderGroup: "Trader", CHAGroup: "CHA"})

	taskID := "task-1"
	req := httptest.NewRequest(http.MethodGet, "/tasks/"+taskID, nil)
	req.SetPathValue("id", taskID)
	req = req.WithContext(withM2MContext(req.Context(), "internal-client"))

	// Mock necessary for getTask to succeed past RBAC
	taskInfo := &persistence.TaskInfo{
		ID:    taskID,
		Type:  plugin.TaskTypeSimpleForm,
		State: plugin.InProgress,
	}
	mockStore.On("GetByID", taskID).Return(taskInfo, nil).Once()
	mockFactory.On("BuildExecutor", mock.Anything, taskInfo.Type, mock.Anything).Return(plugin.Executor{Plugin: mockPlugin}, nil).Once()
	mockStore.On("GetLocalState", taskID).Return(json.RawMessage(`{}`), nil).Once()
	mockStore.On("GetPluginState", taskID).Return("", nil).Once()
	mockPlugin.On("Init", mock.Anything).Return().Once()

	// Mock RenderInfo
	mockPlugin.On("GetRenderInfo", mock.Anything).Return(&plugin.ApiResponse{Success: true}, nil).Once()

	w := httptest.NewRecorder()
	handler.HandleGetTask(w, req)

	// Should be 200 OK because RBAC was bypassed and logic succeeded
	assert.Equal(t, http.StatusOK, w.Code)
}
