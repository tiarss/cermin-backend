package auth

import (
	"errors"
	"strings"
	"testing"

	"cermin-backend/internal/user"
)

func TestParseJWTReturnsClaims(t *testing.T) {
	service := NewService(nil, "test-secret")
	token, err := service.createJWT(&user.User{
		ID:    42,
		Email: "budi@example.com",
	})
	if err != nil {
		t.Fatalf("create jwt: %v", err)
	}

	claims, err := service.ParseJWT(token)
	if err != nil {
		t.Fatalf("parse jwt: %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("expected user id %d, got %d", 42, claims.UserID)
	}
	if claims.Email != "budi@example.com" {
		t.Fatalf("expected email %q, got %q", "budi@example.com", claims.Email)
	}
}

func TestParseJWTRejectsTamperedToken(t *testing.T) {
	service := NewService(nil, "test-secret")
	token, err := service.createJWT(&user.User{
		ID:    42,
		Email: "budi@example.com",
	})
	if err != nil {
		t.Fatalf("create jwt: %v", err)
	}

	tamperedToken := strings.TrimSuffix(token, token[len(token)-1:]) + "x"
	_, err = service.ParseJWT(tamperedToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}
