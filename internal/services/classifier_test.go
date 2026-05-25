package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/npmanos/discourse-labeler/internal/pipeline"
)

func TestLLMClassifierMetaDiscourseTrue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `{"is_meta_discourse": true}`,
					},
					"logprobs": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"token":   "{",
								"logprob": -0.001,
							},
							{
								"token":   "\"is_meta_discourse\"",
								"logprob": -0.002,
							},
							{
								"token":   ":",
								"logprob": -0.003,
							},
							{
								"token":   "true",
								"logprob": -0.051293, // Exp(-0.051293) ~ 0.95
							},
							{
								"token":   "}",
								"logprob": -0.004,
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
		TargetText: "Bluesky is such a toxic bubble right now, standard bad vibes.",
	}

	res, err := classifier.Classify(context.Background(), hp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !res.IsMetaDiscourse {
		t.Error("expected IsMetaDiscourse to be true")
	}
	// Expected probability calculated from 'true' token (~0.95), not '{' (~0.999)
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
						"content": `{"is_meta_discourse": false}`,
					},
					"logprobs": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"token":   "{",
								"logprob": -0.001,
							},
							{
								"token":   "\"is_meta_discourse\"",
								"logprob": -0.002,
							},
							{
								"token":   ":",
								"logprob": -0.003,
							},
							{
								"token":   "false",
								"logprob": -0.10536, // Exp(-0.10536) ~ 0.90
							},
							{
								"token":   "}",
								"logprob": -0.004,
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

	if res.IsMetaDiscourse {
		t.Error("expected IsMetaDiscourse to be false")
	}
	// Expected probability calculated from 'false' token (~0.90)
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
