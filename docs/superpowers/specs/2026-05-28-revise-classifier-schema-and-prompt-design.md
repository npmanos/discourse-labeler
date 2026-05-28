# Spec: Revised Classifier Schema and Prompt Design

- **Status**: Approved
- **Author**: Antigravity (AI Coding Assistant)
- **Date**: 2026-05-28

---

## 1. Objective

To update the **Bluesky Meta-Discourse Labeler** to support a revised classification prompt and a highly structured JSON output schema. The new system transitions from a simple binary classification (`is_meta_discourse` boolean) to a categorical taxonomy with context reasoning and support for human escalation of ambiguous cases.

---

## 2. Requirements

1. **Prompt Revision**: Update the core classification system prompt to include revised definitions of Meta-Discourse, Not Meta-Discourse, context instructions, and output schema rules.
2. **Schema Realignment**: Replace the boolean response structure with a rich JSON schema:
   - `context_analysis`: Mandatory analysis of parent (`parent_post`) and quoted (`quote_post`) posts. These are nullable fields representing the reasoning and classification for context.
   - `target_post`: Primary post analysis, containing `reasoning` (brief text) and `classification` (enum of `definite_meta`, `likely_meta`, `not_meta`, `unsure`).
3. **Categorical Pipeline Routing**:
   - `definite_meta` $\rightarrow$ Emit `"meta-discourse"` label to Ozone.
   - `likely_meta` $\rightarrow$ Emit `"possible-meta-discourse"` label to Ozone.
   - `unsure` $\rightarrow$ Escalate the post to Ozone for human review using `tools.ozone.moderation.defs#modEventEscalate`.
   - `not_meta` $\rightarrow$ No label / no action.
4. **Logprob Probability Extraction**: Update logprob parsing to scan specifically for the new categorical enum tokens to compute model confidence.
5. **Rich Comment Formatting**: Set the `comment` payload in both label and escalation Ozone API calls to present a beautifully structured, highly readable view of the LLM's classification and context reasoning.
6. **Improved Debug Logging**: Replace the post URI in the coordinator logging with the actual target post text formatted as a quoted Go string (`%q`) for rapid evaluation of classification results.

---

## 3. Architecture & Data Structures

### 3.1 Pipeline Types (`internal/pipeline/types.go`)

We will define structured types for the context analysis and target post details directly in the central pipeline types to avoid package duplication or circular dependencies:

```go
package types

type PostClassification struct {
	Reasoning      string `json:"reasoning"`
	Classification string `json:"classification"`
}

type ContextAnalysis struct {
	ParentPost *PostClassification `json:"parent_post"`
	QuotePost  *PostClassification `json:"quote_post"`
}

type ClassificationResult struct {
	Post            *HydratedPost
	Probability     float64

	// Structured LLM metadata
	ContextAnalysis ContextAnalysis
	TargetPost      PostClassification
}
```

### 3.2 Label Emitter Interface (`internal/pipeline/coordinator.go`)

We will expand the `LabelEmitter` interface to allow pushing moderator escalations:

```go
type LabelEmitter interface {
	EmitLabel(ctx context.Context, result *ClassificationResult) error
	IsAlreadyLabeled(ctx context.Context, uri string) (bool, error)
	EmitEscalation(ctx context.Context, result *ClassificationResult) error
}
```

---

## 4. Implementation Plan Detail

### 4.1 System Prompt & Few-Shots (`internal/services/classifier.go`)

- Implement `sysPrompt` as a multiline string matching the revised content moderation prompt.
- Define `SchemaResponse` to decode the LLM json response:
  ```go
  type SchemaResponse struct {
  	ContextAnalysis types.ContextAnalysis    `json:"context_analysis"`
  	TargetPost      types.PostClassification `json:"target_post"`
  }
  ```
- Build the strict map-based JSON schema representation in `openai.ResponseFormatJSONSchemaJSONSchemaParam` for strict schema enforcement.
- Update the 6 few-shot user and assistant messages in `Classify` to use the revised assistant responses.
- Update the probability calculator to scan for enum values `definite_meta`, `likely_meta`, `not_meta`, `unsure`.

### 4.2 Ozone Service Integration (`internal/services/ozone.go`)

- Update `EmitLabel` to format comments showing the target, parent, and quoted reasoning:
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
  ```
- Implement `EmitEscalation(ctx context.Context, result *types.ClassificationResult) error` using `tools.ozone.moderation.defs#modEventEscalate` and the rich formatted comment.

### 4.3 Coordinator Pipeline Orchestration (`internal/pipeline/coordinator.go`)

- Update `processClassification` to decode `res.TargetPost.Classification`.
- Route to `EmitLabel` (`definite_meta`, `likely_meta`), `EmitEscalation` (`unsure`), or skip (`not_meta`).
- Use `hp.TargetText` in the log statements, quoted via `%q`.

---

## 5. Verification Plan

### 5.1 Unit and Integration Testing
- Rewrite `internal/services/classifier_test.go` to assert:
  - Rich JSON structures are parsed successfully.
  - Logprobs are resolved correctly from categorical enums.
  - Parents and quotes are represented.
- Update `internal/pipeline/coordinator_test.go` to adapt to the new `ClassificationResult` structures.

### 5.2 Compile & Lint
- Execute `go test -v ./...` to verify all tests pass.
- Format code and organize imports:
  ```bash
  go fmt ./...
  go run golang.org/x/tools/cmd/goimports@latest -w .
  ```
- Run harness check:
  ```bash
  make verify-harness
  ```
