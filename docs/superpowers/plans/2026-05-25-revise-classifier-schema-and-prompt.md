# Revise Classifier Schema and Prompt Override Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Revise the meta-discourse system prompt, format LLM inputs using dynamic XML `<posts>` tags for contextual posts, and support system prompt overrides via `LLM_SYSTEM_PROMPT` and `LLM_SYSTEM_PROMPT_PATH` environment variables.

**Architecture:** We will extend `config.Config` to load system prompt overrides from raw environment variables or paths, update `LLMClassifier` to support custom prompt injections using functional options, implement an XML-based dynamic input formatter helper, update few-shot conversation history, and verify using mock servers checking the request payload.

**Tech Stack:** Go (Golang), OpenAI Go SDK, Standard testing library (`testing`, `net/http/httptest`).

---

### Task 1: Add system prompt override support to Config

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

We will add a test `TestConfigLoadSystemPromptOverride` to `internal/config/config_test.go` that verifies:
1. When `LLM_SYSTEM_PROMPT` is set, it overrides the config's prompt.
2. When `LLM_SYSTEM_PROMPT_PATH` is set, it reads the file content.
3. When both are set, `LLM_SYSTEM_PROMPT` takes precedence.

Add the following code to `internal/config/config_test.go`:

```go
func TestConfigLoadSystemPromptOverride(t *testing.T) {
	// Scenario 1: Direct string override
	t.Setenv("LLM_SYSTEM_PROMPT", "direct-prompt-override")
	t.Setenv("LLM_SYSTEM_PROMPT_PATH", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.LLMSystemPrompt != "direct-prompt-override" {
		t.Errorf("expected LLMSystemPrompt 'direct-prompt-override', got %q", cfg.LLMSystemPrompt)
	}

	// Scenario 2: File path override
	t.Setenv("LLM_SYSTEM_PROMPT", "")
	tmpFile := t.TempDir() + "/prompt.txt"
	expectedPrompt := "file-prompt-override"
	if err := os.WriteFile(tmpFile, []byte(expectedPrompt), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	t.Setenv("LLM_SYSTEM_PROMPT_PATH", tmpFile)
	cfg, err = Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.LLMSystemPrompt != expectedPrompt {
		t.Errorf("expected LLMSystemPrompt %q, got %q", expectedPrompt, cfg.LLMSystemPrompt)
	}

	// Scenario 3: Both set (direct string takes precedence)
	t.Setenv("LLM_SYSTEM_PROMPT", "direct-precedence")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.LLMSystemPrompt != "direct-precedence" {
		t.Errorf("expected LLMSystemPrompt 'direct-precedence', got %q", cfg.LLMSystemPrompt)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test -v ./internal/config/... -run TestConfigLoadSystemPromptOverride
```
Expected: FAIL due to field not existing, or test failing to compile.

- [ ] **Step 3: Write minimal implementation**

Modify `internal/config/config.go`:
1. Add `LLMSystemPrompt` string field to `Config`:
```diff
 type Config struct {
 	Port                  string
 	LogLevel              string
 	CursorFilePath        string
 	CursorRewindSeconds   int
 	HydrationWorkers      int
 	ClassificationWorkers int
 	GrazeFeedURI          string
 	ContrailsWSURL        string
 	SlingshotURL          string
 	LLMEndpoint           string
 	LLMModel              string
 	LLMTemperature        float64
+	LLMSystemPrompt       string
 	OzoneEndpoint         string
 	LabelerDID            string
 	OzoneAdminToken       string
 	DryRun                bool
 }
```

2. Add a helper function `loadSystemPrompt` at the bottom of the file:
```go
func loadSystemPrompt(promptEnv, promptPathEnv string) string {
	if promptEnv != "" {
		return promptEnv
	}
	if promptPathEnv != "" {
		content, err := os.ReadFile(promptPathEnv)
		if err == nil {
			return string(content)
		}
	}
	return ""
}
```

