package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cermin-backend/internal/config"
	"cermin-backend/internal/router"

	"github.com/gin-gonic/gin"
)

func TestRegisterValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{})

	body := strings.NewReader(`{"name":"Budi","email":"wrong","password":"short"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGoogleOAuthRequiresConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/google", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}
