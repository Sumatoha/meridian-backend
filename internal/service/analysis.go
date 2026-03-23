package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/meridian/api/internal/ai"
	"github.com/meridian/api/internal/dto"
	"github.com/meridian/api/internal/instagram"
	"github.com/meridian/api/internal/repository"
	"github.com/meridian/api/internal/scraper"
)

type AnalysisService struct {
	queries  *repository.Queries
	aiClient *ai.Client
	scraper  *scraper.Scraper
	igReader *instagram.Reader
	logger   *slog.Logger
}

func NewAnalysisService(queries *repository.Queries, aiClient *ai.Client, sc *scraper.Scraper, igReader *instagram.Reader, logger *slog.Logger) *AnalysisService {
	return &AnalysisService{
		queries:  queries,
		aiClient: aiClient,
		scraper:  sc,
		igReader: igReader,
		logger:   logger,
	}
}

// AnalyzeProfile scrapes the profile, runs AI analysis, and stores the result.
func (s *AnalysisService) AnalyzeProfile(ctx context.Context, accountID uuid.UUID) error {
	account, err := s.queries.GetAccountByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("analysis: get account: %w", err)
	}

	// Get brand settings for niche/language
	settings, err := s.queries.GetBrandSettings(ctx, accountID)
	if err != nil {
		s.logger.Warn("no brand settings found, using defaults", slog.String("account_id", accountID.String()))
		settings = repository.BrandSetting{
			ContentLanguage: "ru",
		}
	}

	niche := "general"
	if settings.Niche != nil {
		niche = *settings.Niche
	}

	// Step 1: Fetch profile data (Graph API if OAuth, RapidAPI fallback)
	var profileInfo scraper.ProfileInfo
	var posts []scraper.Post

	if account.IsOauthConnected && account.AccessToken != nil && account.IgUserID != nil {
		s.logger.Info("analysis: using Graph API",
			slog.String("account_id", accountID.String()),
		)
		profileInfo, err = s.igReader.FetchProfile(ctx, *account.IgUserID, *account.AccessToken)
		if err != nil {
			return fmt.Errorf("analysis: graph api profile: %w", err)
		}
		posts, err = s.igReader.FetchPosts(ctx, *account.IgUserID, *account.AccessToken, 30)
		if err != nil {
			return fmt.Errorf("analysis: graph api posts: %w", err)
		}
	} else {
		s.logger.Info("analysis: using RapidAPI scraper",
			slog.String("account_id", accountID.String()),
		)
		profileInfo, posts, err = s.scraper.ScrapeProfile(ctx, account.IgUsername)
		if err != nil {
			return fmt.Errorf("analysis: scrape: %w", err)
		}
	}

	// Update profile info
	followersCount := int32(profileInfo.FollowersCount)
	s.queries.UpdateAccountProfile(ctx, repository.UpdateAccountProfileParams{
		ID:             accountID,
		ProfilePicUrl:  &profileInfo.ProfilePicURL,
		FollowersCount: &followersCount,
	})

	// Store scraped posts
	for _, post := range posts {
		hashtags := post.Hashtags
		if hashtags == nil {
			hashtags = []string{}
		}
		likesCount := int32(post.LikesCount)
		commentsCount := int32(post.CommentsCount)
		s.queries.UpsertScrapedPost(ctx, repository.UpsertScrapedPostParams{
			InstagramAccountID: accountID,
			IgPostID:           post.ID,
			PostType:           &post.PostType,
			Caption:            &post.Caption,
			Hashtags:           hashtags,
			LikesCount:         &likesCount,
			CommentsCount:      &commentsCount,
			PostedAt:           &post.PostedAt,
			ThumbnailUrl:       &post.ThumbnailURL,
		})
	}

	// Step 2: Build AI prompt
	postsJSON, err := json.Marshal(posts)
	if err != nil {
		return fmt.Errorf("analysis: marshal posts: %w", err)
	}

	systemPrompt, userPrompt := ai.BuildAnalysisPrompts(
		settings.ContentLanguage,
		account.IgUsername,
		niche,
		string(postsJSON),
	)

	// Step 3: Call AI
	rawResponse, err := s.aiClient.Generate(ctx, systemPrompt, userPrompt, 4096)
	if err != nil {
		return fmt.Errorf("analysis: AI generate: %w", err)
	}

	// Step 4: Parse response
	dna, err := ai.ParseJSON[brandDnaAIResponse](rawResponse)
	if err != nil {
		return fmt.Errorf("analysis: parse AI response: %w", err)
	}

	// Step 5: Store Brand DNA
	strengthsJSON, _ := json.Marshal(dna.Strengths)
	recommendationsJSON, _ := json.Marshal(dna.Recommendations)
	rawJSON := json.RawMessage(rawResponse)

	_, err = s.queries.CreateBrandDna(ctx, repository.CreateBrandDnaParams{
		InstagramAccountID: accountID,
		Score:              int32(dna.Score),
		Tone:               dna.Tone,
		VisualStyle:        &dna.VisualStyle,
		StrongTopics:       dna.StrongTopics,
		WeakAreas:          dna.WeakAreas,
		BestFormats:        dna.BestFormats,
		BestPostingTimes:   dna.BestPostingTimes,
		AvgPostingFrequency: &dna.AvgPostingFrequency,
		HashtagStrategy:    &dna.HashtagStrategy,
		Strengths:          strengthsJSON,
		Recommendations:    recommendationsJSON,
		RawAnalysis:        rawJSON,
	})
	if err != nil {
		return fmt.Errorf("analysis: store brand DNA: %w", err)
	}

	s.logger.Info("profile analysis completed",
		slog.String("account_id", accountID.String()),
		slog.Int("score", dna.Score),
	)

	return nil
}

