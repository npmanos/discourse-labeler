# Bluesky Meta-Discourse Labeler Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a highly concurrent, zero-data-loss, edge-classified Bluesky network labeler daemon in Go that streams filtered posts, resolves thread context, classifies meta-discourse using a local Gemma-4-e2b model, and emits signed labels to a decoupled Ozone instance.

**Architecture:** A concurrent Go pipeline structured around worker pools and buffered channels to separate high-volume ingestion, I/O-bound context hydration, compute-bound local LLM inference, and transactional XRPC label emission. Cursors are persisted atomically with a 10-second rewind recovery system, backed by a fast in-memory LRU cache and public query lookup to eliminate duplicate label emissions.

**Tech Stack:** Go 1.22, Gorilla WebSockets, OpenAI Go SDK, Google Distroless (Debian 13), Docker Compose, Llama.cpp.

---

## Technical Decompositions & File Maps

### Task 1: Go Module & Environment Configuration
* **Files:**
  * Create: `go.mod`
  * Create: `internal/config/config.go`
  * Create: `internal/config/config_test.go`
  * Create: `.env.example`

- [ ] **Step 1: Initialize the Go module**
  Run:
  ```bash
  go mod init github.com/npmanos/discourse-labeler
  ```
  Expected: Creates `go.mod` with Go version 1.22.

- [ ] **Step 2: Install core dependencies**
  Run:
  ```bash
  go get github.com/gorilla/websocket
  go get github.com/openai/openai-go
  ```
  Expected: Updates `go.mod` and `go.sum` with dependencies.

- [ ] **Step 3: Create the Config structure**
  Create `internal/config/config.go` containing:
  ```go
  package config

  import (
  	"fmt"
  	"os"
  	"strconv"
  )

  type Config struct {
  	Port                 string
  	LogLevel             string
  	CursorFilePath       string
  	CursorRewindSeconds  int
  	HydrationWorkers     int
  	ClassificationWorkers int
  	GrazeFeedURI         string
  	ContrailsWSURL       string
  	SlingshotURL         string
  	LLMEndpoint          string
  	LLMModel             string
  	LLMTemperature       float64
  	OzoneEndpoint        string
  	LabelerDID           string
  	OzoneAdminToken      string
  	DryRun               bool
  }

  func Load() (*Config, error) {
  	return &Config{
  		Port:                 getEnv("PORT", "8081"),
  		LogLevel:             getEnv("LOG_LEVEL", "info"),
  		CursorFilePath:       getEnv("CURSOR_FILE_PATH", "./data/cursor.json"),
  		CursorRewindSeconds:  getEnvInt("CURSOR_REWIND_SECONDS", 10),
  		HydrationWorkers:     getEnvInt("HYDRATION_WORKERS", 10),
  		ClassificationWorkers: getEnvInt("CLASSIFICATION_WORKERS", 4),
  		GrazeFeedURI:         getEnv("GRAZE_FEED_URI", ""),
  		ContrailsWSURL:       getEnv("CONTRAILS_WS_URL", "wss://api.graze.social/app/contrail"),
  		SlingshotURL:         getEnv("SLINGSHOT_URL", "https://slingshot.microcosm.blue"),
  		LLMEndpoint:          getEnv("LLM_ENDPOINT", "http://localhost:8080/v1/"),
  		LLMModel:             getEnv("LLM_MODEL", "google/gemma-4-e2b-gguf"),
  		LLMTemperature:       getEnvFloat("LLM_TEMPERATURE", 0.0),
  		OzoneEndpoint:        getEnv("OZONE_ENDPOINT", "http://localhost:3000"),
  		LabelerDID:           getEnv("LABELER_DID", ""),
  		OzoneAdminToken:      getEnv("OZONE_ADMIN_TOKEN", ""),
  		DryRun:               getEnvBool("DRY_RUN", "false"),
  	}, nil
  }

  func getEnv(key, fallback string) string {
  	if val, ok := os.LookupEnv(key); ok {
  		return val
  	}
  	return fallback
  }

  func getEnvInt(key string, fallback int) int {
  	valStr := getEnv(key, "")
  	if valStr == "" {
  		return fallback
  	}
  	val, err := strconv.Atoi(valStr)
  	if err != nil {
  		return fallback
  	}
  	return val
  }

  func getEnvFloat(key string, fallback float64) float64 {
  	valStr := getEnv(key, "")
  	if valStr == "" {
  		return fallback
  	}
  	val, err := strconv.ParseFloat(valStr, 64)
  	if err != nil {
  		return fallback
  	}
  	return val
  }

  func getEnvBool(key string, fallback string) bool {
  	valStr := getEnv(key, fallback)
  	val, err := strconv.ParseBool(valStr)
  	if err != nil {
  		return false
  	}
  	return val
  }
  ```

