package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	types "github.com/npmanos/discourse-labeler/internal/pipeline"
)

func TestSlingshotHydratorSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		repo := r.URL.Query().Get("repo")
		if repo != "did:plc:parent" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		response := map[string]interface{}{
			"value": map[string]string{
				"text": "Parent text content!",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	rawRecord := `{"text": "Target post text!", "reply": {"parent": {"uri": "at://did:plc:parent/app.bsky.feed.post/111", "cid": "abc"}}}`
	ev := &types.RawEvent{
		Did:    "did:plc:target",
		TimeUS: 100,
		Commit: &types.JetstreamCommit{
			RKey:       "222",
			Collection: "app.bsky.feed.post",
			Record:     []byte(rawRecord),
			CID:        "bafyreihunttf7a3uvtzrgbnyu2rzv24w4zx7xjwqgk4x5w7n5yvq7u7aua",
		},
	}

	hydrator := NewSlingshotHydrator(server.URL)
	hp, err := hydrator.Hydrate(context.Background(), ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hp.TargetText != "Target post text!" {
		t.Errorf("expected TargetText Target post text!, got %s", hp.TargetText)
	}
	if hp.TargetCID != "bafyreihunttf7a3uvtzrgbnyu2rzv24w4zx7xjwqgk4x5w7n5yvq7u7aua" {
		t.Errorf("expected TargetCID bafyreihunttf7a3uvtzrgbnyu2rzv24w4zx7xjwqgk4x5w7n5yvq7u7aua, got %s", hp.TargetCID)
	}
	if hp.ParentText != "Parent text content!" {
		t.Errorf("expected ParentText Parent text content!, got %s", hp.ParentText)
	}
	if !hp.HasParentContext {
		t.Error("expected HasParentContext to be true")
	}
}
