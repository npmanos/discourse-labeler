package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	types "github.com/npmanos/discourse-labeler/internal/pipeline"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const sysPrompt = `You are a content moderator applying classification labels to user posts. Your task is to analyze a social media post and determine if it qualifies as "Bluesky Meta-Discourse."

# DEFINITION: META-DISCOURSE
Meta-discourse consists of posts evaluating, criticizing, or theorizing about the cultural and social experience of the platform itself. This includes:
*   Debating the "vibes," echo chambers, or user base behaviors of Bluesky.
*   Comparing the social experience, engagement dynamics, or toxicity of Bluesky versus X (Twitter) or other platforms.
*   Complaining about the types of conversations people have on the platform (e.g., "dead-end conversations", "too much drama", "people talking to themselves").
*   Meta-commentary, subtweets, or reactions to other users' posts regarding Bluesky's culture.

# DEFINITION: NOT META-DISCOURSE
The following are strictly NOT meta-discourse:
*   Technical discussions about building on the AT Protocol (atproto), creating custom feeds, using APIs, connecting to Jetstream, or hosting infrastructure.
*   Announcements or discussions of new Bluesky application features (e.g., "DMs are now live", "how to use the new video player").
*   General political, social, or pop culture arguments, even if they are toxic or reference platform moderation, as long as they are not explicitly analyzing the platform's culture as a whole.
*   Ordinary posts using platform-specific terminology (like "skeet" or "repost") in passing.

# INSTRUCTIONS
1. Analyze the provided target post.
2. You MUST consider the target post in the context of any provided parent or quoted posts. A target post which replies to or quotes meta-discourse is likely also meta-discourse.
3. Write a brief reasoning step (maximum 2 sentences) analyzing the post against the definitions above. 
4. Assign one of the following classification labels: "definite_meta", "likely_meta", "not_meta", "unsure".

# EXPECTED OUTPUT FORMAT
You must output a valid JSON object matching this schema. The context_analysis block is mandatory. If a parent post or quote post is not provided in the input, you MUST output null for that specific field.

{
  "context_analysis": {
    "parent_post": { "reasoning": "...", "classification": "..." },
    "quote_post": { "reasoning": "...", "classification": "..." }
  },
  "target_post": {
    "reasoning": "...",
    "classification": "..."
  }
}`

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
	ContextAnalysis types.ContextAnalysis    `json:"context_analysis"`
	TargetPost      types.PostClassification `json:"target_post"`
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

	// Build prompt message array with few-shot examples
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
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": {
      "reasoning": "Explicitly evaluates Bluesky's user base, 'vibes', and compares the platform to X.",
      "classification": "definite_meta"
    },
    "quote_post": null
  },
  "target_post": {
    "reasoning": "Replies to meta-discourse and theorizes about Bluesky's specific engagement dynamics and gatekeeping.",
    "classification": "definite_meta"
  }
}`),
		// Example 2
		openai.UserMessage(`<posts>
  <post type="target_post">
    i'm not on bluesky because i want to live in a bubble. i'm on bluesky because i love reading long manifestos about what's wrong with bluesky by people who don't spend enough time here to know someone does this every 10 days.
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": null,
    "quote_post": null
  },
  "target_post": {
    "reasoning": "Provides meta-commentary and reaction to how other users critique Bluesky's culture.",
    "classification": "definite_meta"
  }
}`),
		// Example 3
		openai.UserMessage(`<posts>
  <post type="target_post">
    The Bluesky team just pushed an update for the new video player. You can now scrub through clips without the audio dropping out. Huge improvement over the beta version from last week.
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": null,
    "quote_post": null
  },
  "target_post": {
    "reasoning": "Discusses a new application feature, which is explicitly excluded from being meta-discourse.",
    "classification": "not_meta"
  }
}`),
		// Example 4
		openai.UserMessage(`<posts>
  <post type="target_post">
    Every time I post about this election, my replies fill up with the worst takes imaginable. I can't believe people are actually defending this policy.
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": null,
    "quote_post": null
  },
  "target_post": {
    "reasoning": "A general complaint about political arguments in replies, rather than an analysis of the platform's culture as a whole.",
    "classification": "not_meta"
  }
}`),
		// Example 5
		openai.UserMessage(`<posts>
  <post type="target_post">
    The problem is that too many of you have lost faith in liberalism and the power of free speech and talking to the other side. Aaron Sorkin taught us that if you post hard enough you can actually force Elon Musk to change the way Twitter works.
  </post>
  <post type="quoted_post">
    i think, end of the day, the real problem with Bluesky is that most of its users are here because they want to be in a bubble. and they can tell themselves that it’s just about not being around Nazis it supporting Musk, and that’s part of it, but also there’s a palpable desire to be ensconced.
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": null,
    "quote_post": {
      "reasoning": "Evaluates the desires and behaviors of the Bluesky user base.",
      "classification": "definite_meta"
    }
  },
  "target_post": {
    "reasoning": "Quotes meta-discourse and adds commentary comparing user approaches to speech on Twitter versus Bluesky.",
    "classification": "definite_meta"
  }
}`),
		// Example 6
		openai.UserMessage(`<posts>
  <post type="parent_post">
    I have the right one 🤓
