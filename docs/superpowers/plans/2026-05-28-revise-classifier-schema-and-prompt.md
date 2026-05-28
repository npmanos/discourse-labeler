# Revised Classifier Schema and Prompt Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the Bluesky Meta-Discourse Labeler daemon to apply a revised categorical classification system prompt and schema, routing definite/likely meta to public labels, and escalating unsure cases to Ozone.

**Architecture:** We will centralize parsed structs inside `internal/pipeline/types.go` for seamless integration between services and pipeline coordinator. We will expand the Ozone integration with `EmitEscalation` to support human moderation, and route cases by inspecting categorical classification enums.

**Tech Stack:** Go (Golang), AT Protocol / Ozone APIs.

---

### Task 1: Update Central Pipeline Types

**Files:**
- Modify: `internal/pipeline/types.go`

- [ ] **Step 1: Write minimal structures for categorical classification details**
  Add `PostClassification`, `ContextAnalysis` and update `ClassificationResult` in `internal/pipeline/types.go`.

  ```go
  // Add this inside internal/pipeline/types.go
  
  type PostClassification struct {
  	Reasoning      string `json:"reasoning"`
  	Classification string `json:"classification"`
  }
  
  type ContextAnalysis struct {
  	ParentPost *PostClassification `json:"parent_post"`
  	QuotePost  *PostClassification `json:"quote_post"`
  }
  
  // Modify ClassificationResult structure:
  type ClassificationResult struct {
  	Post            *HydratedPost
  	Probability     float64
  
  	ContextAnalysis ContextAnalysis
  	TargetPost      PostClassification
  }
  ```

- [ ] **Step 2: Commit**
  ```bash
  git add internal/pipeline/types.go
  git commit -m "feat: add PostClassification and ContextAnalysis structs to pipeline types"
  ```

---

### Task 2: Extend LabelEmitter Interface

**Files:**
- Modify: `internal/pipeline/coordinator.go`

- [ ] **Step 1: Expand interface to support EmitEscalation**
  Modify lines 25-30 in `internal/pipeline/coordinator.go` to declare `EmitEscalation`.

  ```go
  // LabelEmitter defines the interface for query and emission of labels.
  type LabelEmitter interface {
  	EmitLabel(ctx context.Context, result *ClassificationResult) error
  	IsAlreadyLabeled(ctx context.Context, uri string) (bool, error)
  	EmitEscalation(ctx context.Context, result *ClassificationResult) error
  }
  ```

- [ ] **Step 2: Commit**
  ```bash
  git add internal/pipeline/coordinator.go
  git commit -m "refactor: add EmitEscalation to LabelEmitter interface"
  ```

---

### Task 3: Implement Ozone Human Escalation and Rich Comments

**Files:**
- Modify: `internal/services/ozone.go`
- Modify: `internal/services/ozone_test.go`

- [ ] **Step 1: Write a failing test for Ozone human escalation**
  Add a test to verify `EmitEscalation` calls the tools.ozone endpoint with `modEventEscalate`.

  ```go
  // Add in internal/services/ozone_test.go:
  func TestOzoneEmitEscalationSuccess(t *testing.T) {
  	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  		if r.Method != "POST" {
  			t.Errorf("expected POST request, got %s", r.Method)
  		}
  		if r.URL.Path != "/xrpc/tools.ozone.moderation.emitEvent" {
  			t.Errorf("unexpected path: %s", r.URL.Path)
  		}
  		var payload map[string]interface{}
  		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
  			t.Fatalf("failed to decode: %v", err)
  		}
  		evt, ok := payload["event"].(map[string]interface{})
  		if !ok || evt["$type"] != "tools.ozone.moderation.defs#modEventEscalate" {
  			t.Errorf("expected modEventEscalate type, got: %v", evt["$type"])
  		}
  		if !strings.Contains(evt["comment"].(string), "Auto-escalated unsure post") {
  			t.Errorf("expected comment to contain prefix, got: %s", evt["comment"])
  		}
  		w.WriteHeader(http.StatusOK)
  	}))
  	defer server.Close()

  	oc := NewOzoneClient(server.URL, "secret-token", "did:web:test-labeler")
  	result := &types.ClassificationResult{
  		Post: &types.HydratedPost{
  			TargetURI: "at://did:web:user/app.bsky.feed.post/123",
  		},
  		Probability: 0.72,
  		TargetPost: types.PostClassification{
  			Classification: "unsure",
  			Reasoning:      "Unclear meta-discourse intent",
  		},
  	}

  	err := oc.EmitEscalation(context.Background(), result)
  	if err != nil {
  		t.Fatalf("unexpected error: %v", err)
  	}
  }
  ```

