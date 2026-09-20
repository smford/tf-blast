package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents user-defined policies and thresholds.
type Config struct {
	Rules           RulesConfig `yaml:"rules"`
	FailOn          string      `yaml:"fail_on"`
	MaxBlast        int         `yaml:"max_blast"`
	IgnoreResources []string    `yaml:"ignore_resources"`
}

// RulesConfig defines pattern matchers for severity tiers.
type RulesConfig struct {
	Critical []string `yaml:"critical"`
	High     []string `yaml:"high"`
	Medium   []string `yaml:"medium"`
	Low      []string `yaml:"low"`
}

// DefaultConfig returns the default heuristic rules and settings.
func DefaultConfig() *Config {
	return &Config{
		Rules: RulesConfig{
			Critical: []string{
				"*rds*",
				"*database*",
				"*dynamodb*",
				"*s3_bucket*",
				"*persistent_volume*",
				"*blob*",
				"*spanner*",
				"*bigtable*",
				"*documentdb*",
			},
			High: []string{
				"*security_group*",
				"*route_table*",
				"*iam_role*",
				"*firewall*",
				"*policy*",
				"*acl*",
				"*nat_gateway*",
				"*internet_gateway*",
			},
			Medium: []string{
				"*instance*",
				"*ecs_service*",
				"*alb*",
				"*elb*",
				"*kubernetes_deployment*",
				"*stateful_set*",
				"*app_service*",
				"*container_group*",
			},
			Low: []string{
				"*tag*",
				"*log_group*",
				"*metric_alarm*",
				"*alert*",
				"*sns_topic*",
				"*route53_record*",
			},
		},
		FailOn:          "",
		MaxBlast:        0,
		IgnoreResources: []string{},
	}
}

// LoadConfig loads a config from the specified path, or attempts to find .tf-blast.yaml.
// If neither is found or path is empty, DefaultConfig is returned.
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		if _, err := os.Stat(".tf-blast.yaml"); err == nil {
			path = ".tf-blast.yaml"
		} else if _, err := os.Stat(".tf-blast.yml"); err == nil {
			path = ".tf-blast.yml"
		}
	}

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var userCfg Config
	if err := yaml.Unmarshal(data, &userCfg); err != nil {
		return nil, err
	}

	// Merge user rules if provided
	if len(userCfg.Rules.Critical) > 0 {
		cfg.Rules.Critical = userCfg.Rules.Critical
	}
	if len(userCfg.Rules.High) > 0 {
		cfg.Rules.High = userCfg.Rules.High
	}
	if len(userCfg.Rules.Medium) > 0 {
		cfg.Rules.Medium = userCfg.Rules.Medium
	}
	if len(userCfg.Rules.Low) > 0 {
		cfg.Rules.Low = userCfg.Rules.Low
	}
	if userCfg.FailOn != "" {
		cfg.FailOn = userCfg.FailOn
	}
	if userCfg.MaxBlast > 0 {
		cfg.MaxBlast = userCfg.MaxBlast
	}
	if len(userCfg.IgnoreResources) > 0 {
		cfg.IgnoreResources = userCfg.IgnoreResources
	}

	return cfg, nil
}