- [ ] **Step 4: Write config tests**
  Create `internal/config/config_test.go` containing:
  ```go
  package config

  import (
  	"os"
  	"testing"
  )

  func TestConfigLoadDefaults(t *testing.T) {
  	os.Clearenv()
  	cfg, err := Load()
  	if err != nil {
  		t.Fatalf("failed to load config: %v", err)
  	}
  	if cfg.Port != "8081" {
  		t.Errorf("expected default Port 8081, got %s", cfg.Port)
  	}
  	if cfg.DryRun != false {
  		t.Errorf("expected default DryRun false, got %t", cfg.DryRun)
  	}
  }

  func TestConfigLoadEnvOverrides(t *testing.T) {
  	os.Setenv("PORT", "9090")
  	os.Setenv("DRY_RUN", "true")
  	os.Setenv("CURSOR_REWIND_SECONDS", "30")
  	defer os.Clearenv()

  	cfg, err := Load()
  	if err != nil {
  		t.Fatalf("failed to load config: %v", err)
  	}
  	if cfg.Port != "9090" {
  		t.Errorf("expected Port 9090, got %s", cfg.Port)
  	}
  	if cfg.DryRun != true {
  		t.Errorf("expected DryRun true, got %t", cfg.DryRun)
  	}
  	if cfg.CursorRewindSeconds != 30 {
  		t.Errorf("expected CursorRewindSeconds 30, got %d", cfg.CursorRewindSeconds)
  	}
  }
  ```

- [ ] **Step 5: Run tests to verify config loading works**
  Run:
  ```bash
  go test -v ./internal/config/...
  ```
  Expected: PASS

- [ ] **Step 6: Write .env.example**
  Create `.env.example` with the environment variables listed in the design spec.

- [ ] **Step 7: Commit**
  Run:
  ```bash
  git add go.mod go.sum internal/config/ .env.example
  git commit -m "feat: initialize Go module and environment configuration"
  ```

---

### Task 2: Core Data Types
* **Files:**
  * Create: `internal/pipeline/types.go`

- [ ] **Step 1: Define types inside `internal/pipeline/types.go`**
  Create `internal/pipeline/types.go` containing:
  ```go
  package types

  import "encoding/json"

  // RawEvent represents the top-level envelope of a Jetstream event
  type RawEvent struct {
  	Did    string           `json:"did"`
  	TimeUS int64            `json:"time_us"`
  	Type   string           `json:"type"` // "commit", "identity", "account"
  	Commit *JetstreamCommit `json:"commit,omitempty"`
  }

  type JetstreamCommit struct {
  	Rev        string          `json:"rev"`
  	Type       string          `json:"type"` // "c" (create), "u" (update), "d" (delete)
  	Collection string          `json:"collection"` // e.g. "app.bsky.feed.post"
  	RKey       string          `json:"rkey"`
  	Record     json.RawMessage `json:"record,omitempty"`
  }

  type BskyPostRecord struct {
  	Type      string     `json:"$type"`
  	CreatedAt string     `json:"createdAt"`
  	Text      string     `json:"text"`
  	Reply     *BskyReply `json:"reply,omitempty"`
  	Embed     *BskyEmbed `json:"embed,omitempty"`
  }

  type BskyReply struct {
  	Parent *BskyLink `json:"parent,omitempty"`
  	Root   *BskyLink `json:"root,omitempty"`
  }

  type BskyEmbed struct {
  	Type   string    `json:"$type"` // e.g. "app.bsky.embed.record"
  	Record *BskyLink `json:"record,omitempty"`
  }

  type BskyLink struct {
  	CID string `json:"cid"`
  	URI string `json:"uri"`
  }

  // HydratedPost carries resolved context alongside the target post
  type HydratedPost struct {
  	TargetDID     string
  	TargetRKey    string
  	TargetURI     string
  	TargetText    string
  	ParentText    string
  	QuotedText    string
  	HasParentContext bool
  	EventTimeUS   int64
  }

  // ClassificationResult contains LLM evaluation metrics
  type ClassificationResult struct {
  	Post            *HydratedPost
  	IsMetaDiscourse bool
  	Probability     float64
  }
  ```

- [ ] **Step 2: Run build check**
  Run:
  ```bash
  go build ./internal/pipeline/...
  ```
  Expected: Compile successfully.

- [ ] **Step 3: Commit**
  Run:
  ```bash
  git add internal/pipeline/types.go
  git commit -m "feat: define pipeline data models for Jetstream, Slingshot, and LLM"
  ```

---

### Task 3: Ingestion Worker (Graze Contrails WebSocket)
* **Files:**
  * Create: `internal/services/contrails.go`
  * Create: `internal/services/contrails_test.go`

