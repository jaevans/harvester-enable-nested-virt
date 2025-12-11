package config

import (
	"os"
	"regexp"

	"log/slog"

	"gopkg.in/yaml.v3"
)

// NamespaceRule represents a namespace with its associated VM name patterns
type NamespaceRule struct {
	Namespace string
	Patterns  []*regexp.Regexp
}

type NamespaceRuleConfig struct {
	Namespace string
	Patterns  []string
}

// Config holds the configuration for the webhook
type Config struct {
	// Server configuration
	Port    int    `yaml:"port,omitempty"`
	CertDir string `yaml:"cert-dir,omitempty"`

	// Logging
	Debug bool `yaml:"debug,omitempty"`

	// VM matching rules
	Rules []NamespaceRuleConfig `yaml:"rules,omitempty"`

	// parsedRules holds the compiled regex patterns for efficient matching
	parsedRules []NamespaceRule
}

func (c *Config) GetParsedRules() []NamespaceRule {
	log := slog.Default()
	if c.parsedRules == nil {
		parsed := make([]NamespaceRule, 0, len(c.Rules))
		for _, rule := range c.Rules {
			patterns := make([]*regexp.Regexp, 0, len(rule.Patterns))
			for _, patternStr := range rule.Patterns {
				regx, err := regexp.Compile(patternStr)
				if err != nil {
					log.Warn("ignoring invalid regex pattern", "pattern", patternStr, "namespace", rule.Namespace)
					continue
				}
				patterns = append(patterns, regx)
			}
			parsed = append(parsed, NamespaceRule{
				Namespace: rule.Namespace,
				Patterns:  patterns,
			})
		}
		c.parsedRules = parsed
	}
	return c.parsedRules
}

func LoadConfig(configFile string) (*Config, error) {
	cfg := Config{}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &cfg)
	return &cfg, err
}

// Matches checks if a VM in the given namespace with the given name matches any rule
func (c *Config) Matches(namespace, vmName string) bool {
	if c == nil {
		return false
	}

	rules := c.GetParsedRules()
	for _, rule := range rules {
		if rule.Namespace == namespace {
			for _, pattern := range rule.Patterns {
				if pattern.MatchString(vmName) {
					return true
				}
			}
		}
	}

	return false
}
