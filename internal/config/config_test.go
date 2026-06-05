package config

import (
	"os"
	"testing"
)

func clearEnv(t *testing.T) {
	vars := []string{
		"PORT", "LOG_LEVEL", "CURSOR_FILE_PATH", "CURSOR_REWIND_SECONDS",
		"HYDRATION_WORKERS", "CLASSIFICATION_WORKERS", "GRAZE_FEED_URI",
		"CONTRAILS_WS_URL", "SLINGSHOT_URL", "LLM_ENDPOINT", "LLM_MODEL", "LLM_API_KEY",
		"LLM_TEMPERATURE", "OZONE_ENDPOINT", "LABELER_DID", "OZONE_ADMIN_TOKEN",
		"DRY_RUN", "LLM_SYSTEM_PROMPT", "LLM_SYSTEM_PROMPT_PATH",
	}
	for _, v := range vars {
		if val, exists := os.LookupEnv(v); exists {
			os.Unsetenv(v)
			t.Cleanup(func() {
				os.Setenv(v, val)
			})
		}
	}
}

func TestConfigLoadDefaults(t *testing.T) {
	clearEnv(t)
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
	if cfg.LLMAPIKey != "" {
		t.Errorf("expected default LLMAPIKey to be empty, got %s", cfg.LLMAPIKey)
	}
}

func TestConfigLoadEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DRY_RUN", "true")
	t.Setenv("CURSOR_REWIND_SECONDS", "30")
	t.Setenv("LLM_API_KEY", "override-key")

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
	if cfg.LLMAPIKey != "override-key" {
		t.Errorf("expected LLMAPIKey override-key, got %s", cfg.LLMAPIKey)
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

func TestConfigLoadSystemPromptOverrideError(t *testing.T) {
	t.Setenv("LLM_SYSTEM_PROMPT", "")
	t.Setenv("LLM_SYSTEM_PROMPT_PATH", "/non/existent/path/to/prompt.txt")

	cfg, err := Load()
	if err == nil {
		t.Fatal("expected error when loading from non-existent prompt path, got nil")
	}
	if cfg != nil {
		t.Errorf("expected nil config on error, got %+v", cfg)
	}
}

func TestLoadEnvFile(t *testing.T) {
	clearEnv(t)

	// Create a temporary .env file
	tmpDir := t.TempDir()
	envPath := tmpDir + "/.env"
	content := `
# This is a comment
PORT=9091
LLM_MODEL="custom-model-from-env"
LLM_API_KEY='single-quoted-key'
OZONE_ADMIN_TOKEN=some-token
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp env file: %v", err)
	}

	if err := loadEnvFile(envPath); err != nil {
		t.Fatalf("failed to load env file: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Port != "9091" {
		t.Errorf("expected Port 9091, got %s", cfg.Port)
	}
	if cfg.LLMModel != "custom-model-from-env" {
		t.Errorf("expected LLMModel custom-model-from-env, got %s", cfg.LLMModel)
	}
	if cfg.LLMAPIKey != "single-quoted-key" {
		t.Errorf("expected LLMAPIKey single-quoted-key, got %s", cfg.LLMAPIKey)
	}
	if cfg.OzoneAdminToken != "some-token" {
		t.Errorf("expected OzoneAdminToken some-token, got %s", cfg.OzoneAdminToken)
	}
}