- [ ] **Step 1: Write Ingestion Service**
  Create `internal/services/contrails.go` containing:
  ```go
  package services

  import (
  	"context"
  	"fmt"
  	"log"
  	"net/url"
  	"time"

  	"github.com/gorilla/websocket"
  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
  )

  type ContrailsIngester struct {
  	WSURL   string
  	FeedURI string
  }

  func NewContrailsIngester(wsURL, feedURI string) *ContrailsIngester {
  	return &ContrailsIngester{
  		WSURL:   wsURL,
  		FeedURI: feedURI,
  	}
  }

  func (ci *ContrailsIngester) Start(ctx context.Context, cursor int64, out chan<- *types.RawEvent) error {
  	u, err := url.Parse(ci.WSURL)
  	if err != nil {
  		return fmt.Errorf("invalid contrails WS URL: %w", err)
  	}

  	q := u.Query()
  	q.Set("feed", ci.FeedURI)
  	if cursor > 0 {
  		q.Set("cursor", fmt.Sprintf("%d", cursor))
  	}
  	u.RawQuery = q.Encode()

  	log.Printf("Connecting to Graze Contrails: %s", u.String())

  	dialer := websocket.DefaultDialer
  	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
  	if err != nil {
  		return fmt.Errorf("contrails dial failure: %w", err)
  	}
  	defer conn.Close()

  	// Start read loop
  	errChan := make(chan error, 1)
  	go func() {
  		for {
  			var ev types.RawEvent
  			err := conn.ReadJSON(&ev)
  			if err != nil {
  				errChan <- err
  				return
  			}
  			select {
  			case out <- &ev:
  			case <-ctx.Done():
  				return
  			}
  		}
  	}()

  	select {
  	case err := <-errChan:
  		return fmt.Errorf("websocket read error: %w", err)
  	case <-ctx.Done():
  		return ctx.Err()
  	}
  }
  ```

- [ ] **Step 2: Write Ingestion Service Mock Tests**
  Create `internal/services/contrails_test.go` with a local WebSocket listener to verify parsing:
  ```go
  package services

  import (
  	"context"
  	"encoding/json"
  	"net/http"
  	"net/http/httptest"
  	"strings"
  	"testing"
  	"time"

  	"github.com/gorilla/websocket"
  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
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
  ```

- [ ] **Step 3: Run Ingester Tests**
  Run:
  ```bash
  go test -v ./internal/services/... -run TestContrailsIngester
  ```
  Expected: PASS

- [ ] **Step 4: Commit**
  Run:
  ```bash
  git add internal/services/contrails.go internal/services/contrails_test.go
  git commit -m "feat: implement Graze Contrails WebSocket ingestion client with tests"
  ```

---

### Task 4: Context Hydrator (Microcosm Slingshot Client)
* **Files:**
  * Create: `internal/services/slingshot.go`
  * Create: `internal/services/slingshot_test.go`

- [ ] **Step 1: Write the Context Hydrator**
  Create `internal/services/slingshot.go` containing:
  ```go
  package services

  import (
  	"context"
  	"encoding/json"
  	"fmt"
  	"net/http"
  	"net/url"
  	"strings"
  	"time"

  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
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
  ```

- [ ] **Step 2: Write Hydrator Tests**
  Create `internal/services/slingshot_test.go` with mock HTTP servers:
  ```go
  package services

  import (
  	"context"
  	"encoding/json"
  	"net/http"
  	"net/http/httptest"
  	"testing"

  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
  )

  func TestSlingshotHydratorSuccess(t *testing.T) {
  	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *testing.Request) {
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
  	if hp.ParentText != "Parent text content!" {
  		t.Errorf("expected ParentText Parent text content!, got %s", hp.ParentText)
  	}
  	if !hp.HasParentContext {
  		t.Error("expected HasParentContext to be true")
  	}
  }
  ```

- [ ] **Step 3: Run Hydrator Tests**
  Run:
  ```bash
  go test -v ./internal/services/... -run TestSlingshotHydrator
  ```
  Expected: PASS

- [ ] **Step 4: Commit**
  Run:
  ```bash
  git add internal/services/slingshot.go internal/services/slingshot_test.go
  git commit -m "feat: implement Slingshot context hydration client with tests"
  ```

---

### Task 5: LLM Classifier (Gemma-4 via Local Llama.cpp)
* **Files:**
  * Create: `internal/services/classifier.go`
  * Create: `internal/services/classifier_test.go`