- [ ] **Step 2: Run test to verify it fails**
  Run: `go test -v ./internal/services/... -run TestOzoneEmitEscalationSuccess`
  Expected: FAIL (compilation failure / EmitEscalation not defined)

- [ ] **Step 3: Implement EmitEscalation and formatOzoneComment**
  Implement `EmitEscalation` and rewrite `EmitLabel` in `internal/services/ozone.go` to format human-readable comments.

  ```go
  func formatOzoneComment(result *types.ClassificationResult) string {
  	comment := fmt.Sprintf("Reasoning: %s", result.TargetPost.Reasoning)
  	if result.ContextAnalysis.ParentPost != nil {
  		comment += fmt.Sprintf("\n\nParent Post: [%s] %s", 
  			result.ContextAnalysis.ParentPost.Classification, 
  			result.ContextAnalysis.ParentPost.Reasoning)
  	}
  	if result.ContextAnalysis.QuotePost != nil {
  		comment += fmt.Sprintf("\n\nQuoted Post: [%s] %s", 
  			result.ContextAnalysis.QuotePost.Classification, 
  			result.ContextAnalysis.QuotePost.Reasoning)
  	}
  	return comment
  }

  func (oc *OzoneClient) EmitLabel(ctx context.Context, result *types.ClassificationResult) error {
  	labelVal := "possible-meta-discourse"
  	if result.TargetPost.Classification == "definite_meta" {
  		labelVal = "meta-discourse"
  	}

  	payload := map[string]interface{}{
  		"event": map[string]interface{}{
  			"$type":           "tools.ozone.moderation.defs#modEventLabel",
  			"createLabelVals": []string{labelVal},
  			"negateLabelVals": []string{},
  			"comment":         fmt.Sprintf("Auto-labeled %s (probability %.2f):\n%s", 
  				labelVal, result.Probability, formatOzoneComment(result)),
  		},
  		"subject": map[string]interface{}{
  			"$type": "com.atproto.repo.strongRef",
  			"uri":   result.Post.TargetURI,
  			"cid":   "",
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

  func (oc *OzoneClient) EmitEscalation(ctx context.Context, result *types.ClassificationResult) error {
  	payload := map[string]interface{}{
  		"event": map[string]interface{}{
  			"$type":   "tools.ozone.moderation.defs#modEventEscalate",
  			"comment": fmt.Sprintf("Auto-escalated unsure post (probability %.2f):\n%s", 
  				result.Probability, formatOzoneComment(result)),
  		},
  		"subject": map[string]interface{}{
  			"$type": "com.atproto.repo.strongRef",
  			"uri":   result.Post.TargetURI,
  			"cid":   "",
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
  		return fmt.Errorf("emitEvent non-success for escalation: %s", resp.Status)
  	}

  	return nil
  }
  ```

- [ ] **Step 4: Run test to verify it passes**
  Run: `go test -v ./internal/services/... -run TestOzoneEmitEscalationSuccess`
  Expected: PASS

- [ ] **Step 5: Commit**
  ```bash
  git add internal/services/ozone.go internal/services/ozone_test.go
  git commit -m "feat: implement EmitEscalation and rich Ozone comment formatting"
  ```

---

### Task 4: Revise LLM Classifier prompt, Schema and Few-Shots

**Files:**
- Modify: `internal/services/classifier.go`
- Modify: `internal/services/classifier_test.go`