3. Initialize `LLMSystemPrompt` inside `Load()`:
```diff
 	return &Config{
 		Port:                  getEnv("PORT", "8081"),
 		LogLevel:              getEnv("LOG_LEVEL", "info"),
 		CursorFilePath:        getEnv("CURSOR_FILE_PATH", "./data/cursor.json"),
 		CursorRewindSeconds:   getEnvInt("CURSOR_REWIND_SECONDS", 10),
 		HydrationWorkers:      getEnvInt("HYDRATION_WORKERS", 10),
 		ClassificationWorkers: getEnvInt("CLASSIFICATION_WORKERS", 4),
 		GrazeFeedURI:          getEnv("GRAZE_FEED_URI", ""),
 		ContrailsWSURL:        getEnv("CONTRAILS_WS_URL", "wss://api.graze.social/app/contrail"),
 		SlingshotURL:          getEnv("SLINGSHOT_URL", "https://slingshot.microcosm.blue"),
 		LLMEndpoint:           getEnv("LLM_ENDPOINT", "http://localhost:8080/v1/"),
 		LLMModel:              getEnv("LLM_MODEL", "google/gemma-4-e2b-gguf"),
 		LLMTemperature:        getEnvFloat("LLM_TEMPERATURE", 0.0),
+		LLMSystemPrompt:       loadSystemPrompt(getEnv("LLM_SYSTEM_PROMPT", ""), getEnv("LLM_SYSTEM_PROMPT_PATH", "")),
 		OzoneEndpoint:         getEnv("OZONE_ENDPOINT", "http://localhost:3000"),
 		LabelerDID:            getEnv("LABELER_DID", ""),
 		OzoneAdminToken:       getEnv("OZONE_ADMIN_TOKEN", ""),
 		DryRun:                getEnvBool("DRY_RUN", false),
 	}, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v ./internal/config/...
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add system prompt override support to Config"
```

---

### Task 2: Implement XML Post Formatter and Prompt/Override Logic in LLMClassifier

**Files:**
- Modify: `internal/services/classifier.go`
- Test: `internal/services/classifier_test.go`

- [ ] **Step 1: Write the failing test**

We will modify `internal/services/classifier_test.go` by:
1. Updating imports if needed (`os` or others).
2. Adding a test `TestLLMClassifierWithSystemPromptOverride` to verify that when `WithSystemPrompt` is passed, the classifier forwards the custom system prompt instead of the default.

Add to `internal/services/classifier_test.go`:

```go
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
						"content": `{"is_meta_discourse": false}`,
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test -v ./internal/services/... -run TestLLMClassifierWithSystemPromptOverride
```
Expected: FAIL due to compilation error (too many arguments to `NewLLMClassifier` or `WithSystemPrompt` not defined).

- [ ] **Step 3: Write minimal implementation**

Modify `internal/services/classifier.go`:
1. Revise the hardcoded `sysPrompt` block:
```go
const sysPrompt = `You are a classification engine powering a network labeler. Your task is to analyze a social media post and determine if it qualifies as "Bluesky Meta-Discourse."

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
Analyze the provided user post. You MUST consider the target post in the context of any provided parent posts or quoted posts. A target post which replies to or quotes a post which is meta discourse is likely also meta discourse. Output a valid JSON object containing exactly one boolean key: `is_meta_discourse`.`
```

2. Modify `LLMClassifier` struct to hold `SystemPrompt`:
```diff
 type LLMClassifier struct {
 	Client       *openai.Client
 	Model        string
+	SystemPrompt string
 }
```

3. Implement `LLMClassifierOption` and `WithSystemPrompt` option:
```go
type LLMClassifierOption func(*LLMClassifier)

func WithSystemPrompt(prompt string) LLMClassifierOption {
	return func(lc *LLMClassifier) {
		lc.SystemPrompt = prompt
	}
}
```

4. Modify `NewLLMClassifier` to accept variadic options:
```go
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
```

5. Add `formatPostInput` helper function in `internal/services/classifier.go`:
```go
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
```

6. In `Classify` method:
- Choose prompt:
```go
	prompt := sysPrompt
	if lc.SystemPrompt != "" {
		prompt = lc.SystemPrompt
	}
```
- Format post:
```go
	targetPost := formatPostInput(hp)
```
- Update messages with XML-structured few-shots:
```go
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
		openai.AssistantMessage(`{"is_meta_discourse": true}`),
		// Example 2
		openai.UserMessage(`<posts>
  <post type="target_post">
    i'm not on bluesky because i want to live in a bubble. i'm on bluesky because i love reading long manifestos about what's wrong with bluesky by people who don't spend enough time here to know someone does this every 10 days.
  </post>
</posts>`),
		openai.AssistantMessage(`{"is_meta_discourse": true}`),
		// Example 3
		openai.UserMessage(`<posts>
  <post type="target_post">
    The Bluesky team just pushed an update for the new video player. You can now scrub through clips without the audio dropping out. Huge improvement over the beta version from last week.
  </post>
