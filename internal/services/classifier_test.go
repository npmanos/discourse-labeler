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
								"logprob": -0.05, // Exp(-0.05) ~ 0.95
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
	if res.Probability < 0.90 || res.Probability > 0.96 {
		t.Errorf("expected Probability near 0.95, got %f", res.Probability)
	}
}