- [ ] **Step 1: Write a failing test for rich classification parsing**
  Update unit tests in `internal/services/classifier_test.go` to assert the parsed `ContextAnalysis` and `TargetPost` structures.

  ```go
  // In internal/services/classifier_test.go, replace the TestLLMClassifierMetaDiscourseTrue mock content and checks:
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
  ```

- [ ] **Step 2: Run test to verify it fails**
  Run: `go test -v ./internal/services/... -run TestLLMClassifierMetaDiscourseTrue`
  Expected: FAIL (compilation errors / unmarshal failure due to outdated structures)

- [ ] **Step 3: Implement prompt, schema maps, few-shots, and enum parsing in `classifier.go`**
  Modify `internal/services/classifier.go` to match the approved design.

  ```go
  const sysPrompt = `You are a content moderator applying classification labels to user posts. Your task is to analyze a social media post and determine if it qualifies as "Bluesky Meta-Discourse."

# DEFINITION: META-DISCOURSE
Meta-discourse consists of posts evaluating, criticizing, or theorizing about the cultural and social experience of the platform itself. This includes:
*   Debating the "vibes," echo chambers, or user base behaviors of Bluesky.
*   Comparing the social experience, engagement dynamics, or toxicity of Bluesky versus X (Twitter) or other platforms.
*   Complaining about the types of conversations people have on the platform (e.g., "dead-end conversations", "too much drama", "people talking to themselves").
*   Meta-commentary, subtweets, or reactions to other users' posts regarding Bluesky's culture.

# DEFINITION: NOT META-DISCOURSE
The following are strictly NOT meta-discourse:
*   Technical discussions about building on the AT Protocol (atproto), creating custom feeds, using APIs, connecting to Jetstream, or hosting infrastructure.
*   Announcements or discussions of new Bluesky application features (e.g., "DMs are now live", "how to use the new video player").
*   General political, social, or pop culture arguments, even if they are toxic or reference platform moderation, as long as they are not explicitly analyzing the platform's culture as a whole.
*   Ordinary posts using platform-specific terminology (like "skeet" or "repost") in passing.

# INSTRUCTIONS
1. Analyze the provided target post.
2. You MUST consider the target post in the context of any provided parent or quoted posts. A target post which replies to or quotes meta-discourse is likely also meta-discourse.
3. Write a brief reasoning step (maximum 2 sentences) analyzing the post against the definitions above. 
4. Assign one of the following classification labels: "definite_meta", "likely_meta", "not_meta", "unsure".

# EXPECTED OUTPUT FORMAT
You must output a valid JSON object matching this schema. The context_analysis block is mandatory. If a parent post or quote post is not provided in the input, you MUST output null for that specific field.