</posts>`),
		openai.AssistantMessage(`{"is_meta_discourse": false}`),
		// Example 4
		openai.UserMessage(`<posts>
  <post type="target_post">
    Every time I post about this election, my replies fill up with the worst takes imaginable. I can't believe people are actually defending this policy.
  </post>
</posts>`),
		openai.AssistantMessage(`{"is_meta_discourse": false}`),
		// Example 5
		openai.UserMessage(`<posts>
  <post type="target_post">
    The problem is that too many of you have lost faith in liberalism and the power of free speech and talking to the other side. Aaron Sorkin taught us that if you post hard enough you can actually force Elon Musk to change the way Twitter works.
  </post>
  <post type="quoted_post">
    i think, end of the day, the real problem with Bluesky is that most of its users are here because they want to be in a bubble. and they can tell themselves that it’s just about not being around Nazis it supporting Musk, and that’s part of it, but also there’s a palpable desire to be ensconced.
  </post>
</posts>`),
		openai.AssistantMessage(`{"is_meta_discourse": true}`),
		// Target Post
		openai.UserMessage(targetPost),
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v ./internal/services/...
```
Expected: PASS (with TestLLMClassifierWithSystemPromptOverride passing, and other existing tests adapted/passing).

- [ ] **Step 5: Commit**

```bash
git add internal/services/classifier.go internal/services/classifier_test.go
git commit -m "feat: implement XML post formatter and system prompt overrides in LLMClassifier"
```

---

### Task 3: Add dedicated test case with parent_post and quoted_post

**Files:**
- Modify: `internal/services/classifier_test.go`

- [ ] **Step 1: Write the failing test**

We will add a test `TestLLMClassifierParentAndQuoted` that verifies:
1. A post with both `ParentText` and `QuotedText` is formatted perfectly into the correct XML structure containing all three `<post type="...">` tags.
2. The exact formatted XML payload is transmitted to the OpenAI API mock server.

Add the following code to `internal/services/classifier_test.go`:

```go
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
						"content": `{"is_meta_discourse": true}`,
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

	if !res.IsMetaDiscourse {
		t.Errorf("expected classification to be true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
go test -v ./internal/services/... -run TestLLMClassifierParentAndQuoted
```
*Note: If Task 2 was already completed, this step will actually compile but it's important to run it to ensure exact output format alignment is verified.*
Expected: FAIL (if formatter logic is not fully aligned) or PASS (if fully correct).

- [ ] **Step 3: Write minimal implementation**

Verify formatting match or update `formatPostInput` helper inside `internal/services/classifier.go` to guarantee exact match.

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test -v ./internal/services/... -run TestLLMClassifierParentAndQuoted
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/services/classifier_test.go
git commit -m "test: add dedicated test validating XML formatting for parent and quoted posts"
```

---

### Task 4: Integrate System Prompt Override in Main Daemon

**Files:**
- Modify: `cmd/labeler/main.go`

- [ ] **Step 1: Write the failing test**

We don't have a direct unit test file for `main.go`, so we will perform a compilation dry-run to verify our changes don't cause compilation issues.
Run:
```bash
go build -o /dev/null ./cmd/labeler
```
Expected: FAIL (if NewLLMClassifier signatures are mismatching, but since we used Go variadic options, it might pass. We still verify correct config parameter passing).

- [ ] **Step 2: Write implementation**

Modify `cmd/labeler/main.go` to pass `services.WithSystemPrompt(cfg.LLMSystemPrompt)` option:
```diff
 	// 1. Initialize Attached Resources/Adapters
 	ingester := services.NewContrailsIngester(cfg.ContrailsWSURL, cfg.GrazeFeedURI)
 	hydrator := services.NewSlingshotHydrator(cfg.SlingshotURL)
-	classifier := services.NewLLMClassifier(cfg.LLMEndpoint, cfg.LLMModel)
+	classifier := services.NewLLMClassifier(cfg.LLMEndpoint, cfg.LLMModel, services.WithSystemPrompt(cfg.LLMSystemPrompt))
 	ozoneClient := services.NewOzoneClient(cfg.OzoneEndpoint, cfg.OzoneAdminToken, cfg.LabelerDID)
 	cursor := types.NewCursorTracker(cfg.CursorFilePath)
```

- [ ] **Step 3: Run verify build compiles**

Run:
```bash
go build -v -o /dev/null ./cmd/labeler
```
Expected: SUCCESS

- [ ] **Step 4: Run all unit tests to ensure absolute stability**

Run:
```bash
go test -v ./...
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/labeler/main.go
git commit -m "feat: pass LLM system prompt override option in labeler daemon entrypoint"
```
