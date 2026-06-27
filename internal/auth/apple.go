package auth

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	appleAuthURL  = "https://appleid.apple.com/auth/authorize"
	appleTokenURL = "https://appleid.apple.com/auth/token"
	appleKeysURL  = "https://appleid.apple.com/auth/keys"
	appleIssuer   = "https://appleid.apple.com"
)

type AppleOAuth struct {
	ClientID    string
	TeamID      string
	KeyID       string
	PrivateKey  string
	RedirectURL string
	State       string
}

type AppleTokenResponse struct {
	IDToken string `json:"id_token"`
}

type AppleUserInfo struct {
	ID    string
	Name  string
	Email string
}

type appleIDTokenClaims struct {
	Issuer string `json:"iss"`
	Aud    string `json:"aud"`
	Sub    string `json:"sub"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
}

type appleJWTHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type appleKeysResponse struct {
	Keys []appleJWK `json:"keys"`
}

type appleJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Crv string `json:"crv"`
}

func (a AppleOAuth) IsConfigured() bool {
	return a.ClientID != "" && a.TeamID != "" && a.KeyID != "" && a.PrivateKey != "" && a.RedirectURL != ""
}

func (a AppleOAuth) AuthURL() string {
	values := url.Values{}
	values.Set("client_id", a.ClientID)
	values.Set("redirect_uri", a.RedirectURL)
	values.Set("response_type", "code")
	values.Set("response_mode", "form_post")
	values.Set("scope", "name email")
	values.Set("state", a.State)

	return appleAuthURL + "?" + values.Encode()
}

func (a AppleOAuth) ExchangeCode(ctx context.Context, code string) (*AppleTokenResponse, error) {
	clientSecret, err := a.ClientSecret(time.Now())
	if err != nil {
		return nil, err
	}

	payload := url.Values{}
	payload.Set("code", code)
	payload.Set("client_id", a.ClientID)
	payload.Set("client_secret", clientSecret)
	payload.Set("redirect_uri", a.RedirectURL)
	payload.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appleTokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("apple token exchange failed: %s", bytes.TrimSpace(body))
	}

	var tokenResponse AppleTokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, err
	}
	if tokenResponse.IDToken == "" {
		return nil, errors.New("apple token exchange did not return an id token")
	}

	return &tokenResponse, nil
}

func (a AppleOAuth) UserInfo(ctx context.Context, idToken string, userPayload string) (*AppleUserInfo, error) {
	claims, err := a.ValidateIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	name, email := appleCallbackUserInfo(userPayload)
	if email == "" {
		email = claims.Email
	}
	if email == "" {
		return nil, errors.New("apple id token did not include an email")
	}
	if name == "" {
		name = appleDefaultName(email)
	}

	return &AppleUserInfo{
		ID:    claims.Sub,
		Name:  name,
		Email: email,
	}, nil
}

func (a AppleOAuth) ClientSecret(now time.Time) (string, error) {
	privateKey, err := a.parsePrivateKey()
	if err != nil {
		return "", err
	}

	header := map[string]string{
		"alg": "ES256",
		"kid": a.KeyID,
		"typ": "JWT",
	}
	payload := map[string]any{
		"iss": a.TeamID,
		"iat": now.Unix(),
		"exp": now.Add(180 * 24 * time.Hour).Unix(),
		"aud": appleIssuer,
		"sub": a.ClientID,
	}

	encodedHeader, err := encodeJWTPart(header)
	if err != nil {
		return "", err
	}
	encodedPayload, err := encodeJWTPart(payload)
	if err != nil {
		return "", err
	}

	signingInput := encodedHeader + "." + encodedPayload
	hash := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", err
	}

	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (a AppleOAuth) ValidateIDToken(ctx context.Context, idToken string) (*appleIDTokenClaims, error) {
	header, claims, signingInput, signature, err := parseAppleIDToken(idToken)
	if err != nil {
		return nil, err
	}

	if claims.Issuer != appleIssuer {
		return nil, errors.New("invalid apple id token issuer")
	}
	if claims.Aud != a.ClientID {
		return nil, errors.New("invalid apple id token audience")
	}
	if claims.Exp <= time.Now().Unix() {
		return nil, errors.New("expired apple id token")
	}
	if claims.Sub == "" {
		return nil, errors.New("apple id token did not include a subject")
	}

	key, err := fetchAppleSigningKey(ctx, header.Kid)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256([]byte(signingInput))
	switch header.Alg {
	case "RS256":
		rsaKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("apple signing key type does not match token algorithm")
		}
		if err := rsa.VerifyPKCS1v15(rsaKey, crypto.SHA256, hash[:], signature); err != nil {
			return nil, errors.New("invalid apple id token signature")
		}
	case "ES256":
		ecdsaKey, ok := key.(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("apple signing key type does not match token algorithm")
		}
		if len(signature) != 64 {
			return nil, errors.New("invalid apple id token signature length")
		}
		r := new(big.Int).SetBytes(signature[:32])
		s := new(big.Int).SetBytes(signature[32:])
		if !ecdsa.Verify(ecdsaKey, hash[:], r, s) {
			return nil, errors.New("invalid apple id token signature")
		}
	default:
		return nil, fmt.Errorf("unsupported apple id token algorithm: %s", header.Alg)
	}

	return claims, nil
}

func (a AppleOAuth) parsePrivateKey() (*ecdsa.PrivateKey, error) {
	privateKeyPEM := strings.ReplaceAll(a.PrivateKey, `\n`, "\n")
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse apple private key PEM")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse apple private key: %w", err)
	}

	privateKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("apple private key must be an ECDSA private key")
	}

	return privateKey, nil
}

func parseAppleIDToken(idToken string) (*appleJWTHeader, *appleIDTokenClaims, string, []byte, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, nil, "", nil, errors.New("invalid apple id token format")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, "", nil, err
	}
	var header appleJWTHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, nil, "", nil, err
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, nil, "", nil, err
	}
	var claims appleIDTokenClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, nil, "", nil, err
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, nil, "", nil, err
	}

	return &header, &claims, parts[0] + "." + parts[1], signature, nil
}

func fetchAppleSigningKey(ctx context.Context, kid string) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleKeysURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("apple keys request failed: %s", bytes.TrimSpace(body))
	}

	var keysResponse appleKeysResponse
	if err := json.Unmarshal(body, &keysResponse); err != nil {
		return nil, err
	}

	for _, key := range keysResponse.Keys {
		if key.Kid != kid {
			continue
		}

		switch key.Kty {
		case "RSA":
			return rsaPublicKey(key)
		case "EC":
			return ecdsaPublicKey(key)
		default:
			return nil, fmt.Errorf("unsupported apple signing key type: %s", key.Kty)
		}
	}

	return nil, errors.New("apple signing key not found")
}

func rsaPublicKey(key appleJWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}

	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, errors.New("invalid apple RSA exponent")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}

func ecdsaPublicKey(key appleJWK) (*ecdsa.PublicKey, error) {
	if key.Crv != "P-256" {
		return nil, fmt.Errorf("unsupported apple EC curve: %s", key.Crv)
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(key.X)
	if err != nil {
		return nil, err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(key.Y)
	if err != nil {
		return nil, err
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

func encodeJWTPart(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

func appleCallbackUserInfo(userPayload string) (string, string) {
	if userPayload == "" {
		return "", ""
	}

	var payload struct {
		Email string `json:"email"`
		Name  struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"name"`
	}
	if err := json.Unmarshal([]byte(userPayload), &payload); err != nil {
		return "", ""
	}

	name := strings.TrimSpace(strings.Join([]string{payload.Name.FirstName, payload.Name.LastName}, " "))
	return name, payload.Email
}

func appleDefaultName(email string) string {
	localPart, _, ok := strings.Cut(email, "@")
	if !ok || localPart == "" {
		return "Apple User"
	}

	return localPart
}
