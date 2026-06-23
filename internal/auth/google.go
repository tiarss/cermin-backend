package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	googleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenURL    = "https://oauth2.googleapis.com/token"
	googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
)

type GoogleOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	State        string
}

type GoogleUserInfo struct {
	ID    string `json:"sub"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (g GoogleOAuth) AuthURL() string {
	values := url.Values{}
	values.Set("client_id", g.ClientID)
	values.Set("redirect_uri", g.RedirectURL)
	values.Set("response_type", "code")
	values.Set("scope", "openid email profile")
	values.Set("state", g.State)
	values.Set("access_type", "offline")
	values.Set("prompt", "consent")

	return googleAuthURL + "?" + values.Encode()
}

func (g GoogleOAuth) ExchangeCode(ctx context.Context, code string) (string, error) {
	payload := url.Values{}
	payload.Set("code", code)
	payload.Set("client_id", g.ClientID)
	payload.Set("client_secret", g.ClientSecret)
	payload.Set("redirect_uri", g.RedirectURL)
	payload.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("google token exchange failed: %s", bytes.TrimSpace(body))
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", err
	}

	return tokenResponse.AccessToken, nil
}

func (g GoogleOAuth) UserInfo(ctx context.Context, accessToken string) (*GoogleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

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
		return nil, fmt.Errorf("google userinfo failed: %s", bytes.TrimSpace(body))
	}

	var userInfo GoogleUserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}
