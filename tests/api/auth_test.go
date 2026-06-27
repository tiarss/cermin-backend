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

func TestAppleOAuthRequiresConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/apple", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestAppleOAuthRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{
		AppleClientID:    "com.tiarss.cerminapp",
		AppleTeamID:      "XZRB3HA52S",
		AppleKeyID:       "42VBGDX622",
		ApplePrivateKey:  "private-key-placeholder",
		AppleRedirectURL: "https://cermin-api.tiarlab.com/api/v1/auth/apple/callback",
		AppleOAuthState:  "state-value",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/apple", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d", http.StatusTemporaryRedirect, rec.Code)
	}

	location := rec.Header().Get("Location")
	if !strings.Contains(location, "https://appleid.apple.com/auth/authorize") {
		t.Fatalf("expected redirect to Apple, got %q", location)
	}
	if !strings.Contains(location, "client_id=com.tiarss.cerminapp") {
		t.Fatalf("expected client id in redirect, got %q", location)
	}
	if !strings.Contains(location, "response_mode=form_post") {
		t.Fatalf("expected form_post response mode in redirect, got %q", location)
	}
}
