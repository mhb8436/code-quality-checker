package types

import (
	"time"

	"code-quality-checker/internal/config"
)

// Issue 코드 품질 이슈
type Issue struct {
	RuleID      string           `json:"rule_id"`
	File        string           `json:"file"`
	Line        int              `json:"line"`
	Column      int              `json:"column"`
	Severity    config.Severity  `json:"severity"`
	Category    string           `json:"category"`
	Message     string           `json:"message"`
	Description string           `json:"description"`
	Suggestion  string           `json:"suggestion,omitempty"`
	CodeSnippet string           `json:"code_snippet,omitempty"`
}

// Summary 분석 요약 정보
type Summary struct {
	TotalFiles     int                        `json:"total_files"`
	TotalIssues    int                        `json:"total_issues"`
	SeverityCount  map[config.Severity]int    `json:"severity_count"`
	CategoryCount  map[string]int             `json:"category_count"`
	LanguageCount  map[string]int             `json:"language_count"`
}

// AnalysisResult 분석 결과
type AnalysisResult struct {
	Summary   Summary       `json:"summary"`
	Issues    []Issue       `json:"issues"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Config    interface{}   `json:"config,omitempty"`
}

// HasCriticalIssues 심각한 이슈가 있는지 확인
func (r *AnalysisResult) HasCriticalIssues() bool {
	return r.Summary.SeverityCount[config.SeverityCritical] > 0
}