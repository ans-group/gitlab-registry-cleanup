package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	AccessToken  string             `yaml:"access_token"`
	URL          string             `yaml:"url"`
	Policies     []PolicyConfig     `yaml:"policies"`
	Repositories []RepositoryConfig `yaml:"repositories"`
}

func (c *Config) GetPolicyConfig(name string) (PolicyConfig, error) {
	for _, cfg := range c.Policies {
		if cfg.Name == name {
			return cfg, nil
		}
	}

	return PolicyConfig{}, fmt.Errorf("Cannot find policy %s", name)
}

type PolicyConfig struct {
	Name   string       `yaml:"name"`
	Filter FilterConfig `yaml:"filter"`
}

type RepositoryConfig struct {
	Project  int      `yaml:"project"`
	Group    int      `yaml:"group"`
	Recurse  bool     `yaml:"recurse"`
	Images   []string `yaml:"images"`
	Policies []string `yaml:"policies"`
}

type FilterConfig struct {
	Include string `yaml:"include"`
	Exclude string `yaml:"exclude"`
	Keep    int    `yaml:"keep"`
	Age     int    `yaml:"age"`
}

func Parse(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config file: %w", err)
	}

	config := &Config{}

	err = yaml.Unmarshal(bytes, config)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal config file: %w", err)
	}

	return config, nil
}
