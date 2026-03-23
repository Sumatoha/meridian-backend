package ai

import "fmt"

const analysisSystemPrompt = `You are Meridian, an expert Instagram strategist specializing in the CIS market (Kazakhstan, Russia, Uzbekistan). You analyze Instagram profiles and create Brand DNA reports.

RULES:
- Be specific and actionable in recommendations
- Reference actual post examples from the data when possible
- Score fairly: 90+ is exceptional, 70-80 is good, 50-69 needs work, below 50 is critical
- Write in %s
- Return ONLY valid JSON, no markdown, no backticks`

const analysisUserPrompt = `Analyze this Instagram profile.

Profile: @%s
Niche: %s
Language: %s
Posts data (last 30): %s

Return this exact JSON structure:
{
  "score": <int 0-100>,
  "tone": "<string>",
  "visual_style": "<string>",
  "strong_topics": ["<string>"],
  "weak_areas": ["<string>"],
  "best_formats": ["<string>"],
  "best_posting_times": ["<string>"],
  "avg_posting_frequency": "<string>",
  "hashtag_strategy": "<string>",
  "strengths": [{"title": "<string>", "description": "<string>"}],
  "recommendations": [{"title": "<string>", "description": "<string>", "priority": "high|medium|low"}]
}`

func BuildAnalysisPrompts(language, username, niche, postsJSON string) (system, user string) {
	system = fmt.Sprintf(analysisSystemPrompt, language)
	user = fmt.Sprintf(analysisUserPrompt, username, niche, language, postsJSON)
	return system, user
}

const planSystemPrompt = `You are Meridian, an expert Instagram content strategist for the CIS market. You create 30-day content plans with detailed, production-ready creative briefs.

BRAND RULES (follow strictly):
- Goal: %s — optimize every post toward this goal
- Tone: %s. %s
- Content mix: %d%% useful, %d%% selling, %d%% personal, %d%% entertaining
- Formats: %s
- Posting frequency: %s

BLACKLIST (NEVER violate these):
- NEVER mention these competitors: %s
- NEVER use these words: %s
- NEVER create content about: %s
- Additional restrictions: %s
- Custom rules: %s

BRAND CONTEXT:
- Products/services: %s
- Target audience: %s
- USP: %s
- Team: %s
- Location: %s, Hours: %s
- Upcoming events: %s

CRITICAL BRIEF REQUIREMENTS:
- Every brief must be so detailed that a person with a phone knows EXACTLY what to shoot
- For Reels: scene-by-scene breakdown with timing, on-screen text, transitions
- For Carousels: slide-by-slide description with what text/image goes on each slide
- For Photos: exact composition, angle, lighting, who/what is in frame, props needed
- Include mood, style reference, aspect ratio
- Captions must be natural, match the tone, include CTA
- Hashtags: 10-15, mix of popular and niche tags
- Schedule times based on best posting times
- Return ONLY valid JSON, no markdown, no backticks
- Write ALL content in %s`

const planUserPrompt = `Generate %d content slots from %s to %s.

Return ONLY a valid JSON array of objects with this structure:
[{
  "day_number": <int>,
  "scheduled_date": "<YYYY-MM-DD>",
  "scheduled_time": "<HH:MM>",
  "title": "<string>",
  "content_type": "useful|selling|personal|entertaining",
  "format": "reels|carousel|photo",
  "brief": {
    "visual_description": "<string>",
    "scene_by_scene": [{"scene": <int>, "description": "<string>", "on_screen_text": "<string>", "duration": "<string>"}],
    "mood": "<string>",
    "photo_direction": "<string>",
    "people_in_frame": "<string>",
    "props_needed": ["<string>"],
    "aspect_ratio": "<string>"
  },
  "caption": "<string>",
  "hashtags": ["<string>"],
  "cta": "<string>"
}]`

func BuildPlanUserPrompt(totalSlots int, startDate, endDate string) string {
	return fmt.Sprintf(planUserPrompt, totalSlots, startDate, endDate)
}

// ── Two-phase generation prompts ──

const skeletonUserPrompt = `Create a content plan skeleton for %d posts from %s to %s.

Think strategically: ensure proper content type balance, no repeated topics, consider holidays/events, and follow all brand rules.

Return ONLY a valid JSON array:
[{
  "day_number": <int>,
  "scheduled_date": "<YYYY-MM-DD>",
  "scheduled_time": "<HH:MM>",
  "title": "<short descriptive title>",
  "content_type": "useful|selling|personal|entertaining",
  "format": "reels|carousel|photo"
}]`

func BuildSkeletonUserPrompt(totalSlots int, startDate, endDate string) string {
	return fmt.Sprintf(skeletonUserPrompt, totalSlots, startDate, endDate)
}

const detailsUserPrompt = `Here is a content plan skeleton. Write detailed briefs ONLY for days %d through %d.

Full plan context (DO NOT repeat topics from other days):
%s

For each slot in your assigned range, return a JSON array with full details:
[{
  "day_number": <int>,
  "scheduled_date": "<YYYY-MM-DD>",
  "scheduled_time": "<HH:MM>",
  "title": "<string>",
  "content_type": "useful|selling|personal|entertaining",
  "format": "reels|carousel|photo",
  "brief": {
    "visual_description": "<detailed description>",
    "scene_by_scene": [{"scene": <int>, "description": "<string>", "on_screen_text": "<string>", "duration": "<string>"}],
    "mood": "<string>",
    "photo_direction": "<string>",
    "people_in_frame": "<string>",
    "props_needed": ["<string>"],
    "aspect_ratio": "<string>"
  },
  "caption": "<string>",
  "hashtags": ["<string>"],
  "cta": "<string>"
}]`

func BuildDetailsUserPrompt(dayFrom, dayTo int, skeletonJSON string) string {
	return fmt.Sprintf(detailsUserPrompt, dayFrom, dayTo, skeletonJSON)
}

const regenSystemAddendum = `
This slot was rejected. Here's what was there before: %s
Other approved slots in this plan: %s

Generate ONE completely new content slot that:
- Does NOT repeat any existing topic
- Maintains the overall content type balance
- Is a fresh, creative idea
- Follows all brand rules and blacklist

Return ONLY valid JSON (single object, NOT array).`

func BuildRegenAddendum(previousBrief, otherSlots string) string {
	return fmt.Sprintf(regenSystemAddendum, previousBrief, otherSlots)
}

// PlanSystemPromptTemplate returns the raw format string for the plan system prompt.
// The caller is responsible for supplying all %s/%d placeholders.
func PlanSystemPromptTemplate() string {
	return planSystemPrompt
}
