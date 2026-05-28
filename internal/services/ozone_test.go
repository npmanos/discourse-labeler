package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	types "github.com/npmanos/discourse-labeler/internal/pipeline"
)

func TestOzoneIsAlreadyLabeledTrue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/xrpc/com.atproto.label.queryLabels" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		response := map[string]interface{}{
			"labels": []map[string]interface{}{
				{
					"val": "meta-discourse",
					"src": "did:web:test-labeler",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	oc := NewOzoneClient(server.URL, "token", "did:web:test-labeler")
	labeled, err := oc.IsAlreadyLabeled(context.Background(), "at://did:plc:user/app.bsky.feed.post/123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !labeled {
		t.Error("expected IsAlreadyLabeled to be true")
	}
}

func TestOzoneEmitLabelSuccess(t *testing.T) {
	var capturedMethod string
	var capturedAuth string
	var capturedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")
		if r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&capturedPayload)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	oc := NewOzoneClient(server.URL, "secret-token", "did:web:test-labeler")
	result := &types.ClassificationResult{
		Post: &types.HydratedPost{
			TargetURI: "at://did:plc:user/app.bsky.feed.post/123",
		},
		Probability: 0.90,
		TargetPost: types.PostClassification{
			Classification: types.LabelDefiniteMeta,
			Reasoning:      "Target post explicitly meta-references thread dynamics.",
		},
	}

	err := oc.EmitLabel(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMethod != "POST" {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if capturedAuth != "Bearer secret-token" {
		t.Errorf("expected Bearer secret-token, got %s", capturedAuth)
	}

	event, ok := capturedPayload["event"].(map[string]interface{})
	if !ok {
		t.Fatal("expected payload to contain an event object")
	}

	if event["$type"] != "tools.ozone.moderation.defs#modEventLabel" {
		t.Errorf("expected event $type tools.ozone.moderation.defs#modEventLabel, got %v", event["$type"])
	}

	comment, ok := event["comment"].(string)
	if !ok {
		t.Fatal("expected event comment to be a string")
	}

	if !strings.Contains(comment, "Target post explicitly meta-references thread dynamics") {
		t.Errorf("expected comment to contain target post reasoning, got: %s", comment)
	}
}

func TestOzoneEmitEscalationSuccess(t *testing.T) {
	var capturedMethod string
	var capturedAuth string
	var capturedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")
		if r.Method == "POST" {
			_ = json.NewDecoder(r.Body).Decode(&capturedPayload)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	oc := NewOzoneClient(server.URL, "secret-token", "did:web:test-labeler")

	result := &types.ClassificationResult{
		Post: &types.HydratedPost{
			TargetURI: "at://did:plc:user/app.bsky.feed.post/123",
		},
		Probability: 0.82,
		TargetPost: types.PostClassification{
			Classification: types.LabelLikelyMeta,
			Reasoning:      "Target post discusses thread moderation.",
		},
		ContextAnalysis: types.ContextAnalysis{
			ParentPost: &types.PostClassification{
				Classification: types.LabelNotMeta,
				Reasoning:      "Simple conversation greeting.",
			},
			QuotePost: &types.PostClassification{
				Classification: types.LabelDefiniteMeta,
				Reasoning:      "Quoted post references feed algorithms.",
			},
		},
	}

	err := oc.EmitEscalation(context.Background(), result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedMethod != "POST" {
		t.Errorf("expected POST, got %s", capturedMethod)
	}
	if capturedAuth != "Bearer secret-token" {
		t.Errorf("expected Bearer secret-token, got %s", capturedAuth)
	}

	event, ok := capturedPayload["event"].(map[string]interface{})
	if !ok {
		t.Fatal("expected payload to contain an event object")
	}

	if event["$type"] != "tools.ozone.moderation.defs#modEventEscalate" {
		t.Errorf("expected event $type tools.ozone.moderation.defs#modEventEscalate, got %v", event["$type"])
	}

	if event["escalateTo"] != "admin" {
		t.Errorf("expected escalateTo field to be admin, got %v", event["escalateTo"])
	}

	comment, ok := event["comment"].(string)
	if !ok {
		t.Fatal("expected event comment to be a string")
	}

	expectedSubstrings := []string{
		"Target post discusses thread moderation",
		"Simple conversation greeting",
		"Quoted post references feed algorithms",
		"likely_meta",
		"not_meta",
		"definite_meta",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(comment, sub) {
			t.Errorf("expected comment to contain %q, but got:\n%s", sub, comment)
		}
	}
}
