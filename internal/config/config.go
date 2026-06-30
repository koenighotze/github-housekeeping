package config

import (
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

type GitHub struct {
	TokenRef string `yaml:"token_ref"`
}

type Repository struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
}

type MergePolicy struct {
	Allow []string `yaml:"allow"`
}

type CIPollPolicy struct {
	Timeout  time.Duration `yaml:"timeout"`
	Interval time.Duration `yaml:"interval"`
}

type Policy struct {
	Merge  MergePolicy  `yaml:"merge"`
	CIPoll CIPollPolicy `yaml:"ci_poll"`
}

type Config struct {
	GitHub       GitHub       `yaml:"github"`
	Repositories []Repository `yaml:"repositories"`
	Policy       Policy       `yaml:"policy"`
}

var validAllowValues = map[string]bool{"patch": true, "minor": true}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.GitHub.TokenRef == "" {
		return nil, fmt.Errorf("config: github.token_ref is required")
	}

	if len(cfg.Repositories) == 0 {
		return nil, fmt.Errorf("config: repositories list is empty")
	}

	for _, v := range cfg.Policy.Merge.Allow {
		if !validAllowValues[v] {
			return nil, fmt.Errorf("config: invalid merge allow value %q — only patch and minor are permitted (major is always held)", v)
		}
	}

	return &cfg, nil
}
