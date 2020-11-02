package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	AccessToken  string             `yaml:"access_token"`
	URL          string             `yaml:"url"`
	Repositories []RepositoryConfig `yaml:"repositories"`
}

type RepositoryConfig struct {
	Project int          `yaml:"project"`
	Image   string       `yaml:"image"`
	Filter  FilterConfig `yaml:"filter"`
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
