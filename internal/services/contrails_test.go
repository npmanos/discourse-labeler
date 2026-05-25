package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/npmanos/discourse-labeler/internal/pipeline"
)

var upgrader = websocket.Upgrader{}

func TestContrailsIngesterStream(t *testing.T) {
	// Spin up a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		payload := types.RawEvent{
			Did:    "did:plc:test",
			TimeUS: 123456,
			Type:   "commit",
		}
		_ = c.WriteJSON(payload)
	}))
	defer server.Close()

	wsURL := strings.Replace(server.URL, "http", "ws", 1)
	ingester := NewContrailsIngester(wsURL, "at://test-feed")

	out := make(chan *types.RawEvent, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		_ = ingester.Start(ctx, 0, out)
	}()

	select {
	case ev := <-out:
		if ev.Did != "did:plc:test" {
			t.Errorf("expected DID did:plc:test, got %s", ev.Did)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}
