package api_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cermin-backend/internal/config"
	"cermin-backend/internal/router"

	"github.com/gin-gonic/gin"
)

const testJWTSecret = "test-secret"

func TestAdminRoutesRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{JWTSecret: testJWTSecret})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAdminCreateUserValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{JWTSecret: testJWTSecret})

	body := strings.NewReader(`{"name":"Budi","email":"wrong","password":"short"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", body)
	req.Header.Set("Content-Type", "application/json")
	setAuthHeader(t, req)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAdminListUsersQueryValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{JWTSecret: testJWTSecret})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?page=0", nil)
	setAuthHeader(t, req)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestAdminGetUserIDValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := router.Setup(nil, config.Config{JWTSecret: testJWTSecret})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/not-a-number", nil)
	setAuthHeader(t, req)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func setAuthHeader(t *testing.T, req *http.Request) {
	t.Helper()
	req.Header.Set("Authorization", "Bearer "+signedTestJWT(t))
}

func signedTestJWT(t *testing.T) string {
	t.Helper()

	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	payload := map[string]any{
		"sub":   "1",
		"email": "admin@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	encodedHeader := encodeJWTPart(t, header)
	encodedPayload := encodeJWTPart(t, payload)
	unsignedToken := encodedHeader + "." + encodedPayload

	mac := hmac.New(sha256.New, []byte(testJWTSecret))
	mac.Write([]byte(unsignedToken))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsignedToken + "." + signature
}

func encodeJWTPart(t *testing.T, value any) string {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal test jwt part: %v", err)
	}

	return base64.RawURLEncoding.EncodeToString(data)
}
