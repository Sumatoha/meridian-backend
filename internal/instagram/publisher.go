package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const graphAPIBase = "https://graph.facebook.com/v21.0"

// Publisher handles Instagram Content Publishing API interactions.
type Publisher struct {
	httpClient *http.Client
	appID      string
	appSecret  string
}

// PublishResult contains the result of a publish operation.
type PublishResult struct {
	IGPostID  string `json:"id"`
	Permalink string `json:"permalink"`
}

func NewPublisher(appID, appSecret string) *Publisher {
	return &Publisher{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		appID:      appID,
		appSecret:  appSecret,
	}
}

// PublishPhoto publishes a single photo post.
func (p *Publisher) PublishPhoto(ctx context.Context, igUserID, accessToken, imageURL, caption string) (PublishResult, error) {
	// Step 1: Create media container
	containerID, err := p.createPhotoContainer(ctx, igUserID, accessToken, imageURL, caption)
	if err != nil {
		return PublishResult{}, fmt.Errorf("create photo container: %w", err)
	}

	// Step 2: Publish the container
	return p.publishContainer(ctx, igUserID, accessToken, containerID)
}

// PublishCarousel publishes a carousel post.
func (p *Publisher) PublishCarousel(ctx context.Context, igUserID, accessToken string, mediaURLs []string, caption string) (PublishResult, error) {
	// Step 1: Create item containers
	var children []string
	for _, mediaURL := range mediaURLs {
		childID, err := p.createPhotoContainer(ctx, igUserID, accessToken, mediaURL, "")
		if err != nil {
			return PublishResult{}, fmt.Errorf("create carousel child: %w", err)
		}
		children = append(children, childID)
	}

	// Step 2: Create carousel container
	containerID, err := p.createCarouselContainer(ctx, igUserID, accessToken, children, caption)
	if err != nil {
		return PublishResult{}, fmt.Errorf("create carousel container: %w", err)
	}

	// Step 3: Publish
	return p.publishContainer(ctx, igUserID, accessToken, containerID)
}

// PublishReels publishes a video reel.
func (p *Publisher) PublishReels(ctx context.Context, igUserID, accessToken, videoURL, caption string) (PublishResult, error) {
	containerID, err := p.createReelsContainer(ctx, igUserID, accessToken, videoURL, caption)
	if err != nil {
		return PublishResult{}, fmt.Errorf("create reels container: %w", err)
	}

	// Wait for video processing
	if err := p.waitForProcessing(ctx, accessToken, containerID); err != nil {
		return PublishResult{}, fmt.Errorf("wait for processing: %w", err)
	}

	return p.publishContainer(ctx, igUserID, accessToken, containerID)
}

// RefreshLongLivedToken refreshes a long-lived access token.
func (p *Publisher) RefreshLongLivedToken(ctx context.Context, token string) (string, time.Time, error) {
	params := url.Values{
		"grant_type":   {"fb_exchange_token"},
		"client_id":    {p.appID},
		"client_secret": {p.appSecret},
		"fb_exchange_token": {token},
	}

	reqURL := fmt.Sprintf("%s/oauth/access_token?%s", graphAPIBase, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", time.Time{}, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, err
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", time.Time{}, fmt.Errorf("parse token response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return result.AccessToken, expiresAt, nil
}

func (p *Publisher) createPhotoContainer(ctx context.Context, igUserID, accessToken, imageURL, caption string) (string, error) {
	params := url.Values{
		"image_url":    {imageURL},
		"access_token": {accessToken},
	}
	if caption != "" {
		params.Set("caption", caption)
	}

	return p.postContainer(ctx, igUserID, params)
}

func (p *Publisher) createCarouselContainer(ctx context.Context, igUserID, accessToken string, children []string, caption string) (string, error) {
	params := url.Values{
		"media_type":   {"CAROUSEL"},
		"children":     {strings.Join(children, ",")},
		"caption":      {caption},
		"access_token": {accessToken},
	}

	return p.postContainer(ctx, igUserID, params)
}

func (p *Publisher) createReelsContainer(ctx context.Context, igUserID, accessToken, videoURL, caption string) (string, error) {
	params := url.Values{
		"media_type":   {"REELS"},
		"video_url":    {videoURL},
		"caption":      {caption},
		"access_token": {accessToken},
	}

	return p.postContainer(ctx, igUserID, params)
}

func (p *Publisher) postContainer(ctx context.Context, igUserID string, params url.Values) (string, error) {
	reqURL := fmt.Sprintf("%s/%s/media", graphAPIBase, igUserID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Meta API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.ID, nil
}

func (p *Publisher) publishContainer(ctx context.Context, igUserID, accessToken, containerID string) (PublishResult, error) {
	params := url.Values{
		"creation_id":  {containerID},
		"access_token": {accessToken},
	}

	reqURL := fmt.Sprintf("%s/%s/media_publish", graphAPIBase, igUserID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(params.Encode()))
	if err != nil {
		return PublishResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return PublishResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PublishResult{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return PublishResult{}, fmt.Errorf("publish error (status %d): %s", resp.StatusCode, string(body))
	}

	var result PublishResult
	if err := json.Unmarshal(body, &result); err != nil {
		return PublishResult{}, err
	}

	return result, nil
}

func (p *Publisher) waitForProcessing(ctx context.Context, accessToken, containerID string) error {
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}

		reqURL := fmt.Sprintf("%s/%s?fields=status_code&access_token=%s", graphAPIBase, containerID, accessToken)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return err
		}

		resp, err := p.httpClient.Do(req)
		if err != nil {
			return err
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var status struct {
			StatusCode string `json:"status_code"`
		}
		if err := json.Unmarshal(body, &status); err != nil {
			return err
		}

		if status.StatusCode == "FINISHED" {
			return nil
		}
		if status.StatusCode == "ERROR" {
			return fmt.Errorf("video processing failed")
		}
	}

	return fmt.Errorf("video processing timeout")
}
