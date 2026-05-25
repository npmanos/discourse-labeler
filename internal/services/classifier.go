package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/npmanos/discourse-labeler/internal/pipeline"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const sysPrompt = `You are a classification engine powering a network labeler. Your task is to analyze a social media post and determine if it qualifies as "Bluesky Meta-Discourse."

# DEFINITION: META-DISCOURSE (TRUE)
Meta-discourse consists of posts evaluating, criticizing, or theorizing about the cultural and social experience of the platform itself. This includes:
- Debating the "vibes," echo chambers, or user base behaviors of Bluesky.
- Comparing the social experience, engagement dynamics, or toxicity of Bluesky versus X (Twitter) or other platforms.
- Complaining about the types of conversations people have on the platform (e.g., "dead-end conversations", "too much drama", "people talking to themselves").
- Meta-commentary, subtweets, or reactions to other users' posts regarding Bluesky's culture.

# DEFINITION: NOT META-DISCOURSE (FALSE)
The following are strictly NOT meta-discourse:
- Technical discussions about building on the AT Protocol (atproto), creating custom feeds, using APIs, connecting to Jetstream, or hosting infrastructure.
- Announcements or discussions of new Bluesky application features (e.g., "DMs are now live", "how to use the new video player").
- General political, social, or pop culture arguments, even if they are toxic or reference platform moderation, as long as they are not explicitly analyzing the platform's culture as a whole.
- Ordinary posts using platform-specific terminology (like "skeet" or "repost") in passing.

# INSTRUCTIONS
Analyze the provided user post. You MUST consider the target post in the context of any provided parent posts or quoted posts. A target post which replies to or quotes a post which is meta discourse is likely also meta discourse. Output a valid JSON object containing exactly one boolean key: is_meta_discourse.`

type LLMClassifier struct {
	Client       *openai.Client
	Model        string
	SystemPrompt string
}

type LLMClassifierOption func(*LLMClassifier)

func WithSystemPrompt(prompt string) LLMClassifierOption {
	return func(lc *LLMClassifier) {
		lc.SystemPrompt = prompt
	}
}

func NewLLMClassifier(endpoint, model string, opts ...LLMClassifierOption) *LLMClassifier {
	client := openai.NewClient(
		option.WithBaseURL(endpoint),
		option.WithAPIKey("local-llama-nopass"),
	)
	lc := &LLMClassifier{
		Client: &client,
		Model:  model,
	}
	for _, opt := range opts {
		opt(lc)
	}
	return lc
}

type SchemaResponse struct {
	IsMetaDiscourse bool `json:"is_meta_discourse"`
}

func formatPostInput(hp *types.HydratedPost) string {
	var sb strings.Builder
	sb.WriteString("<posts>\n")
	if hp.HasParentContext && hp.ParentText != "" {
		sb.WriteString(fmt.Sprintf("  <post type=\"parent_post\">\n    %s\n  </post>\n", strings.TrimSpace(hp.ParentText)))
	}
	sb.WriteString(fmt.Sprintf("  <post type=\"target_post\">\n    %s\n  </post>\n", strings.TrimSpace(hp.TargetText)))
	if hp.QuotedText != "" {
		sb.WriteString(fmt.Sprintf("  <post type=\"quoted_post\">\n    %s\n  </post>\n", strings.TrimSpace(hp.QuotedText)))
	}
	sb.WriteString("</posts>")
	return sb.String()
}

func (lc *LLMClassifier) Classify(ctx context.Context, hp *types.HydratedPost) (*types.ClassificationResult, error) {
	if hp == nil {
		return nil, fmt.Errorf("hydrated post cannot be nil")
	}

	prompt := sysPrompt
	if lc.SystemPrompt != "" {
		prompt = lc.SystemPrompt
	}

	targetPost := formatPostInput(hp)

	// Build prompt message array with exactly 5 few-shot examples from the spec
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(prompt),
		// Example 1
		openai.UserMessage(`<posts>
  <post type="parent_post">
    i think, end of the day, the real problem with Bluesky is that most of its users are here *because* they want to be in a bubble. it's why despite the activity on here, the site still gives people bad vibes. X, despite it all, is still a more fun place.
  </post>
  <post type="target_post">
    it's why despite the activity on here, and more people clicking links, etc, the site still gives people bad vibes. the pile-ons are one thing, but those happen on all social media. it's that the typical mode here is one of distanced engagement, occasionally doing a 180 into angry gatekeeping.
  </post>
