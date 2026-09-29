package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	os.Clearenv()
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxConcurrentJobs != 1 {
		t.Fatalf("expected max jobs 1, got %d", cfg.MaxConcurrentJobs)
	}
	if len(cfg.AllowedMediaRoots) != 1 {
		t.Fatalf("expected one media root, got %v", cfg.AllowedMediaRoots)
	}
}