- [ ] **Step 1: Write the Classifier Service**
  Create `internal/services/classifier.go` containing:
  ```go
  package services

  import (
  	"context"
  	"encoding/json"
  	"fmt"
  	"math"

  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
  	"github.com/openai/openai-go"
  	"github.com/openai/openai-go/option"
  )

  type LLMClassifier struct {
  	Client *openai.Client
  	Model  string
  }

  func NewLLMClassifier(endpoint, model string) *LLMClassifier {
  	return &LLMClassifier{
  		Client: openai.NewClient(
  			option.WithBaseURL(endpoint),
  			option.WithAPIKey("local-llama-nopass"),
  		),
  		Model: model,
  	}
  }

  type SchemaResponse struct {
  	IsMetaDiscourse bool `json:"is_meta_discourse"`
  }

  func (lc *LLMClassifier) Classify(ctx context.Context, hp *types.HydratedPost) (*types.ClassificationResult, error) {
  	sysPrompt := `You are a classification engine powering a network labeler. Your task is to analyze a social media post and determine if it qualifies as "Bluesky Meta-Discourse."

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
  Analyze the provided user post. Output a valid JSON object containing exactly one boolean key: is_meta_discourse.`

  	targetPost := hp.TargetText
  	if hp.HasParentContext {
  		targetPost = fmt.Sprintf("Context (Parent Post): %s\n\nTarget Post: %s", hp.ParentText, hp.TargetText)
  	}

  	// Build prompt message array
  	messages := []openai.ChatCompletionMessageParamUnion{
  		openai.SystemMessage(sysPrompt),
  		openai.UserMessage("i think, end of the day, the real problem with Bluesky is that most of its users are here *because* they want to be in a bubble. it's why despite the activity on here, the site still gives people bad vibes. X, despite it all, is still a more fun place."),
  		openai.AssistantMessage(`{"is_meta_discourse": true}`),
  		openai.UserMessage("Finally got my labeler up and running! I'm streaming Jetstream into a Go backend and using a local Ollama container to classify text. The atproto documentation for cryptographically signing the labels was a bit dense but I figured it out."),
  		openai.AssistantMessage(`{"is_meta_discourse": false}`),
  		openai.UserMessage(targetPost),
  	}

  	// Set up JSON Schema parameters
  	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
  		Name:        openai.F("DiscourseSchema"),
  		Description: openai.F("Identifies if a post contains meta-discourse"),
  		Strict:      openai.F(true),
  		Schema: openai.F(map[string]interface{}{
  			"type": "object",
  			"properties": map[string]interface{}{
  				"is_meta_discourse": map[string]interface{}{
  					"type": "boolean",
  				},
  			},
  			"required":             []string{"is_meta_discourse"},
  			"additionalProperties": false,
  		}),
  	}

  	resp, err := lc.Client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
  		Model:       openai.F(lc.Model),
  		Temperature: openai.F(0.0),
  		Messages:    openai.F(messages),
  		Logprobs:    openai.F(true),
  		TopLogprobs: openai.F(2),
  		ResponseFormat: openai.F(openai.ChatCompletionResponseFormatParam{
  			Type:       openai.F(openai.ChatCompletionResponseFormatTypeJSONSchema),
  			JSONSchema: openai.F(schemaParam),
  		}),
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

  	// Calculate probability from logprobs
  	if resp.Choices[0].Logprobs != nil && len(resp.Choices[0].Logprobs.Content) > 0 {
  		logprobsContent := resp.Choices[0].Logprobs.Content[0]
  		logprobVal := logprobsContent.Logprob
  		result.Probability = math.Exp(logprobVal)
  	}

  	return result, nil
  }
  ```

- [ ] **Step 2: Write Classifier Tests**
  Create `internal/services/classifier_test.go` with a mock HTTP Server for LLM responses:
  ```go
  package services

  import (
  	"context"
  	"encoding/json"
  	"net/http"
  	"net/http/httptest"
  	"testing"

  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
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
  ```

- [ ] **Step 3: Run Classifier Tests**
  Run:
  ```bash
  go test -v ./internal/services/... -run TestLLMClassifier
  ```
  Expected: PASS

- [ ] **Step 4: Commit**
  Run:
  ```bash
  git add internal/services/classifier.go internal/services/classifier_test.go
  git commit -m "feat: implement llama.cpp OpenAI-compatible classification client with tests"
  ```

---

### Task 6: Decoupled Ozone API Client
* **Files:**
  * Create: `internal/services/ozone.go`
  * Create: `internal/services/ozone_test.go`

- [ ] **Step 1: Write the Ozone Service Adapter**
  Create `internal/services/ozone.go` containing:
  ```go
  package services

  import (
  	"bytes"
  	"context"
  	"encoding/json"
  	"fmt"
  	"net/http"
  	"net/url"
  	"time"

  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
  )

  type OzoneClient struct {
  	Endpoint   string
  	AdminToken string
  	LabelerDID string
  	HTTPClient *http.Client
  }

  func NewOzoneClient(endpoint, adminToken, labelerDID string) *OzoneClient {
  	return &OzoneClient{
  		Endpoint:   endpoint,
  		AdminToken: adminToken,
  		LabelerDID: labelerDID,
  		HTTPClient: &http.Client{
  			Timeout: 5 * time.Second,
  		},
  	}
  }

  type queryLabelsResponse struct {
  	Labels []struct {
  		Val string `json:"val"`
  		Src string `json:"src"`
  	} `json:"labels"`
  }

  // IsAlreadyLabeled queries Ozone/Public AppView to check if we've already labeled this subject
  func (oc *OzoneClient) IsAlreadyLabeled(ctx context.Context, targetURI string) (bool, error) {
  	u, err := url.Parse(fmt.Sprintf("%s/xrpc/com.atproto.label.queryLabels", oc.Endpoint))
  	if err != nil {
  		return false, err
  	}

  	q := u.Query()
  	q.Set("uriPatterns", targetURI)
  	q.Set("sources", oc.LabelerDID)
  	u.RawQuery = q.Encode()

  	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
  	if err != nil {
  		return false, err
  	}

  	resp, err := oc.HTTPClient.Do(req)
  	if err != nil {
  		return false, err
  	}
  	defer resp.Body.Close()

  	if resp.StatusCode != http.StatusOK {
  		return false, fmt.Errorf("queryLabels return non-200: %s", resp.Status)
  	}

  	var res queryLabelsResponse
  	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
  		return false, err
  	}

  	for _, lbl := range res.Labels {
  		if lbl.Src == oc.LabelerDID && (lbl.Val == "meta_discourse" || lbl.Val == "possible_meta_discourse") {
  			return true, nil
  		}
  	}

  	return false, nil
  }

  // EmitLabel pushes an auto-moderation event adding the label to Ozone
  func (oc *OzoneClient) EmitLabel(ctx context.Context, result *types.ClassificationResult) error {
  	labelVal := "possible_meta_discourse"
  	if result.Probability >= 0.85 {
  		labelVal = "meta_discourse"
  	}

  	payload := map[string]interface{}{
  		"event": map[string]interface{}{
  			"$type":      "tools.ozone.moderation.defs#modEventLabel",
  			"createLabelVals": []string{labelVal},
  			"negateLabelVals": []string{},
  			"comment":         fmt.Sprintf("Auto-classified with probability %.2f", result.Probability),
  		},
  		"subject": map[string]interface{}{
  			"$type": "com.atproto.repo.strongRef",
  			"uri":   result.Post.TargetURI,
  			"cid":   "", // Omit for general StrongRef parsing in Ozone
  		},
  		"createdBy": oc.LabelerDID,
  	}

  	body, err := json.Marshal(payload)
  	if err != nil {
  		return err
  	}

  	endpoint := fmt.Sprintf("%s/xrpc/tools.ozone.moderation.emitEvent", oc.Endpoint)
  	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
  	if err != nil {
  		return err
  	}

  	req.Header.Set("Content-Type", "application/json")
  	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oc.AdminToken))

  	resp, err := oc.HTTPClient.Do(req)
  	if err != nil {
  		return err
  	}
  	defer resp.Body.Close()

  	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
  		return fmt.Errorf("emitEvent non-success: %s", resp.Status)
  	}

  	return nil
  }
  ```