https://bsky.app/profile/generalmusician.bsky.social/post/3lsrtbmb5q22k
  </post>
  <post type="target_post">
    I think this is how we became Bluesky friends 😅
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": {
      "reasoning": "Ordinary conversational post sharing a link with no analysis of platform culture.",
      "classification": "not_meta"
    },
    "quote_post": null
  },
  "target_post": {
    "reasoning": "Uses platform-specific terminology ('Bluesky friends') in passing without evaluating the social experience.",
    "classification": "not_meta"
  }
}`),
		// Target Post
		openai.UserMessage(targetPost),
	}

	// Set up JSON Schema parameters
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "DiscourseSchema",
		Description: openai.String("Identifies if a post contains meta-discourse with reasoning and context analysis"),
		Strict:      openai.Bool(true),
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"context_analysis": map[string]interface{}{
					"type":        "object",
					"description": "Mandatory analysis of parent or quoted posts. Null if not present.",
					"properties": map[string]interface{}{
						"parent_post": map[string]interface{}{
							"anyOf": []interface{}{
								map[string]interface{}{"type": "null"},
								map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"reasoning": map[string]interface{}{"type": "string"},
										"classification": map[string]interface{}{
											"type": "string",
											"enum": []string{"definite_meta", "likely_meta", "not_meta", "unsure"},
										},
									},
									"required":             []string{"reasoning", "classification"},
									"additionalProperties": false,
								},
							},
						},
						"quote_post": map[string]interface{}{
							"anyOf": []interface{}{
								map[string]interface{}{"type": "null"},
								map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"reasoning": map[string]interface{}{"type": "string"},
										"classification": map[string]interface{}{
											"type": "string",
											"enum": []string{"definite_meta", "likely_meta", "not_meta", "unsure"},
										},
									},
									"required":             []string{"reasoning", "classification"},
									"additionalProperties": false,
								},
							},
						},
					},
					"required":             []string{"parent_post", "quote_post"},
					"additionalProperties": false,
				},
				"target_post": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"reasoning": map[string]interface{}{"type": "string"},
						"classification": map[string]interface{}{
							"type": "string",
							"enum": []string{"definite_meta", "likely_meta", "not_meta", "unsure"},
						},
					},
					"required":             []string{"reasoning", "classification"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"context_analysis", "target_post"},
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
		Probability:     1.0, // Default to 100% confidence if logprobs are absent
		ContextAnalysis: schemaResp.ContextAnalysis,
		TargetPost:      schemaResp.TargetPost,
		IsMetaDiscourse: schemaResp.TargetPost.Classification == types.LabelDefiniteMeta || schemaResp.TargetPost.Classification == types.LabelLikelyMeta,
	}

	// Calculate probability from logprobs by finding the target classification enum
	if len(resp.Choices[0].Logprobs.Content) > 0 {
		found := false
		for _, tc := range resp.Choices[0].Logprobs.Content {
			trimmed := strings.ToLower(strings.Trim(tc.Token, " \t\n\r\"'{}[]:,"))
			if trimmed == "definite_meta" || trimmed == "likely_meta" || trimmed == "not_meta" || trimmed == "unsure" {
				result.Probability = math.Exp(tc.Logprob)
				found = true
				break
			}
		}
		if !found {
			// Fall back to the first token if not found
			result.Probability = math.Exp(resp.Choices[0].Logprobs.Content[0].Logprob)
		}
	}

	return result, nil
}
