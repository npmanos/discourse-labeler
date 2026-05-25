package config

import (
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