// AnalyzePublic scrapes a public profile and runs AI analysis without requiring
// an account or storing anything in the database. Used for the free landing-page audit.
func (s *AnalysisService) AnalyzePublic(ctx context.Context, username string) (dto.PublicAuditResult, error) {
	s.logger.Info("public audit: starting",
		slog.String("username", username),
	)

	// Step 1: Scrape profile + posts via RapidAPI
	profileInfo, posts, err := s.scraper.ScrapeProfile(ctx, username)
	if err != nil {
		return dto.PublicAuditResult{}, fmt.Errorf("public audit: scrape: %w", err)
	}

	if len(posts) == 0 {
		return dto.PublicAuditResult{}, fmt.Errorf("public audit: no posts found for @%s", username)
	}

	// Step 2: Build AI prompt
	postsJSON, err := json.Marshal(posts)
	if err != nil {
		return dto.PublicAuditResult{}, fmt.Errorf("public audit: marshal posts: %w", err)
	}

	systemPrompt, userPrompt := ai.BuildAnalysisPrompts(
		"ru",
		username,
		"general",
		string(postsJSON),
	)

	// Step 3: Call AI
	rawResponse, err := s.aiClient.Generate(ctx, systemPrompt, userPrompt, 4096)
	if err != nil {
		return dto.PublicAuditResult{}, fmt.Errorf("public audit: AI generate: %w", err)
	}

	// Step 4: Parse response
	dna, err := ai.ParseJSON[brandDnaAIResponse](rawResponse)
	if err != nil {
		return dto.PublicAuditResult{}, fmt.Errorf("public audit: parse AI response: %w", err)
	}

	// Cap strengths and recommendations for free audit
	strengths := dna.Strengths
	if len(strengths) > 3 {
		strengths = strengths[:3]
	}
	recommendations := dna.Recommendations
	if len(recommendations) > 3 {
		recommendations = recommendations[:3]
	}

	s.logger.Info("public audit: completed",
		slog.String("username", username),
		slog.Int("score", dna.Score),
		slog.Int("followers", profileInfo.FollowersCount),
	)

	return dto.PublicAuditResult{
		ID:              uuid.New().String(),
		Username:        username,
		Score:           dna.Score,
		Strengths:       strengths,
		Recommendations: recommendations,
	}, nil
}

// GetLatestAnalysis returns the most recent Brand DNA for an account.
func (s *AnalysisService) GetLatestAnalysis(ctx context.Context, accountID uuid.UUID) (dto.BrandDnaDTO, error) {
	dna, err := s.queries.GetLatestBrandDna(ctx, accountID)
	if err != nil {
		return dto.BrandDnaDTO{}, fmt.Errorf("get analysis: %w", err)
	}

	return brandDnaToDTO(dna)
}

type brandDnaAIResponse struct {
	Score               int                `json:"score"`
	Tone                string             `json:"tone"`
	VisualStyle         string             `json:"visual_style"`
	StrongTopics        []string           `json:"strong_topics"`
	WeakAreas           []string           `json:"weak_areas"`
	BestFormats         []string           `json:"best_formats"`
	BestPostingTimes    []string           `json:"best_posting_times"`
	AvgPostingFrequency string             `json:"avg_posting_frequency"`
	HashtagStrategy     string             `json:"hashtag_strategy"`
	Strengths           []dto.Strength     `json:"strengths"`
	Recommendations     []dto.Recommendation `json:"recommendations"`
}

func brandDnaToDTO(d repository.BrandDna) (dto.BrandDnaDTO, error) {
	var strengths []dto.Strength
	if err := json.Unmarshal(d.Strengths, &strengths); err != nil {
		return dto.BrandDnaDTO{}, fmt.Errorf("unmarshal strengths: %w", err)
	}

	var recommendations []dto.Recommendation
	if err := json.Unmarshal(d.Recommendations, &recommendations); err != nil {
		return dto.BrandDnaDTO{}, fmt.Errorf("unmarshal recommendations: %w", err)
	}

	return dto.BrandDnaDTO{
		ID:               d.ID,
		Score:            int(d.Score),
		Tone:             d.Tone,
		VisualStyle:      d.VisualStyle,
		StrongTopics:     d.StrongTopics,
		WeakAreas:        d.WeakAreas,
		BestFormats:      d.BestFormats,
		BestPostingTimes: d.BestPostingTimes,
		Strengths:        strengths,
		Recommendations:  recommendations,
		CreatedAt:        d.CreatedAt,
	}, nil
}
