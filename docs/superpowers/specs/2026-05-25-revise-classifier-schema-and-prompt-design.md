# Spec: Revise Classifier Schema and Support System Prompt Override

## Goal
The goal of this task is to:
1. Revise the default classification system prompt used for Bluesky Meta-Discourse detection to match the updated instructions.
2. Structure the input to the classifier using an XML-like `<posts>` block containing `<post type="parent_post|quoted_post|target_post">` tags to properly handle context (replies and quotes).
3. Preload updated, XML-structured few-shot examples as conversation history.
4. Support overriding the system prompt at run time via two environment variables:
   - `LLM_SYSTEM_PROMPT` (raw string value)
   - `LLM_SYSTEM_PROMPT_PATH` (file path containing the prompt)
5. Ensure a robust test covers target posts that have both a parent post and a quoted post.

---

## Proposed Changes

### 1. Configuration System (`internal/config/config.go`)
We will add environment variable loading to parse direct prompt overrides and read prompt files during initialization.

- Update `Config` struct with a `LLMSystemPrompt` string field.
- In `Load()`, resolve the prompt in this order of precedence:
  1. If `LLM_SYSTEM_PROMPT` is set, use it.
  2. If `LLM_SYSTEM_PROMPT_PATH` is set, read the file and use its content.
  3. Otherwise, leave it empty (the classifier will fall back to the default hardcoded system prompt).

### 2. Input Schema & Functional Options (`internal/services/classifier.go`)
We will:
- Update the default hardcoded system prompt `sysPrompt` to the revised instructions.
- Add a helper function `formatPostInput(hp *types.HydratedPost) string` that formats target, parent, and quoted posts as XML tags.
- Update the few-shot examples in `Classify` to use the XML-formatted payloads.
- Introduce `LLMClassifierOption` and `WithSystemPrompt` functional options.
- Update the `NewLLMClassifier` constructor to accept optional `LLMClassifierOption` arguments.

### 3. Daemon Entrypoint (`cmd/labeler/main.go`)
- Update the instantiation of the classifier to pass the system prompt override option:
  ```go
  classifier := services.NewLLMClassifier(cfg.LLMEndpoint, cfg.LLMModel, services.WithSystemPrompt(cfg.LLMSystemPrompt))
  ```

### 4. Tests (`internal/services/classifier_test.go`)
- Adapt existing tests to the new XML format.
- Add a dedicated test `TestLLMClassifierParentAndQuoted` that validates correct formatting and classification when a post has both `ParentText` and `QuotedText`.

---

## Verification Plan

### Automated Tests
Run unit tests for the classification service:
```bash
go test -v ./internal/services/...
```

Run all unit tests in the project to verify no regressions:
```bash
go test -v ./...
```
