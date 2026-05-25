package config

import (
	"os"
	"testing"
)

func TestConfigLoadDefaults(t *testing.T) {
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
	t.Setenv("PORT", "9090")
	t.Setenv("DRY_RUN", "true")
	t.Setenv("CURSOR_REWIND_SECONDS", "30")

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

func TestConfigLoadInvalidTypesFallback(t *testing.T) {
	t.Setenv("CURSOR_REWIND_SECONDS", "invalid_number")
	t.Setenv("LLM_TEMPERATURE", "not_a_float")
	t.Setenv("DRY_RUN", "non_bool_value")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if cfg.CursorRewindSeconds != 10 {
		t.Errorf("expected fallback CursorRewindSeconds 10, got %d", cfg.CursorRewindSeconds)
	}
	if cfg.LLMTemperature != 0.0 {
		t.Errorf("expected fallback LLMTemperature 0.0, got %f", cfg.LLMTemperature)
	}
	if cfg.DryRun != false {
		t.Errorf("expected fallback DryRun false, got %t", cfg.DryRun)
	}
}

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
