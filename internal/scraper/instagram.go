package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

var hashtagRegex = regexp.MustCompile(`#(\w+)`)

// ExtractHashtags parses hashtags from a caption string.
func ExtractHashtags(caption string) []string {
	matches := hashtagRegex.FindAllString(caption, -1)
	if matches == nil {
		return []string{}
	}
	return matches
}

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

// Scraper handles Instagram data collection via direct API + headless fallback.
type Scraper struct {
	httpClient *http.Client
	logger     *slog.Logger
}

func NewScraper(logger *slog.Logger) *Scraper {
	return &Scraper{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

// ScrapeProfile tries the direct API first, falls back to headless browser.
func (s *Scraper) ScrapeProfile(ctx context.Context, username string) (ProfileInfo, []Post, error) {
	// Step 1: Direct API (fast path)
	profile, posts, err := s.scrapeViaAPI(ctx, username)
	if err == nil {
		s.logger.Info("scraper: direct API succeeded",
			slog.String("username", username),
			slog.Int("posts", len(posts)),
		)
		return profile, posts, nil
	}

	s.logger.Warn("scraper: direct API failed, trying headless",
		slog.String("username", username),
		slog.String("error", err.Error()),
	)

	// Step 2: Headless browser (slow path)
	profile, posts, err = s.scrapeViaHeadless(ctx, username)
	if err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("scraper: all methods failed for @%s: %w", username, err)
	}

	s.logger.Info("scraper: headless succeeded",
		slog.String("username", username),
		slog.Int("posts", len(posts)),
	)
	return profile, posts, nil
}

// ── Direct API ──────────────────────────────────────────────────────────────

const igAppID = "936619743392459"

var userAgents = []string{
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.2 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:134.0) Gecko/20100101 Firefox/134.0",
}

var retryDelays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

func (s *Scraper) scrapeViaAPI(ctx context.Context, username string) (ProfileInfo, []Post, error) {
	apiURL := fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)

	var lastErr error
	for attempt := range retryDelays {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ProfileInfo{}, nil, ctx.Err()
			case <-time.After(retryDelays[attempt-1]):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return ProfileInfo{}, nil, fmt.Errorf("build request: %w", err)
		}

		ua := userAgents[rand.Intn(len(userAgents))]
		req.Header.Set("User-Agent", ua)
		req.Header.Set("x-ig-app-id", igAppID)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Referer", fmt.Sprintf("https://www.instagram.com/%s/", username))

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: request error: %w", attempt+1, err)
			s.logger.Warn("scraper: API attempt failed",
				slog.Int("attempt", attempt+1),
				slog.String("error", err.Error()),
			)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			profile, posts, err := parseWebProfileInfo(body)
			if err != nil {
				lastErr = fmt.Errorf("attempt %d: parse error: %w", attempt+1, err)
				s.logger.Warn("scraper: API parse failed",
					slog.Int("attempt", attempt+1),
					slog.String("error", err.Error()),
					slog.Int("body_len", len(body)),
				)
				continue
			}
			return profile, posts, nil
		}

		// Log first 200 chars of body for debugging blocked responses
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		lastErr = fmt.Errorf("attempt %d: status %d", attempt+1, resp.StatusCode)
		s.logger.Warn("scraper: API non-200",
			slog.Int("attempt", attempt+1),
			slog.Int("status", resp.StatusCode),
			slog.String("body_snippet", snippet),
		)
	}

	return ProfileInfo{}, nil, fmt.Errorf("direct API: %w", lastErr)
}

