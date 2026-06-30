package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cermin-backend/internal/user"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailAlreadyUsed   = errors.New("email already used")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("expired token")
	ErrOAuthProvider      = errors.New("oauth provider error")
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

type JWTClaims struct {
	UserID    int64
	Email     string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

type jwtPayload struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
	Exp     int64  `json:"exp"`
	Iat     int64  `json:"iat"`
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
	createdUser, err := s.users.Create(ctx, user.CreateUserRequest{
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

func (s *Service) LoginWithGoogleCode(ctx context.Context, google GoogleOAuth, code string) (*AuthResult, error) {
	accessToken, err := google.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthProvider, err)
	}

	googleUser, err := google.UserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthProvider, err)
	}

	return s.LoginOrCreateGoogleUser(ctx, *googleUser)
}

func (s *Service) LoginWithAppleCode(ctx context.Context, apple AppleOAuth, code string, userPayload string) (*AuthResult, error) {
	tokenResponse, err := apple.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthProvider, err)
	}

	appleUser, err := apple.UserInfo(ctx, tokenResponse.IDToken, userPayload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOAuthProvider, err)
	}

	return s.LoginOrCreateAppleUser(ctx, *appleUser)
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
	createdUser, err := s.users.Create(ctx, user.CreateUserRequest{
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

func (s *Service) LoginOrCreateAppleUser(ctx context.Context, appleUser AppleUserInfo) (*AuthResult, error) {
	foundUser, err := s.users.FindByAppleID(ctx, appleUser.ID)
	if err == nil {
		return s.authResult(foundUser)
	}
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	existingEmailUser, err := s.users.FindByEmail(ctx, appleUser.Email)
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}
	if existingEmailUser != nil {
		return nil, ErrEmailAlreadyUsed
	}

	appleID := appleUser.ID
	createdUser, err := s.users.Create(ctx, user.CreateUserRequest{
		Name:         appleUser.Name,
		Email:        appleUser.Email,
		AuthProvider: "apple",
		AppleID:      &appleID,
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

func (s *Service) ParseJWT(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	unsignedToken := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(s.jwtSecret))
	mac.Write([]byte(unsignedToken))
	expectedSignature := mac.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}
	if !hmac.Equal(signature, expectedSignature) {
		return nil, ErrInvalidToken
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(payload.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return nil, ErrInvalidToken
	}

	expiresAt := time.Unix(payload.Exp, 0)
	if time.Now().After(expiresAt) {
		return nil, ErrExpiredToken
	}

	return &JWTClaims{
		UserID:    userID,
		Email:     payload.Email,
		ExpiresAt: expiresAt,
		IssuedAt:  time.Unix(payload.Iat, 0),
	}, nil
}
