package config

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v3"
)

// Severity 심각도 열거형
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ParseSeverity 문자열을 Severity로 변환
func ParseSeverity(s string) Severity {
	switch strings.ToLower(s) {
	case "low":
		return SeverityLow
	case "medium":
		return SeverityMedium
	case "high":
		return SeverityHigh
	case "critical":
		return SeverityCritical
	default:
		return SeverityLow
	}
}

// RuleConfig 개별 규칙 설정
type RuleConfig struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Severity    string            `yaml:"severity"`
	Category    string            `yaml:"category"`
	Description string            `yaml:"description"`
	Enabled     bool              `yaml:"enabled"`
	Pattern     PatternConfig     `yaml:"pattern"`
	Exclude     []string          `yaml:"exclude,omitempty"`
	Custom      map[string]string `yaml:"custom,omitempty"`
}

// PatternConfig 패턴 매칭 설정
type PatternConfig struct {
	Type       string   `yaml:"type"`        // regex, ast-pattern, method-analysis
	Regex      string   `yaml:"regex,omitempty"`
	ASTPattern string   `yaml:"ast_pattern,omitempty"`
	Conditions []string `yaml:"conditions,omitempty"`
}

// LanguageRules 언어별 규칙
type LanguageRules struct {
	Language string       `yaml:"language"`
	Rules    []RuleConfig `yaml:"rules"`
}

// Config 전체 설정
type Config struct {
	Version   string          `yaml:"version"`
	Languages []LanguageRules `yaml:"languages"`
}

// LoadConfig 설정 파일 로드
func LoadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("설정 파일 읽기 실패: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("설정 파일 파싱 실패: %w", err)
	}

	// 기본값 설정
	for i := range config.Languages {
		for j := range config.Languages[i].Rules {
			rule := &config.Languages[i].Rules[j]
			if rule.Enabled == false && rule.ID != "" {
				rule.Enabled = true // 기본적으로 활성화
			}
		}
	}

	return &config, nil
}

// GetRulesForLanguage 특정 언어의 규칙 반환
func (c *Config) GetRulesForLanguage(language string) []RuleConfig {
	for _, langRules := range c.Languages {
		if langRules.Language == language {
			var enabledRules []RuleConfig
			for _, rule := range langRules.Rules {
				if rule.Enabled {
					enabledRules = append(enabledRules, rule)
				}
			}
			return enabledRules
		}
	}
	return nil
}

// FilterByCategories 카테고리별 필터링
func (c *Config) FilterByCategories(categories string) {
	if categories == "" {
		return
	}

	categoryList := strings.Split(categories, ",")
	categoryMap := make(map[string]bool)
	for _, cat := range categoryList {
		categoryMap[strings.TrimSpace(cat)] = true
	}

	for i := range c.Languages {
		var filteredRules []RuleConfig
		for _, rule := range c.Languages[i].Rules {
			if categoryMap[rule.Category] {
				filteredRules = append(filteredRules, rule)
			}
		}
		c.Languages[i].Rules = filteredRules
	}
}

// FilterBySeverity 심각도별 필터링
func (c *Config) FilterBySeverity(minSeverity Severity) {
	for i := range c.Languages {
		var filteredRules []RuleConfig
		for _, rule := range c.Languages[i].Rules {
			ruleSeverity := ParseSeverity(rule.Severity)
			if ruleSeverity >= minSeverity {
				filteredRules = append(filteredRules, rule)
			}
		}
		c.Languages[i].Rules = filteredRules
	}
}

// GetAllCategories 모든 카테고리 목록 반환
func (c *Config) GetAllCategories() []string {
	categoryMap := make(map[string]bool)
	for _, langRules := range c.Languages {
		for _, rule := range langRules.Rules {
			categoryMap[rule.Category] = true
		}
	}

	var categories []string
	for category := range categoryMap {
		categories = append(categories, category)
	}
	return categories
}