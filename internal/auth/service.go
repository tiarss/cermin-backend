package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cermin-backend/internal/user"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyUsed   = errors.New("email already used")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Service struct {
	users     user.Repository
	jwtSecret string
}

func NewService(users user.Repository, jwtSecret string) *Service {
	return &Service{
		users:     users,
		jwtSecret: jwtSecret,
	}
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	Token string          `json:"token"`
	User  user.PublicUser `json:"user"`
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	existingUser, err := s.users.FindByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyUsed
	}

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	passwordHash := string(passwordHashBytes)
	createdUser, err := s.users.Create(ctx, user.CreateUserInput{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: &passwordHash,
		AuthProvider: "local",
	})
	if err != nil {
		return nil, err
	}

	return s.authResult(createdUser)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	foundUser, err := s.users.FindByEmail(ctx, input.Email)
	if errors.Is(err, user.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if foundUser.PasswordHash == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*foundUser.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.authResult(foundUser)
}

func (s *Service) LoginOrCreateGoogleUser(ctx context.Context, googleUser GoogleUserInfo) (*AuthResult, error) {
	foundUser, err := s.users.FindByGoogleID(ctx, googleUser.ID)
	if err == nil {
		return s.authResult(foundUser)
	}
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	existingEmailUser, err := s.users.FindByEmail(ctx, googleUser.Email)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}
	if existingEmailUser != nil {
		return nil, ErrEmailAlreadyUsed
	}

	googleID := googleUser.ID
	createdUser, err := s.users.Create(ctx, user.CreateUserInput{
		Name:         googleUser.Name,
		Email:        googleUser.Email,
		AuthProvider: "google",
		GoogleID:     &googleID,
	})
	if err != nil {
		return nil, err
	}

	return s.authResult(createdUser)
}

func (s *Service) authResult(foundUser *user.User) (*AuthResult, error) {
	token, err := s.createJWT(foundUser)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token: token,
		User:  user.ToPublicUser(foundUser),
	}, nil
}

func (s *Service) createJWT(foundUser *user.User) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	payload := map[string]any{
		"sub":   fmt.Sprintf("%d", foundUser.ID),
		"email": foundUser.Email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsignedToken := encodedHeader + "." + encodedPayload

	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	mac.Write([]byte(unsignedToken))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return strings.Join([]string{encodedHeader, encodedPayload, signature}, "."), nil
}
