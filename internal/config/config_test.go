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
