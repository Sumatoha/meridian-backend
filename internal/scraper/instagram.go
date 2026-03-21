package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Post represents a scraped Instagram post.
type Post struct {
	ID            string    `json:"id"`
	PostType      string    `json:"post_type"`
	Caption       string    `json:"caption"`
	Hashtags      []string  `json:"hashtags"`
	LikesCount    int       `json:"likes_count"`
	CommentsCount int       `json:"comments_count"`
	PostedAt      time.Time `json:"posted_at"`
	ThumbnailURL  string    `json:"thumbnail_url"`
}

// ProfileInfo represents basic scraped profile data.
type ProfileInfo struct {
	Username       string `json:"username"`
	ProfilePicURL  string `json:"profile_pic_url"`
	FollowersCount int    `json:"followers_count"`
}

// Scraper handles Instagram data collection.
type Scraper struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

// NewScraper creates a new Instagram scraper.
// For RapidAPI-based scraping, provide the API key and base URL.
func NewScraper(apiKey, baseURL string) *Scraper {
	return &Scraper{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     apiKey,
		baseURL:    baseURL,
	}
}

// ScrapeProfile retrieves profile info and recent posts for a username.
func (s *Scraper) ScrapeProfile(ctx context.Context, username string) (ProfileInfo, []Post, error) {
	profile, err := s.fetchProfile(ctx, username)
	if err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("scraper: fetch profile: %w", err)
	}

	posts, err := s.fetchPosts(ctx, username, 30)
	if err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("scraper: fetch posts: %w", err)
	}

	return profile, posts, nil
}

func (s *Scraper) fetchProfile(ctx context.Context, username string) (ProfileInfo, error) {
	url := fmt.Sprintf("%s/user/info?username=%s", s.baseURL, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ProfileInfo{}, err
	}
	req.Header.Set("x-rapidapi-key", s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ProfileInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ProfileInfo{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return ProfileInfo{}, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var profile ProfileInfo
	if err := json.Unmarshal(body, &profile); err != nil {
		return ProfileInfo{}, fmt.Errorf("parse profile: %w", err)
	}

	return profile, nil
}

func (s *Scraper) fetchPosts(ctx context.Context, username string, count int) ([]Post, error) {
	url := fmt.Sprintf("%s/user/posts?username=%s&count=%d", s.baseURL, username, count)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-rapidapi-key", s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var posts []Post
	if err := json.Unmarshal(body, &posts); err != nil {
		return nil, fmt.Errorf("parse posts: %w", err)
	}

	return posts, nil
}