{
  "context_analysis": {
    "parent_post": { "reasoning": "...", "classification": "..." },
    "quote_post": { "reasoning": "...", "classification": "..." }
  },
  "target_post": {
    "reasoning": "...",
    "classification": "..."
  }
}`

  type SchemaResponse struct {
  	ContextAnalysis types.ContextAnalysis    `json:"context_analysis"`
  	TargetPost      types.PostClassification `json:"target_post"`
  }
  ```

  And rewrite the schema parameters:
  ```go
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "DiscourseSchema",
		Description: openai.String("Identifies if a post contains meta-discourse with reasoning and context analysis"),
		Strict:      openai.Bool(true),
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"context_analysis": map[string]interface{}{
					"type":        "object",
					"description": "Mandatory analysis of parent or quoted posts. Null if not present.",
					"properties": map[string]interface{}{
						"parent_post": map[string]interface{}{
							"anyOf": []interface{}{
								map[string]interface{}{"type": "null"},
								map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"reasoning":      map[string]interface{}{"type": "string"},
										"classification": map[string]interface{}{
											"type": "string",
											"enum": []string{"definite_meta", "likely_meta", "not_meta", "unsure"},
										},
									},
									"required":             []string{"reasoning", "classification"},
									"additionalProperties": false,
								},
							},
						},
						"quote_post": map[string]interface{}{
							"anyOf": []interface{}{
								map[string]interface{}{"type": "null"},
								map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"reasoning":      map[string]interface{}{"type": "string"},
										"classification": map[string]interface{}{
											"type": "string",
											"enum": []string{"definite_meta", "likely_meta", "not_meta", "unsure"},
										},
									},
									"required":             []string{"reasoning", "classification"},
									"additionalProperties": false,
								},
							},
						},
					},
					"required":             []string{"parent_post", "quote_post"},
					"additionalProperties": false,
				},
				"target_post": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"reasoning":      map[string]interface{}{"type": "string"},
						"classification": map[string]interface{}{
							"type": "string",
							"enum": []string{"definite_meta", "likely_meta", "not_meta", "unsure"},
						},
					},
					"required":             []string{"reasoning", "classification"},
					"additionalProperties": false,
				},
			},
			"required":             []string{"context_analysis", "target_post"},
			"additionalProperties": false,
		},
	}
  ```

  And rewrite the few-shots array to match the revised assistant messages:
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
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": {
      "reasoning": "Explicitly evaluates Bluesky's user base, 'vibes', and compares the platform to X.",
      "classification": "definite_meta"
    },
    "quote_post": null
  },
  "target_post": {
    "reasoning": "Replies to meta-discourse and theorizes about Bluesky's specific engagement dynamics and gatekeeping.",
    "classification": "definite_meta"
  }
}`),
		// Example 2
		openai.UserMessage(`<posts>
  <post type="target_post">
    i'm not on bluesky because i want to live in a bubble. i'm on bluesky because i love reading long manifestos about what's wrong with bluesky by people who don't spend enough time here to know someone does this every 10 days.
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": null,
    "quote_post": null
  },
  "target_post": {
    "reasoning": "Provides meta-commentary and reaction to how other users critique Bluesky's culture.",
    "classification": "definite_meta"
  }
}`),
		// Example 3
		openai.UserMessage(`<posts>
  <post type="target_post">
    The Bluesky team just pushed an update for the new video player. You can now scrub through clips without the audio dropping out. Huge improvement over the beta version from last week.
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": null,
    "quote_post": null
  },
  "target_post": {
    "reasoning": "Discusses a new application feature, which is explicitly excluded from being meta-discourse.",
    "classification": "not_meta"
  }
}`),
		// Example 4
		openai.UserMessage(`<posts>
  <post type="target_post">
    Every time I post about this election, my replies fill up with the worst takes imaginable. I can't believe people are actually defending this policy.
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": null,
    "quote_post": null
  },
  "target_post": {
    "reasoning": "A general complaint about political arguments in replies, rather than an analysis of the platform's culture as a whole.",
    "classification": "not_meta"
  }
}`),
		// Example 5
		openai.UserMessage(`<posts>
  <post type="target_post">
    The problem is that too many of you have lost faith in liberalism and the power of free speech and talking to the other side. Aaron Sorkin taught us that if you post hard enough you can actually force Elon Musk to change the way Twitter works.
  </post>
  <post type="quoted_post">
    i think, end of the day, the real problem with Bluesky is that most of its users are here because they want to be in a bubble. and they can tell themselves that it’s just about not being around Nazis it supporting Musk, and that’s part of it, but also there’s a palpable desire to be ensconced.
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": null,
    "quote_post": {
      "reasoning": "Evaluates the desires and behaviors of the Bluesky user base.",
      "classification": "definite_meta"
    }
  },
  "target_post": {
    "reasoning": "Quotes meta-discourse and adds commentary comparing user approaches to speech on Twitter versus Bluesky.",
    "classification": "definite_meta"
  }
}`),
		// Example 6
		openai.UserMessage(`<posts>
  <post type="parent_post">
    I have the right one 🤓
https://bsky.app/profile/generalmusician.bsky.social/post/3lsrtbmb5q22k
  </post>
  <post type="target_post">
    I think this is how we became Bluesky friends 😅
  </post>