- [ ] **Step 2: Write Ozone Service Tests**
  Create `internal/services/ozone_test.go` with mock verification handlers:
  ```go
  package services

  import (
  	"context"
  	"encoding/json"
  	"net/http"
  	"net/http/httptest"
  	"testing"

  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
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
  					"val": "meta_discourse",
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
  ```

- [ ] **Step 3: Run Ozone Client Tests**
  Run:
  ```bash
  go test -v ./internal/services/... -run TestOzone
  ```
  Expected: PASS

- [ ] **Step 4: Commit**
  Run:
  ```bash
  git add internal/services/ozone.go internal/services/ozone_test.go
  git commit -m "feat: implement decoupled Ozone API XRPC client and validation"
  ```

---

### Task 7: Atomic Cursor Persistence
* **Files:**
  * Create: `internal/pipeline/cursor.go`
  * Create: `internal/pipeline/cursor_test.go`

- [ ] **Step 1: Write Atomic Cursor Tracker**
  Create `internal/pipeline/cursor.go` containing:
  ```go
  package pipeline

  import (
  	"encoding/json"
  	"fmt"
  	"os"
  	"path/filepath"
  	"sync"
  )

  type CursorTracker struct {
  	filePath string
  	mu       sync.Mutex
  }

  type cursorPayload struct {
  	Cursor int64 `json:"cursor"`
  }

  func NewCursorTracker(filePath string) *CursorTracker {
  	return &CursorTracker{filePath: filePath}
  }

  func (ct *CursorTracker) Read() (int64, error) {
  	ct.mu.Lock()
  	defer ct.mu.Unlock()

  	file, err := os.Open(ct.filePath)
  	if err != nil {
  		if os.IsNotExist(err) {
  			return 0, nil
  		}
  		return 0, err
  	}
  	defer file.Close()

  	var p cursorPayload
  	if err := json.NewDecoder(file).Decode(&p); err != nil {
  		return 0, err
  	}
  	return p.Cursor, nil
  }

  func (ct *CursorTracker) Write(cursor int64) error {
  	ct.mu.Lock()
  	defer ct.mu.Unlock()

  	dir := filepath.Dir(ct.filePath)
  	if err := os.MkdirAll(dir, 0755); err != nil {
  		return err
  	}

  	tmpPath := ct.filePath + ".tmp"
  	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
  	if err != nil {
  		return err
  	}
  	defer tmpFile.Close()

  	p := cursorPayload{Cursor: cursor}
  	if err := json.NewEncoder(tmpFile).Encode(p); err != nil {
  		return err
  	}

  	if err := tmpFile.Sync(); err != nil {
  		return err
  	}
  	tmpFile.Close()

  	return os.Rename(tmpPath, ct.filePath)
  }
  ```

- [ ] **Step 2: Write Cursor Persistence Tests**
  Create `internal/pipeline/cursor_test.go` verifying write and reload consistency:
  ```go
  package pipeline

  import (
  	"os"
  	"testing"
  )

  func TestCursorTrackerAtomicReadWrite(t *testing.T) {
  	tmpFile := "./test_data/cursor_test.json"
  	defer os.RemoveAll("./test_data")

  	tracker := NewCursorTracker(tmpFile)
  	c, err := tracker.Read()
  	if err != nil {
  		t.Fatalf("unexpected read error: %v", err)
  	}
  	if c != 0 {
  		t.Errorf("expected 0, got %d", c)
  	}

  	err = tracker.Write(987654321)
  	if err != nil {
  		t.Fatalf("unexpected write error: %v", err)
  	}

  	c, err = tracker.Read()
  	if err != nil {
  		t.Fatalf("unexpected reread error: %v", err)
  	}
  	if c != 987654321 {
  		t.Errorf("expected 987654321, got %d", c)
  	}
  }
  ```

