package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/meridian/api/internal/scraper"
)

// Reader fetches Instagram data via the Graph API for OAuth-connected accounts.
type Reader struct {
	httpClient *http.Client
}

func NewReader() *Reader {
	return &Reader{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// FetchProfile returns profile info using the Graph API.
func (r *Reader) FetchProfile(ctx context.Context, igUserID, accessToken string) (scraper.ProfileInfo, error) {
	params := url.Values{
		"fields":       {"username,profile_picture_url,followers_count"},
		"access_token": {accessToken},
	}

	reqURL := fmt.Sprintf("%s/%s?%s", graphAPIBase, igUserID, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return scraper.ProfileInfo{}, fmt.Errorf("build profile request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return scraper.ProfileInfo{}, fmt.Errorf("profile request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return scraper.ProfileInfo{}, fmt.Errorf("read profile response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return scraper.ProfileInfo{}, fmt.Errorf("profile fetch failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Username        string `json:"username"`
		ProfilePicURL   string `json:"profile_picture_url"`
		FollowersCount  int    `json:"followers_count"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return scraper.ProfileInfo{}, fmt.Errorf("parse profile: %w", err)
	}

	return scraper.ProfileInfo{
		Username:       result.Username,
		ProfilePicURL:  result.ProfilePicURL,
		FollowersCount: result.FollowersCount,
	}, nil
}

// FetchPosts returns recent posts using the Graph API.
func (r *Reader) FetchPosts(ctx context.Context, igUserID, accessToken string, limit int) ([]scraper.Post, error) {
	params := url.Values{
		"fields":       {"id,media_type,caption,timestamp,like_count,comments_count,thumbnail_url,permalink"},
		"limit":        {fmt.Sprintf("%d", limit)},
		"access_token": {accessToken},
	}

	reqURL := fmt.Sprintf("%s/%s/media?%s", graphAPIBase, igUserID, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build posts request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("posts request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read posts response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("posts fetch failed (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			ID            string `json:"id"`
			MediaType     string `json:"media_type"`
			Caption       string `json:"caption"`
			Timestamp     string `json:"timestamp"`
			LikeCount     int    `json:"like_count"`
			CommentsCount int    `json:"comments_count"`
			ThumbnailURL  string `json:"thumbnail_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse posts: %w", err)
	}

	posts := make([]scraper.Post, 0, len(result.Data))
	for _, item := range result.Data {
		postedAt, _ := time.Parse(time.RFC3339, item.Timestamp)

		mediaType := mapMediaType(item.MediaType)

		posts = append(posts, scraper.Post{
			ID:            item.ID,
			PostType:      mediaType,
			Caption:       item.Caption,
			Hashtags:      scraper.ExtractHashtags(item.Caption),
			LikesCount:    item.LikeCount,
			CommentsCount: item.CommentsCount,
			PostedAt:      postedAt,
			ThumbnailURL:  item.ThumbnailURL,
		})
	}

	return posts, nil
}

func mapMediaType(graphType string) string {
	switch graphType {
	case "IMAGE":
		return "photo"
	case "VIDEO":
		return "reels"
	case "CAROUSEL_ALBUM":
		return "carousel"
	default:
		return "photo"
	}
}
