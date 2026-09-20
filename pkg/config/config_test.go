package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.Rules.Critical) == 0 {
		t.Errorf("expected default critical rules")
	}
	if len(cfg.Rules.High) == 0 {
		t.Errorf("expected default high rules")
	}
}

func TestLoadConfig_CustomYaml(t *testing.T) {
	yamlContent := `
rules:
  critical:
    - "*custom_db*"
fail_on: "critical"
max_blast: 10
ignore_resources:
  - "*null_resource*"
`
	tmpfile, err := os.CreateTemp("", "tf-blast-cfg-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(yamlContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.FailOn != "critical" {
		t.Errorf("expected fail_on 'critical', got '%s'", cfg.FailOn)
	}
	if cfg.MaxBlast != 10 {
		t.Errorf("expected max_blast 10, got %d", cfg.MaxBlast)
	}
	if len(cfg.Rules.Critical) != 1 || cfg.Rules.Critical[0] != "*custom_db*" {
		t.Errorf("unexpected critical rules: %v", cfg.Rules.Critical)
	}
	if len(cfg.IgnoreResources) != 1 || cfg.IgnoreResources[0] != "*null_resource*" {
		t.Errorf("unexpected ignore_resources: %v", cfg.IgnoreResources)
	}
}