- [ ] **Step 3: Run Cursor Tests**
  Run:
  ```bash
  go test -v ./internal/pipeline/... -run TestCursorTracker
  ```
  Expected: PASS

- [ ] **Step 4: Commit**
  Run:
  ```bash
  git add internal/pipeline/cursor.go internal/pipeline/cursor_test.go
  git commit -m "feat: implement atomic cursor persistence and read/write cycle with tests"
  ```

---

### Task 8: Pipeline Coordinator (Concurrency, LRU, & Resilient Routing)
* **Files:**
  * Create: `internal/pipeline/coordinator.go`
  * Create: `internal/pipeline/coordinator_test.go`

- [ ] **Step 1: Write a simple LRU cache**
  To keep the project clean and dependencies minimal, add a simple thread-safe LRU cache directly into `internal/pipeline/coordinator.go` or split it.
  Create `internal/pipeline/coordinator.go` containing:
  ```go
  package pipeline

  import (
  	"context"
  	"log"
  	"sync"
  	"time"

  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
  )

  // Thread-Safe LRU Cache
  type LRUCache struct {
  	mu    sync.Mutex
  	items map[string]bool
  	keys  []string
  	size  int
  }

  func NewLRUCache(size int) *LRUCache {
  	return &LRUCache{
  		items: make(map[string]bool),
  		keys:  make([]string, 0, size),
  		size:  size,
  	}
  }

  func (c *LRUCache) ContainsOrAdd(key string) bool {
  	c.mu.Lock()
  	defer c.mu.Unlock()

  	if c.items[key] {
  		return true
  	}

  	if len(c.keys) >= c.size {
  		// Evict oldest
  		oldest := c.keys[0]
  		c.keys = c.keys[1:]
  		delete(c.items, oldest)
  	}

  	c.keys = append(c.keys, key)
  	c.items[key] = true
  	return false
  }

  type Coordinator struct {
  	Ingester              Ingester
  	Hydrator              Hydrator
  	Classifier            Classifier
  	OzoneClient           LabelEmitter
  	OzoneQuery            interface {
  		IsAlreadyLabeled(ctx context.Context, uri string) (bool, error)
  	}
  	Cursor                *CursorTracker
  	CursorRewindSeconds  int
  	HydrationWorkers      int
  	ClassificationWorkers int
  	DryRun                bool
  	LRU                   *LRUCache
  }

  func NewCoordinator(ingester Ingester, hydrator Hydrator, classifier Classifier, emitter LabelEmitter, oq interface {
  	IsAlreadyLabeled(ctx context.Context, uri string) (bool, error)
  }, cursor *CursorTracker, rewindSecs, hydWorkers, classWorkers int, dryRun bool) *Coordinator {
  	return &Coordinator{
  		Ingester:              ingester,
  		Hydrator:              hydrator,
  		Classifier:            classifier,
  		OzoneClient:           emitter,
  		OzoneQuery:            oq,
  		Cursor:                cursor,
  		CursorRewindSeconds:  rewindSecs,
  		HydrationWorkers:      hydWorkers,
  		ClassificationWorkers: classWorkers,
  		DryRun:                dryRun,
  		LRU:                   NewLRUCache(5000),
  	}
  }

  func (co *Coordinator) Run(ctx context.Context) error {
  	// Load cursor and apply rewind
  	startCursor, err := co.Cursor.Read()
  	if err != nil {
  		return err
  	}
  	if startCursor > 0 && co.CursorRewindSeconds > 0 {
  		rewind := int64(co.CursorRewindSeconds) * 1_000_000
  		if startCursor > rewind {
  			startCursor -= rewind
  			log.Printf("Recovery: Rewinding cursor by %d seconds to time_us: %d", co.CursorRewindSeconds, startCursor)
  		}
  	}

  	rawChan := make(chan *types.RawEvent, 1000)
  	hydratedChan := make(chan *types.HydratedPost, 100)

  	// 1. Ingestion Goroutine
  	go func() {
  		err := co.Ingester.Start(ctx, startCursor, rawChan)
  		if err != nil && ctx.Err() == nil {
  			log.Printf("Ingestion stopped with error: %v", err)
  		}
  	}()

  	// 2. Hydration Workers
  	var wg sync.WaitGroup
  	for i := 0; i < co.HydrationWorkers; i++ {
  		wg.Add(1)
  		go func() {
  			defer wg.Done()
  			for {
  				select {
  				case ev, ok := <-rawChan:
  					if !ok {
  						return
  					}
  					if ev.Type != "commit" || ev.Commit == nil || ev.Commit.Collection != "app.bsky.feed.post" {
  						continue
  					}

  					// Deduplication checks
  					postURI := "at://" + ev.Did + "/" + ev.Commit.Collection + "/" + ev.Commit.RKey
  					if co.LRU.ContainsOrAdd(postURI) {
  						continue
  					}

  					go co.hydrateWithRetry(ctx, ev, hydratedChan, 0)
  				case <-ctx.Done():
  					return
  				}
  			}
  		}()
  	}

  	// 3. Classification & Emission Workers
  	var commitWg sync.WaitGroup
  	var cursorMutex sync.Mutex
  	var processCount int

  	for i := 0; i < co.ClassificationWorkers; i++ {
  		wg.Add(1)
  		go func() {
  			defer wg.Done()
  			for {
  				select {
  				case hp, ok := <-hydratedChan:
  					if !ok {
  						return
  					}

  					res, err := co.Classifier.Classify(ctx, hp)
  					if err != nil {
  						log.Printf("Classification error for %s: %v", hp.TargetURI, err)
  						continue
  					}

  					if res.IsMetaDiscourse {
  						log.Printf("[MATCH] Meta-Discourse (Prob: %.2f) at URI: %s\nText: %s", res.Probability, hp.TargetURI, hp.TargetText)
  						if co.DryRun {
  							log.Printf("[DRY-RUN] Suppressed label emission to Ozone")
  						} else {
  							// Check query labels prior to emission
  							labeled, err := co.OzoneQuery.IsAlreadyLabeled(ctx, hp.TargetURI)
  							if err != nil {
  								log.Printf("Error checking labels in Ozone for %s: %v", hp.TargetURI, err)
  							}
  							if !labeled {
  								err = co.OzoneClient.EmitLabel(ctx, res)
  								if err != nil {
  									log.Printf("Failed to emit label to Ozone for %s: %v", hp.TargetURI, err)
  								} else {
  									log.Printf("[EMITTED] Label posted successfully")
  								}
  							} else {
  								log.Printf("[SKIPPED] Label already present in Ozone")
  							}
  						}
  					}

  					// Atomic Cursor Management
  					cursorMutex.Lock()
  					processCount++
  					shouldSync := res.IsMetaDiscourse || processCount%100 == 0
  					cursorMutex.Unlock()

  					if shouldSync {
  						_ = co.Cursor.Write(hp.EventTimeUS)
  					}
  				case <-ctx.Done():
  					return
  				}
  			}
  		}()
  	}

  	<-ctx.Done()
  	close(rawChan)
  	wg.Wait()
  	close(hydratedChan)
  	commitWg.Wait()

  	return nil
  }

  func (co *Coordinator) hydrateWithRetry(ctx context.Context, ev *types.RawEvent, out chan<- *types.HydratedPost, attempt int) {
  	hp, err := co.Hydrator.Hydrate(ctx, ev)
  	if err == nil {
  		select {
  		case out <- hp:
  		case <-ctx.Done():
  		}
  		return
  	}

  	if attempt >= 3 {
  		log.Printf("Hydration completely failed for event %d after 3 retries: %v", ev.TimeUS, err)
  		return
  	}

  	// Exponential backoff retry queueing
  	backoff := time.Duration(1<<(attempt+1)) * 250 * time.Millisecond
  	time.AfterFunc(backoff, func() {
  		co.hydrateWithRetry(ctx, ev, out, attempt+1)
  	})
  }
  ```

