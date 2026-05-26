package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	oc := NewOzoneClient(server.URL, "secret-token", "did:web:test-labeler")
	result := &types.ClassificationResult{
		Post: &types.HydratedPost{
			TargetURI: "at://did:plc:user/app.bsky.feed.post/123",
		},
		Probability: 0.90,
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
}
