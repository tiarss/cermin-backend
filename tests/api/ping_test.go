package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cermin-backend/internal/config"
	"cermin-backend/internal/router"

	"github.com/gin-gonic/gin"
)

func TestPingRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["message"] != "pong" {
		t.Fatalf("expected message pong, got %q", body["message"])
	}
}