- [ ] **Step 2: Write Coordinator Tests**
  Create `internal/pipeline/coordinator_test.go` mocking all dependencies:
  ```go
  package pipeline

  import (
  	"context"
  	"os"
  	"testing"
  	"time"

  	"github.com/npmanos/discourse-labeler/internal/pipeline/types"
  )

  type mockIngester struct{}
  func (m *mockIngester) Start(ctx context.Context, cursor int64, out chan<- *types.RawEvent) error {
  	out <- &types.RawEvent{
  		Did:    "did:plc:user",
  		TimeUS: 500,
  		Type:   "commit",
  		Commit: &types.JetstreamCommit{
  			RKey:       "post1",
  			Collection: "app.bsky.feed.post",
  			Record:     []byte(`{"text": "discourse"}`),
  		},
  	}
  	<-ctx.Done()
  	return nil
  }

  type mockHydrator struct{}
  func (m *mockHydrator) Hydrate(ctx context.Context, post *types.RawEvent) (*types.HydratedPost, error) {
  	return &types.HydratedPost{
  		TargetDID:   post.Did,
  		TargetRKey:  post.Commit.RKey,
  		TargetURI:   "at://user/post1",
  		TargetText:  "discourse",
  		EventTimeUS: post.TimeUS,
  	}, nil
  }

  type mockClassifier struct{}
  func (m *mockClassifier) Classify(ctx context.Context, post *types.HydratedPost) (*types.ClassificationResult, error) {
  	return &types.ClassificationResult{
  		Post:            post,
  		IsMetaDiscourse: true,
  		Probability:     0.90,
  	}, nil
  }

  type mockEmitter struct {
  	called bool
  }
  func (m *mockEmitter) EmitLabel(ctx context.Context, result *types.ClassificationResult) error {
  	m.called = true
  	return nil
  }
  func (m *mockEmitter) IsAlreadyLabeled(ctx context.Context, uri string) (bool, error) {
  	return false, nil
  }

  func TestCoordinatorIntegrationRun(t *testing.T) {
  	tmpFile := "./test_data/cursor_coord.json"
  	defer os.RemoveAll("./test_data")

  	tracker := NewCursorTracker(tmpFile)
  	emitter := &mockEmitter{}

  	coordinator := NewCoordinator(
  		&mockIngester{},
  		&mockHydrator{},
  		&mockClassifier{},
  		emitter,
  		emitter,
  		tracker,
  		0, 2, 2, false,
  	)

  	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
  	defer cancel()

  	_ = coordinator.Run(ctx)

  	if !emitter.called {
  		t.Error("expected mockEmitter to have been called")
  	}

  	cur, err := tracker.Read()
  	if err != nil || cur != 500 {
  		t.Errorf("expected cursor to be saved as 500, got %d (err: %v)", cur, err)
  	}
  }
  ```