// parseWebProfileInfo extracts profile + posts from the Instagram web_profile_info response.
func parseWebProfileInfo(body []byte) (ProfileInfo, []Post, error) {
	var resp struct {
		Data struct {
			User struct {
				Username      string `json:"username"`
				FullName      string `json:"full_name"`
				ProfilePicURL string `json:"profile_pic_url_hd"`
				EdgeFollowedBy struct {
					Count int `json:"count"`
				} `json:"edge_followed_by"`
				EdgeOwnerToTimeline struct {
					Edges []struct {
						Node struct {
							ID            string `json:"id"`
							TypeName      string `json:"__typename"`
							DisplayURL    string `json:"display_url"`
							ThumbnailSrc  string `json:"thumbnail_src"`
							IsVideo       bool   `json:"is_video"`
							TakenAtTimestamp int64 `json:"taken_at_timestamp"`
							EdgeMediaToCaption struct {
								Edges []struct {
									Node struct {
										Text string `json:"text"`
									} `json:"node"`
								} `json:"edges"`
							} `json:"edge_media_to_caption"`
							EdgeMediaToComment struct {
								Count int `json:"count"`
							} `json:"edge_media_to_comment"`
							EdgeLikedBy struct {
								Count int `json:"count"`
							} `json:"edge_liked_by"`
							EdgeSidecarToChildren *struct {
								Edges []struct{} `json:"edges"`
							} `json:"edge_sidecar_to_children"`
						} `json:"node"`
					} `json:"edges"`
				} `json:"edge_owner_to_timeline_media"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("unmarshal web_profile_info: %w", err)
	}

	user := resp.Data.User
	if user.Username == "" {
		return ProfileInfo{}, nil, fmt.Errorf("empty user in response")
	}

	profile := ProfileInfo{
		Username:       user.Username,
		ProfilePicURL:  user.ProfilePicURL,
		FollowersCount: user.EdgeFollowedBy.Count,
	}

	posts := make([]Post, 0, len(user.EdgeOwnerToTimeline.Edges))
	for _, edge := range user.EdgeOwnerToTimeline.Edges {
		node := edge.Node

		caption := ""
		if len(node.EdgeMediaToCaption.Edges) > 0 {
			caption = node.EdgeMediaToCaption.Edges[0].Node.Text
		}

		postType := "photo"
		if node.IsVideo {
			postType = "reels"
		} else if node.EdgeSidecarToChildren != nil {
			postType = "carousel"
		}

		thumbnail := node.ThumbnailSrc
		if thumbnail == "" {
			thumbnail = node.DisplayURL
		}

		posts = append(posts, Post{
			ID:            node.ID,
			PostType:      postType,
			Caption:       caption,
			Hashtags:      ExtractHashtags(caption),
			LikesCount:    node.EdgeLikedBy.Count,
			CommentsCount: node.EdgeMediaToComment.Count,
			PostedAt:      time.Unix(node.TakenAtTimestamp, 0),
			ThumbnailURL:  thumbnail,
		})
	}

	return profile, posts, nil
}

// ── Headless browser fallback ───────────────────────────────────────────────

// sharedDataRegex matches window._sharedData or additional_data in page source.
var sharedDataRegex = regexp.MustCompile(`window\._sharedData\s*=\s*({.+?});</script>`)

func (s *Scraper) scrapeViaHeadless(ctx context.Context, username string) (ProfileInfo, []Post, error) {
	// Try system chromium first (for Docker/Railway), fall back to Rod's auto-download
	l := launcher.New().Headless(true).
		// Flags needed for running in containers without a display
		Set("no-sandbox").
		Set("disable-gpu").
		Set("disable-dev-shm-usage")

	// Check for system-installed chromium (ROD_BROWSER env or system path)
	if envBin := os.Getenv("ROD_BROWSER"); envBin != "" {
		l = l.Bin(envBin)
		s.logger.Info("scraper: using ROD_BROWSER", slog.String("path", envBin))
	} else if path, exists := launcher.LookPath(); exists {
		l = l.Bin(path)
		s.logger.Info("scraper: using system browser", slog.String("path", path))
	}

	url, err := l.Launch()
	if err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("headless: launch browser: %w", err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("headless: connect: %w", err)
	}
	defer browser.MustClose()

	page, err := browser.Page(proto.TargetCreateTarget{
		URL: fmt.Sprintf("https://www.instagram.com/%s/", username),
	})
	if err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("headless: open page: %w", err)
	}

	// Wait for page to load with timeout
	loadCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	page = page.Context(loadCtx)
	if err := page.WaitStable(2 * time.Second); err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("headless: page didn't stabilize: %w", err)
	}

	// Try to extract profile data from the page's JSON
	// Instagram embeds data in various script tags
	html, err := page.HTML()
	if err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("headless: get HTML: %w", err)
	}

	// Try _sharedData first
	if matches := sharedDataRegex.FindStringSubmatch(html); len(matches) > 1 {
		profile, posts, err := parseSharedData(matches[1])
		if err == nil && profile.Username != "" {
			return profile, posts, nil
		}
	}

	// Try extracting from the script tags with type="application/json"
	profile, posts, err := s.extractFromScriptTags(page)
	if err != nil {
		return ProfileInfo{}, nil, fmt.Errorf("headless: extract data: %w", err)
	}

	return profile, posts, nil
}

func (s *Scraper) extractFromScriptTags(page *rod.Page) (ProfileInfo, []Post, error) {
	scripts, err := page.Elements("script[type='application/json']")
	if err != nil {
		return ProfileInfo{}, nil, err
	}

	for _, script := range scripts {
		text, err := script.Text()
		if err != nil || text == "" {
			continue
		}

		// Look for user data in any JSON script tag
		if !strings.Contains(text, "edge_owner_to_timeline_media") &&
			!strings.Contains(text, "edge_followed_by") {
			continue
		}

		// Try to find user object anywhere in the JSON
		profile, posts, err := parseEmbeddedUserData(text)
		if err == nil && profile.Username != "" {
			return profile, posts, nil
		}
	}

	return ProfileInfo{}, nil, fmt.Errorf("no profile data found in page scripts")
}

// parseSharedData extracts from window._sharedData format.
func parseSharedData(raw string) (ProfileInfo, []Post, error) {
	var data struct {
		EntryData struct {
			ProfilePage []struct {
				Graphql struct {
					User json.RawMessage `json:"user"`
				} `json:"graphql"`
			} `json:"ProfilePage"`
		} `json:"entry_data"`
	}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return ProfileInfo{}, nil, err
	}
	if len(data.EntryData.ProfilePage) == 0 {
		return ProfileInfo{}, nil, fmt.Errorf("no ProfilePage in _sharedData")
	}

	// Re-wrap into web_profile_info format and reuse parser
	wrapper := fmt.Sprintf(`{"data":{"user":%s}}`, data.EntryData.ProfilePage[0].Graphql.User)
	return parseWebProfileInfo([]byte(wrapper))
}

// parseEmbeddedUserData tries to find user data in a JSON blob.
func parseEmbeddedUserData(raw string) (ProfileInfo, []Post, error) {
	// Instagram sometimes nests the user object differently.
	// Try direct web_profile_info format first.
	profile, posts, err := parseWebProfileInfo([]byte(raw))
	if err == nil && profile.Username != "" {
		return profile, posts, nil
	}

	// Try to find user key at any depth
	var generic map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &generic); err != nil {
		return ProfileInfo{}, nil, err
	}

	// Recursively search for a key containing edge_followed_by
	found := findUserJSON(generic, 3)
	if found != nil {
		wrapper := fmt.Sprintf(`{"data":{"user":%s}}`, string(found))
		return parseWebProfileInfo([]byte(wrapper))
	}

	return ProfileInfo{}, nil, fmt.Errorf("user object not found")
}

// findUserJSON recursively searches for a JSON object containing edge_followed_by.
func findUserJSON(obj map[string]json.RawMessage, depth int) json.RawMessage {
	if depth <= 0 {
		return nil
	}

	for key, val := range obj {
		if key == "user" || key == "edge_followed_by" {
			// This level or parent might be the user object
			if _, ok := obj["edge_followed_by"]; ok {
				// This IS the user object
				b, _ := json.Marshal(obj)
				return b
			}
			if key == "user" {
				return val
			}
		}

		var nested map[string]json.RawMessage
		if json.Unmarshal(val, &nested) == nil {
			if found := findUserJSON(nested, depth-1); found != nil {
				return found
			}
		}
	}

	return nil
}