</posts>`),
		openai.AssistantMessage(`{"is_meta_discourse": true}`),
		// Example 2
		openai.UserMessage(`<posts>
  <post type="target_post">
    i'm not on bluesky because i want to live in a bubble. i'm on bluesky because i love reading long manifestos about what's wrong with bluesky by people who don't spend enough time here to know someone does this every 10 days.
  </post>
</posts>`),
		openai.AssistantMessage(`{"is_meta_discourse": true}`),
		// Example 3
		openai.UserMessage(`<posts>
  <post type="target_post">
    The Bluesky team just pushed an update for the new video player. You can now scrub through clips without the audio dropping out. Huge improvement over the beta version from last week.
  </post>
</posts>`),
		openai.AssistantMessage(`{"is_meta_discourse": false}`),
		// Example 4
		openai.UserMessage(`<posts>
  <post type="target_post">
    Every time I post about this election, my replies fill up with the worst takes imaginable. I can't believe people are actually defending this policy.
  </post>
</posts>`),
		openai.AssistantMessage(`{"is_meta_discourse": false}`),
		// Example 5
		openai.UserMessage(`<posts>
  <post type="target_post">
    The problem is that too many of you have lost faith in liberalism and the power of free speech and talking to the other side. Aaron Sorkin taught us that if you post hard enough you can actually force Elon Musk to change the way Twitter works.
  </post>
  <post type="quoted_post">
    i think, end of the day, the real problem with Bluesky is that most of its users are here because they want to be in a bubble. and they can tell themselves that it’s just about not being around Nazis it supporting Musk, and that’s part of it, but also there’s a palpable desire to be ensconced.
  </post>
</posts>`),
		openai.AssistantMessage(`{"is_meta_discourse": true}`),
		// Target Post
		openai.UserMessage(targetPost),
	}

	// Set up JSON Schema parameters
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "DiscourseSchema",
		Description: openai.String("Identifies if a post contains meta-discourse"),
		Strict:      openai.Bool(true),
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"is_meta_discourse": map[string]interface{}{
					"type": "boolean",
				},
			},
			"required":             []string{"is_meta_discourse"},
			"additionalProperties": false,
		},
	}

	resp, err := lc.Client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       lc.Model,
		Temperature: openai.Float(0.0),
		Messages:    messages,
		Logprobs:    openai.Bool(true),
		TopLogprobs: openai.Int(2),
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("llm classification request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty chat completion choices returned")
	}

	var schemaResp SchemaResponse
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &schemaResp); err != nil {
		return nil, fmt.Errorf("failed to parse schema response content: %w", err)
	}

	result := &types.ClassificationResult{
		Post:            hp,
		IsMetaDiscourse: schemaResp.IsMetaDiscourse,
		Probability:     1.0, // Default to 100% confidence if logprobs are absent
	}

	// Calculate probability from logprobs by finding the actual boolean token
	if len(resp.Choices[0].Logprobs.Content) > 0 {
		found := false
		for _, tc := range resp.Choices[0].Logprobs.Content {
			trimmed := strings.ToLower(strings.Trim(tc.Token, " \t\n\r\"'{}[]:,"))
			if trimmed == "true" || trimmed == "false" {
				result.Probability = math.Exp(tc.Logprob)
				found = true
				break
			}
		}
		if !found {
			// Fall back to the first token if a boolean token is not found
			result.Probability = math.Exp(resp.Choices[0].Logprobs.Content[0].Logprob)
		}
	}

	return result, nil
}