</posts>`),
		openai.AssistantMessage(`{
  "context_analysis": {
    "parent_post": {
      "reasoning": "Ordinary conversational post sharing a link with no analysis of platform culture.",
      "classification": "not_meta"
    },
    "quote_post": null
  },
  "target_post": {
    "reasoning": "Uses platform-specific terminology ('Bluesky friends') in passing without evaluating the social experience.",
    "classification": "not_meta"
  }
}`),
		// Target Post
		openai.UserMessage(targetPost),
	}
  ```

  And rewrite the Classify return builder:
  ```go
	var schemaResp SchemaResponse
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &schemaResp); err != nil {
		return nil, fmt.Errorf("failed to parse schema response content: %w", err)
	}

	result := &types.ClassificationResult{
		Post:            hp,
		Probability:     1.0, // Default to 100% confidence if logprobs are absent
		ContextAnalysis: schemaResp.ContextAnalysis,
		TargetPost:      schemaResp.TargetPost,
	}

	// Calculate probability from logprobs by finding the target classification enum
	if len(resp.Choices[0].Logprobs.Content) > 0 {
		found := false
		for _, tc := range resp.Choices[0].Logprobs.Content {
			trimmed := strings.ToLower(strings.Trim(tc.Token, " \t\n\r\"'{}[]:,"))
			if trimmed == "definite_meta" || trimmed == "likely_meta" || trimmed == "not_meta" || trimmed == "unsure" {
				result.Probability = math.Exp(tc.Logprob)
				found = true
				break
			}
		}
		if !found {
			// Fall back to the first token if not found
			result.Probability = math.Exp(resp.Choices[0].Logprobs.Content[0].Logprob)
		}
	}

	return result, nil
  ```

- [ ] **Step 4: Adapt remaining classifier_test.go unit tests and run**
  Adapt all mock JSON models in `classifier_test.go` to match the new structure, and run:
  `go test -v ./internal/services/...`
  Expected: PASS

- [ ] **Step 5: Commit**
  ```bash
  git add internal/services/classifier.go internal/services/classifier_test.go
  git commit -m "feat: implement revised system prompt, schema maps, logprob calculations, and updated few-shots"
  ```

---

### Task 5: Implement Coordinator Orchestration and Debug Logging

**Files:**
- Modify: `internal/pipeline/coordinator.go`
- Modify: `internal/pipeline/coordinator_test.go`

- [ ] **Step 1: Write failing tests in coordinator_test.go**
  Adapt the mock structures in `coordinator_test.go` to utilize `TargetPost` and categorical enums, verifying that both label emission and escalation triggers work as expected.

- [ ] **Step 2: Run test to verify it fails**
  Run: `go test -v ./internal/pipeline/...`
  Expected: FAIL

- [ ] **Step 3: Implement processClassification routing in `coordinator.go`**
  Implement the categorical routing logic and replace `hp.TargetURI` with `%q` formatted `hp.TargetText` in the coordinator debug logs.

- [ ] **Step 4: Run test to verify it passes**
  Run: `go test -v ./internal/pipeline/...`
  Expected: PASS

- [ ] **Step 5: Commit**
  ```bash
  git add internal/pipeline/coordinator.go internal/pipeline/coordinator_test.go
  git commit -m "feat: implement categorical routing and rich text debugging in coordinator"
  ```

---

### Task 6: Final Formatting and Harness Verification

**Files:**
- Modify: None (run verification tools)

- [ ] **Step 1: Standardize formatting**
  Run:
  ```bash
  go fmt ./...
  go run golang.org/x/tools/cmd/goimports@latest -w .
  ```

- [ ] **Step 2: Run all tests**
  Run: `go test -v ./...`
  Expected: PASS (All tests across the entire repository)

- [ ] **Step 3: Harness validation**
  Run: `make verify-harness`
  Expected: PASS

- [ ] **Step 4: Push feature branch**
  Push all changes to origin:
  ```bash
  git push -u origin feature/revised-prompt-schema
  ```
