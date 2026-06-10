package config

import (
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cfg := Config{
		IntervalMinutes: 30,
		Push: PushCfg{
			Enabled:    true,
			Categories: []string{"breadth", "depth"},
			Endpoint:   "https://example.com",
		},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.IntervalMinutes != 30 {
		t.Errorf("IntervalMinutes: got %d, want 30", got.IntervalMinutes)
	}
	if !got.Push.Enabled {
		t.Error("Push.Enabled: got false, want true")
	}
	if got.Push.Endpoint != "https://example.com" {
		t.Errorf("Push.Endpoint: got %q, want %q", got.Push.Endpoint, "https://example.com")
	}
	if len(got.Push.Categories) != 2 {
		t.Errorf("Push.Categories: got %v", got.Push.Categories)
	}
}

func TestIntervalDefault(t *testing.T) {
	var c Config
	if c.Interval() != 15*time.Minute {
		t.Errorf("Interval() default: got %v, want 15m", c.Interval())
	}
}

func TestIntervalCustom(t *testing.T) {
	c := Config{IntervalMinutes: 5}
	if c.Interval() != 5*time.Minute {
		t.Errorf("Interval() custom: got %v, want 5m", c.Interval())
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load of missing file: %v", err)
	}
	if cfg.IntervalMinutes != 0 {
		t.Errorf("expected zero IntervalMinutes, got %d", cfg.IntervalMinutes)
	}
}
