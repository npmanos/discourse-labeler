package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/npmanos/discourse-labeler/internal/pipeline"
)

type SlingshotHydrator struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewSlingshotHydrator(baseURL string) *SlingshotHydrator {
	return &SlingshotHydrator{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type getRecordResponse struct {
	Value struct {
		Text string `json:"text"`
	} `json:"value"`
}

func (sh *SlingshotHydrator) Hydrate(ctx context.Context, ev *types.RawEvent) (*types.HydratedPost, error) {
	if ev.Commit == nil || ev.Commit.Collection != "app.bsky.feed.post" {
		return nil, fmt.Errorf("unsupported collection or nil commit")
	}

	var record types.BskyPostRecord
	if err := json.Unmarshal(ev.Commit.Record, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal post record: %w", err)
	}

	targetURI := fmt.Sprintf("at://%s/%s/%s", ev.Did, ev.Commit.Collection, ev.Commit.RKey)
	hp := &types.HydratedPost{
		TargetDID:   ev.Did,
		TargetRKey:  ev.Commit.RKey,
		TargetURI:   targetURI,
		TargetText:  record.Text,
		EventTimeUS: ev.TimeUS,
	}

	// Context 1: Parent post resolution
	if record.Reply != nil && record.Reply.Parent != nil {
		parentText, err := sh.fetchRecord(ctx, record.Reply.Parent.URI)
		if err == nil {
			hp.ParentText = parentText
			hp.HasParentContext = true
		} else {
			return nil, fmt.Errorf("parent hydration failed: %w", err)
		}
	}

	// Context 2: Quoted record resolution
	if record.Embed != nil && record.Embed.Type == "app.bsky.embed.record" && record.Embed.Record != nil {
		quotedText, err := sh.fetchRecord(ctx, record.Embed.Record.URI)
		if err == nil {
			hp.QuotedText = quotedText
		}
	}

	return hp, nil
}

func (sh *SlingshotHydrator) fetchRecord(ctx context.Context, atURI string) (string, error) {
	parts := strings.Split(strings.TrimPrefix(atURI, "at://"), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid AT-URI format: %s", atURI)
	}
	repo := parts[0]
	collection := parts[1]
	rkey := parts[2]

	endpoint := fmt.Sprintf("%s/xrpc/com.atproto.repo.getRecord", sh.BaseURL)
	reqURL, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	q := reqURL.Query()
	q.Set("repo", repo)
	q.Set("collection", collection)
	q.Set("rkey", rkey)
	reqURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := sh.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %s", resp.Status)
	}

	var res getRecordResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	return res.Value.Text, nil
}
