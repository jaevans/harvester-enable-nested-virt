package config

import (
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
)

// NamespaceRule represents a namespace with its associated VM name patterns
type NamespaceRule struct {
	Namespace string
	Patterns  []*regexp.Regexp
}

// Config holds the configuration for the webhook
type Config struct {
	Rules []NamespaceRule
}

// ParseConfigMap parses a ConfigMap into a Config structure
// Expected format in ConfigMap data:
// namespace1: regex1,regex2,regex3
// namespace2: regex4,regex5
func ParseConfigMap(cm *corev1.ConfigMap) (*Config, error) {
	if cm == nil {
		return nil, fmt.Errorf("configmap is nil")
	}

	config := &Config{
		Rules: make([]NamespaceRule, 0),
	}

	for namespace, patternsStr := range cm.Data {
		if patternsStr == "" {
			continue
		}

		patterns := make([]*regexp.Regexp, 0)
		// Split by comma to get individual patterns
		for i := 0; i < len(patternsStr); {
			// Find the next comma or end of string
			end := i
			for end < len(patternsStr) && patternsStr[end] != ',' {
				end++
			}

			pattern := patternsStr[i:end]
			// Trim spaces
			for len(pattern) > 0 && (pattern[0] == ' ' || pattern[0] == '\t') {
				pattern = pattern[1:]
			}
			for len(pattern) > 0 && (pattern[len(pattern)-1] == ' ' || pattern[len(pattern)-1] == '\t') {
				pattern = pattern[:len(pattern)-1]
			}

			if pattern != "" {
				re, err := regexp.Compile(pattern)
				if err != nil {
					return nil, fmt.Errorf("invalid regex pattern '%s' for namespace '%s': %w", pattern, namespace, err)
				}
				patterns = append(patterns, re)
			}

			i = end + 1
		}

		if len(patterns) > 0 {
			config.Rules = append(config.Rules, NamespaceRule{
				Namespace: namespace,
				Patterns:  patterns,
			})
		}
	}

	return config, nil
}

// Matches checks if a VM in the given namespace with the given name matches any rule
func (c *Config) Matches(namespace, vmName string) bool {
	if c == nil {
		return false
	}

	for _, rule := range c.Rules {
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
