package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const instagramOAuthBase = "https://www.instagram.com/oauth/authorize"
const instagramTokenURL = "https://api.instagram.com/oauth/access_token"
const instagramGraphBase = "https://graph.instagram.com"

// OAuthClient handles Meta/Instagram OAuth token operations.
type OAuthClient struct {
	httpClient  *http.Client
	appID       string
	appSecret   string
	redirectURI string
}

func NewOAuthClient(appID, appSecret, redirectURI string) *OAuthClient {
	return &OAuthClient{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		appID:       appID,
		appSecret:   appSecret,
		redirectURI: redirectURI,
	}
}

// BuildAuthURL constructs the Instagram OAuth authorization URL.
func (c *OAuthClient) BuildAuthURL(state string) string {
	params := url.Values{
		"client_id":     {c.appID},
		"redirect_uri":  {c.redirectURI},
		"response_type": {"code"},
		"scope":         {"instagram_business_basic,instagram_business_content_publish"},
		"state":         {state},
	}
	authURL := instagramOAuthBase + "?" + params.Encode()
	slog.Info("oauth: built auth URL",
		slog.String("redirect_uri", c.redirectURI),
		slog.String("auth_url", authURL),
	)
	return authURL
}

// ExchangeCode exchanges an authorization code for a short-lived access token.
func (c *OAuthClient) ExchangeCode(ctx context.Context, code string) (string, string, error) {
	data := url.Values{}
	data.Set("client_id", c.appID)
	data.Set("client_secret", c.appSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("redirect_uri", c.redirectURI)
	data.Set("code", code)

	slog.Info("oauth: exchanging code",
		slog.String("redirect_uri", c.redirectURI),
		slog.String("token_url", instagramTokenURL),
		slog.Int("code_len", len(code)),
		slog.String("code_prefix", code[:min(20, len(code))]),
		slog.String("code_suffix", code[max(0, len(code)-20):]),
	)

	resp, err := http.PostForm(instagramTokenURL, data)
	if err != nil {
		return "", "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		UserID      int64  `json:"user_id"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("parse token response: %w", err)
	}

	return result.AccessToken, fmt.Sprintf("%d", result.UserID), nil
}

// ExchangeLongLivedToken swaps a short-lived token for a 60-day long-lived token.
func (c *OAuthClient) ExchangeLongLivedToken(ctx context.Context, shortToken string) (string, time.Time, error) {
	params := url.Values{
		"grant_type":    {"ig_exchange_token"},
		"client_secret": {c.appSecret},
		"access_token":  {shortToken},
	}

	reqURL := instagramGraphBase + "/access_token?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("build long-lived token request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("long-lived token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("read long-lived token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("long-lived token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", time.Time{}, fmt.Errorf("parse long-lived token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return result.AccessToken, expiresAt, nil
}

// GetProfile fetches the authenticated user's Instagram profile.
func (c *OAuthClient) GetProfile(ctx context.Context, accessToken string) (igUserID, username, profilePicURL string, err error) {
	params := url.Values{
		"fields":       {"user_id,username,profile_picture_url"},
		"access_token": {accessToken},
	}

	reqURL := instagramGraphBase + "/me?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("build profile request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("profile request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", fmt.Errorf("read profile response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("profile fetch failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		UserID          string `json:"user_id"`
		Username        string `json:"username"`
		ProfilePicURL   string `json:"profile_picture_url"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", "", fmt.Errorf("parse profile response: %w", err)
	}

	return result.UserID, result.Username, result.ProfilePicURL, nil
}
