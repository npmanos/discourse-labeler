package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	types "github.com/npmanos/discourse-labeler/internal/pipeline"
)

func TestLLMClassifierMetaDiscourseTrue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{
							"context_analysis": {
								"parent_post": {
									"reasoning": "Explicitly analyzes Bluesky echo chambers",
									"classification": "definite_meta"
								},
								"quote_post": null
							},
							"target_post": {
								"reasoning": "Agrees and discusses platform vibes",
								"classification": "definite_meta"
							}
						}`,
					},
					"logprobs": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"token":   "definite_meta",
								"logprob": -0.051293, // Exp(-0.051293) ~ 0.95
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	classifier := NewLLMClassifier(server.URL+"/v1/", "test-model")
	hp := &types.HydratedPost{
		TargetText: "Bluesky has echo chambers",
	}

	res, err := classifier.Classify(context.Background(), hp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TargetPost.Classification != "definite_meta" {
		t.Errorf("expected Classification definite_meta, got %s", res.TargetPost.Classification)
	}
	if res.ContextAnalysis.ParentPost == nil || res.ContextAnalysis.ParentPost.Classification != "definite_meta" {
		t.Errorf("expected parent post definite_meta, got: %+v", res.ContextAnalysis.ParentPost)
	}
	if res.Probability < 0.94 || res.Probability > 0.96 {
		t.Errorf("expected Probability near 0.95, got %f", res.Probability)
	}
}

func TestLLMClassifierMetaDiscourseFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{
							"context_analysis": {
								"parent_post": null,
								"quote_post": null
							},
							"target_post": {
								"reasoning": "Discusses programming in Go and Jetstream, which is technical and not meta-discourse.",
								"classification": "not_meta"
							}
						}`,
					},
					"logprobs": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"token":   "not_meta",
								"logprob": -0.10536, // Exp(-0.10536) ~ 0.90
							},
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	classifier := NewLLMClassifier(server.URL+"/v1/", "test-model")
	hp := &types.HydratedPost{
		TargetText: "I'm setting up a local Jetstream ingestion tool and building feeds in Go.",
	}

	res, err := classifier.Classify(context.Background(), hp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TargetPost.Classification != "not_meta" {
		t.Errorf("expected Classification not_meta, got %s", res.TargetPost.Classification)
	}
	if res.ContextAnalysis.ParentPost != nil {
		t.Errorf("expected parent post to be nil, got: %+v", res.ContextAnalysis.ParentPost)
	}
	if res.ContextAnalysis.QuotePost != nil {
		t.Errorf("expected quote post to be nil, got: %+v", res.ContextAnalysis.QuotePost)
	}
	if res.Probability < 0.89 || res.Probability > 0.91 {
		t.Errorf("expected Probability near 0.90, got %f", res.Probability)
	}
}

func TestLLMClassifierNilPostError(t *testing.T) {
	classifier := NewLLMClassifier("http://localhost:8080/v1/", "test-model")
	_, err := classifier.Classify(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error when classifying a nil hydrated post, but got none")
	}
}

func TestLLMClassifierWithSystemPromptOverride(t *testing.T) {
	expectedOverride := "custom-system-prompt-here"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if len(reqBody.Messages) == 0 {
			t.Fatal("expected messages, got none")
		}
		if reqBody.Messages[0].Role != "system" {
			t.Errorf("expected first message to be system, got %s", reqBody.Messages[0].Role)
		}
		if reqBody.Messages[0].Content != expectedOverride {
			t.Errorf("expected system prompt %q, got %q", expectedOverride, reqBody.Messages[0].Content)
		}

		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{
							"context_analysis": {
								"parent_post": null,
								"quote_post": null
							},
							"target_post": {
								"reasoning": "some reasoning",
								"classification": "not_meta"
							}
						}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	classifier := NewLLMClassifier(server.URL+"/v1/", "test-model", WithSystemPrompt(expectedOverride))
	hp := &types.HydratedPost{
		TargetText: "test post content",
	}

	_, err := classifier.Classify(context.Background(), hp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLLMClassifierParentAndQuoted(t *testing.T) {
	expectedParent := "this is a parent message discussing bluesky vibes"
	expectedTarget := "replying to the parent with standard target text"
	expectedQuote := "quoting some other discourse here"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		// The last user message must contain the exact formatted XML payload
		lastMsgIdx := len(reqBody.Messages) - 1
		lastMessage := reqBody.Messages[lastMsgIdx]
		if lastMessage.Role != "user" {
			t.Errorf("expected last message to be user, got %s", lastMessage.Role)
		}

		expectedXML := fmt.Sprintf("<posts>\n  <post type=\"parent_post\">\n    %s\n  </post>\n  <post type=\"target_post\">\n    %s\n  </post>\n  <post type=\"quoted_post\">\n    %s\n  </post>\n</posts>", expectedParent, expectedTarget, expectedQuote)
		if lastMessage.Content != expectedXML {
			t.Errorf("expected XML target payload:\n%s\n\nGot:\n%s", expectedXML, lastMessage.Content)
		}

		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{
							"context_analysis": {
								"parent_post": {
									"reasoning": "Discusses bluesky vibes, which is meta-discourse",
									"classification": "definite_meta"
								},
								"quote_post": {
									"reasoning": "Discusses quoting some other discourse",
									"classification": "likely_meta"
								}
							},
							"target_post": {
								"reasoning": "Replies to parent with standard target text",
								"classification": "likely_meta"
							}
						}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	classifier := NewLLMClassifier(server.URL+"/v1/", "test-model")
	hp := &types.HydratedPost{
		ParentText:       expectedParent,
		TargetText:       expectedTarget,
		QuotedText:       expectedQuote,
		HasParentContext: true,
	}

	res, err := classifier.Classify(context.Background(), hp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.TargetPost.Classification != "likely_meta" {
		t.Errorf("expected classification likely_meta, got: %s", res.TargetPost.Classification)
	}
	if res.ContextAnalysis.ParentPost == nil || res.ContextAnalysis.ParentPost.Classification != "definite_meta" {
		t.Errorf("expected parent post classification definite_meta, got: %+v", res.ContextAnalysis.ParentPost)
	}
	if res.ContextAnalysis.QuotePost == nil || res.ContextAnalysis.QuotePost.Classification != "likely_meta" {
		t.Errorf("expected quote post classification likely_meta, got: %+v", res.ContextAnalysis.QuotePost)
	}
}

func TestLLMClassifierWithAPIKey(t *testing.T) {
	expectedAPIKey := "test-custom-api-key-123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expectedAuth := "Bearer " + expectedAPIKey
		if auth != expectedAuth {
			t.Errorf("expected Authorization header %q, got %q", expectedAuth, auth)
		}

		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{
							"context_analysis": {
								"parent_post": null,
								"quote_post": null
							},
							"target_post": {
								"reasoning": "some reasoning",
								"classification": "not_meta"
							}
						}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	classifier := NewLLMClassifier(server.URL+"/v1/", "test-model", WithAPIKey(expectedAPIKey))
	hp := &types.HydratedPost{
		TargetText: "test post content",
	}

	_, err := classifier.Classify(context.Background(), hp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