- [ ] **Step 3: Run Coordinator Integration Tests**
  Run:
  ```bash
  go test -v ./internal/pipeline/... -run TestCoordinatorIntegrationRun
  ```
  Expected: PASS

- [ ] **Step 4: Commit**
  Run:
  ```bash
  git add internal/pipeline/coordinator.go internal/pipeline/coordinator_test.go
  git commit -m "feat: implement high-performance concurrent pipeline coordinator and worker pools with tests"
  ```

---

### Task 9: Command Entry Point & Signal Handling
* **Files:**
  * Create: `cmd/labeler/main.go`

- [ ] **Step 1: Write the Core Entry Point**
  Create `cmd/labeler/main.go` containing:
  ```go
  package main

  import (
  	"context"
  	"log"
  	"os"
  	"os/signal"
  	"syscall"

  	"github.com/npmanos/discourse-labeler/internal/config"
  	"github.com/npmanos/discourse-labeler/internal/pipeline"
  	"github.com/npmanos/discourse-labeler/internal/services"
  )

  func main() {
  	log.Println("Starting Bluesky Meta-Discourse Labeler Daemon...")

  	cfg, err := config.Load()
  	if err != nil {
  		log.Fatalf("Fatal: failed to load configuration: %v", err)
  	}

  	if cfg.GrazeFeedURI == "" {
  		log.Fatal("Fatal: GRAZE_FEED_URI environment variable is required")
  	}

  	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
  	defer stop()

  	// 1. Initialize Attached Resources/Adapters
  	ingester := services.NewContrailsIngester(cfg.ContrailsWSURL, cfg.GrazeFeedURI)
  	hydrator := services.NewSlingshotHydrator(cfg.SlingshotURL)
  	classifier := services.NewLLMClassifier(cfg.LLMEndpoint, cfg.LLMModel)
  	ozoneClient := services.NewOzoneClient(cfg.OzoneEndpoint, cfg.OzoneAdminToken, cfg.LabelerDID)
  	cursor := pipeline.NewCursorTracker(cfg.CursorFilePath)

  	// 2. Build Pipeline Coordinator
  	coordinator := pipeline.NewCoordinator(
  		ingester,
  		hydrator,
  		classifier,
  		ozoneClient,
  		ozoneClient,
  		cursor,
  		cfg.CursorRewindSeconds,
  		cfg.HydrationWorkers,
  		cfg.ClassificationWorkers,
  		cfg.DryRun,
  	)

  	log.Println("Pipeline initialized. Spawning worker pools...")

  	// 3. Start Ingestion and Worker Loops
  	if err := coordinator.Run(ctx); err != nil && err != context.Canceled {
  		log.Fatalf("Fatal: Coordinator execution error: %v", err)
  	}

  	log.Println("Gracefully shut down all worker pools. Storing final states.")
  }
  ```

- [ ] **Step 2: Run Compile Verification on Complete Application**
  Run:
  ```bash
  go build -o labeler ./cmd/labeler/main.go
  ```
  Expected: Builds a single `labeler` binary with zero compile warnings.

- [ ] **Step 3: Run Full Suite of Unit Tests**
  Run:
  ```bash
  go test -v ./...
  ```
  Expected: PASS for all packages!

- [ ] **Step 4: Commit**
  Run:
  ```bash
  git add cmd/labeler/main.go
  git commit -m "feat: implement daemon entry point and OS signal handling"
  ```

---

### Task 10: Docker Setup
* **Files:**
  * Create: `Dockerfile`
  * Create: `docker-compose.yml`

- [ ] **Step 1: Write multi-stage Dockerfile**
  Create `Dockerfile` exactly as defined in the technical specification, referencing `static-debian13:nonroot`.

- [ ] **Step 2: Write docker-compose.yml**
  Create `docker-compose.yml` exactly as defined in the technical specification, including the commented-out GPU toggles and local volumes.

- [ ] **Step 3: Verify Docker Compile Build**
  Run:
  ```bash
  docker compose build
  ```
  Expected: Successfully builds the `labeler_daemon` image with a static Distroless target, logging zero build failures.

- [ ] **Step 4: Commit**
  Run:
  ```bash
  git add Dockerfile docker-compose.yml
  git commit -m "feat: add multi-stage Distroless Debian 13 Dockerfile and Compose configurations"
  ```
