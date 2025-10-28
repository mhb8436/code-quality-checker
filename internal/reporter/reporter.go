package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"code-quality-checker/internal/config"
	"code-quality-checker/internal/types"
)

// Reporter 리포터 인터페이스
type Reporter interface {
	Generate(result *types.AnalysisResult, outputFile string) error
}

// New 새로운 리포터 생성
func New(format string) (Reporter, error) {
	switch strings.ToLower(format) {
	case "console", "text":
		return &ConsoleReporter{}, nil
	case "json":
		return &JSONReporter{}, nil
	case "html":
		return &HTMLReporter{}, nil
	default:
		return nil, fmt.Errorf("지원하지 않는 출력 형식: %s", format)
	}
}

// ConsoleReporter 콘솔 출력 리포터
type ConsoleReporter struct{}

func (r *ConsoleReporter) Generate(result *types.AnalysisResult, outputFile string) error {
	var output strings.Builder

	// 헤더 출력
	output.WriteString("🔍 Code Quality Checker 분석 결과\n")
	output.WriteString(strings.Repeat("=", 50) + "\n\n")

	// 요약 정보
	output.WriteString("📊 분석 요약\n")
	output.WriteString(strings.Repeat("-", 20) + "\n")
	output.WriteString(fmt.Sprintf("검사 파일 수: %d개\n", result.Summary.TotalFiles))
	output.WriteString(fmt.Sprintf("발견된 이슈: %d개\n", result.Summary.TotalIssues))
	output.WriteString(fmt.Sprintf("분석 시간: %.2f초\n\n", result.Duration.Seconds()))

	// 심각도별 통계
	if result.Summary.TotalIssues > 0 {
		output.WriteString("⚠️  심각도별 통계\n")
		output.WriteString(strings.Repeat("-", 20) + "\n")
		for severity, count := range result.Summary.SeverityCount {
			if count > 0 {
				emoji := r.getSeverityEmoji(severity)
				output.WriteString(fmt.Sprintf("%s %s: %d개\n", emoji, severity.String(), count))
			}
		}
		output.WriteString("\n")

		// 카테고리별 통계
		output.WriteString("📂 카테고리별 통계\n")
		output.WriteString(strings.Repeat("-", 20) + "\n")
		for category, count := range result.Summary.CategoryCount {
			output.WriteString(fmt.Sprintf("  %s: %d개\n", category, count))
		}
		output.WriteString("\n")

		// 이슈 상세 목록
		output.WriteString("🐛 발견된 이슈 목록\n")
		output.WriteString(strings.Repeat("=", 50) + "\n\n")

		// 심각도별로 그룹화하여 출력
		issuesBySeverity := r.groupIssuesBySeverity(result.Issues)
		
		severityOrder := []config.Severity{
			config.SeverityCritical,
			config.SeverityHigh,
			config.SeverityMedium,
			config.SeverityLow,
		}

		for _, severity := range severityOrder {
			issues, exists := issuesBySeverity[severity]
			if !exists || len(issues) == 0 {
				continue
			}

			emoji := r.getSeverityEmoji(severity)
			output.WriteString(fmt.Sprintf("%s %s 이슈 (%d개)\n", emoji, strings.ToUpper(severity.String()), len(issues)))
			output.WriteString(strings.Repeat("-", 30) + "\n")

			for i, issue := range issues {
				if i >= 10 { // 각 심각도별로 최대 10개까지만 표시
					output.WriteString(fmt.Sprintf("  ... 및 %d개 추가 이슈\n", len(issues)-i))
					break
				}

				output.WriteString(fmt.Sprintf("  📁 %s:%d:%d\n", issue.File, issue.Line, issue.Column))
				output.WriteString(fmt.Sprintf("     [%s] %s\n", issue.RuleID, issue.Message))
				if issue.Suggestion != "" {
					output.WriteString(fmt.Sprintf("     💡 %s\n", issue.Suggestion))
				}
				if issue.CodeSnippet != "" {
					output.WriteString(fmt.Sprintf("     📋 %s\n", issue.CodeSnippet))
				}
				output.WriteString("\n")
			}
		}
	} else {
		output.WriteString("✅ 이슈가 발견되지 않았습니다!\n\n")
	}

	// 언어별 통계
	if len(result.Summary.LanguageCount) > 0 {
		output.WriteString("💻 언어별 파일 수\n")
		output.WriteString(strings.Repeat("-", 20) + "\n")
		for language, count := range result.Summary.LanguageCount {
			output.WriteString(fmt.Sprintf("  %s: %d개\n", language, count))
		}
		output.WriteString("\n")
	}

	// 권장사항
	if result.Summary.TotalIssues > 0 {
		output.WriteString("💡 권장사항\n")
		output.WriteString(strings.Repeat("-", 20) + "\n")
		
		if result.Summary.SeverityCount[config.SeverityCritical] > 0 {
			output.WriteString("🚨 Critical 이슈는 즉시 수정이 필요합니다!\n")
		}
		if result.Summary.SeverityCount[config.SeverityHigh] > 0 {
			output.WriteString("⚠️  High 이슈는 릴리즈 전에 수정하세요.\n")
		}
		if result.Summary.SeverityCount[config.SeverityMedium] > 0 {
			output.WriteString("📝 Medium 이슈는 점진적으로 개선하세요.\n")
		}
	}

	// 출력
	if outputFile != "" {
		return r.writeToFile(output.String(), outputFile)
	} else {
		fmt.Print(output.String())
		return nil
	}
}

func (r *ConsoleReporter) getSeverityEmoji(severity config.Severity) string {
	switch severity {
	case config.SeverityCritical:
		return "🚨"
	case config.SeverityHigh:
		return "⚠️"
	case config.SeverityMedium:
		return "📝"
	case config.SeverityLow:
		return "💡"
	default:
		return "❓"
	}
}

func (r *ConsoleReporter) groupIssuesBySeverity(issues []types.Issue) map[config.Severity][]types.Issue {
	grouped := make(map[config.Severity][]types.Issue)
	
	for _, issue := range issues {
		grouped[issue.Severity] = append(grouped[issue.Severity], issue)
	}
	
	return grouped
}

func (r *ConsoleReporter) writeToFile(content string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// JSONReporter JSON 출력 리포터
type JSONReporter struct{}

func (r *JSONReporter) Generate(result *types.AnalysisResult, outputFile string) error {
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 마샬링 실패: %w", err)
	}

	if outputFile != "" {
		return r.writeToFile(jsonData, outputFile)
	} else {
		fmt.Print(string(jsonData))
		return nil
	}
}

func (r *JSONReporter) writeToFile(data []byte, filename string) error {
	return os.WriteFile(filename, data, 0644)
}

// HTMLReporter HTML 출력 리포터
type HTMLReporter struct{}

func (r *HTMLReporter) Generate(result *types.AnalysisResult, outputFile string) error {
	html := r.generateHTML(result)

	if outputFile != "" {
		return r.writeToFile(html, outputFile)
	} else {
		fmt.Print(html)
		return nil
	}
}

func (r *HTMLReporter) generateHTML(result *types.AnalysisResult) string {
	var html strings.Builder

	html.WriteString(`<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Code Quality Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; }
        .header { background: #2c3e50; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .summary { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .issues { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .issue { border-left: 4px solid #e74c3c; margin-bottom: 15px; padding: 15px; background: #fafafa; }
        .issue.critical { border-left-color: #e74c3c; }
        .issue.high { border-left-color: #f39c12; }
        .issue.medium { border-left-color: #3498db; }
        .issue.low { border-left-color: #27ae60; }
        .severity-badge { display: inline-block; padding: 4px 8px; border-radius: 4px; color: white; font-size: 12px; font-weight: bold; }
        .critical { background-color: #e74c3c; }
        .high { background-color: #f39c12; }
        .medium { background-color: #3498db; }
        .low { background-color: #27ae60; }
        .stats { display: flex; gap: 20px; flex-wrap: wrap; }
        .stat-card { background: #ecf0f1; padding: 15px; border-radius: 8px; flex: 1; min-width: 200px; }
        .code-snippet { background: #2c3e50; color: #ecf0f1; padding: 10px; border-radius: 4px; font-family: monospace; margin-top: 10px; }
        h1, h2, h3 { margin-top: 0; }
        .file-path { color: #7f8c8d; font-family: monospace; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔍 Code Quality Report</h1>
            <p>분석 완료 시간: ` + result.EndTime.Format("2006-01-02 15:04:05") + `</p>
            <p>분석 시간: ` + fmt.Sprintf("%.2f초", result.Duration.Seconds()) + `</p>
        </div>

        <div class="summary">
            <h2>📊 분석 요약</h2>
            <div class="stats">
                <div class="stat-card">
                    <h3>` + fmt.Sprintf("%d", result.Summary.TotalFiles) + `</h3>
                    <p>검사된 파일</p>
                </div>
                <div class="stat-card">
                    <h3>` + fmt.Sprintf("%d", result.Summary.TotalIssues) + `</h3>
                    <p>발견된 이슈</p>
                </div>`)

	// 심각도별 통계
	for severity, count := range result.Summary.SeverityCount {
		if count > 0 {
			html.WriteString(`
                <div class="stat-card">
                    <h3>` + fmt.Sprintf("%d", count) + `</h3>
                    <p><span class="severity-badge ` + severity.String() + `">` + strings.ToUpper(severity.String()) + `</span></p>
                </div>`)
		}
	}

	html.WriteString(`
            </div>
        </div>`)

	// 이슈 목록
	if result.Summary.TotalIssues > 0 {
		html.WriteString(`
        <div class="issues">
            <h2>🐛 발견된 이슈</h2>`)

		for _, issue := range result.Issues {
			html.WriteString(`
            <div class="issue ` + issue.Severity.String() + `">
                <div class="file-path">` + issue.File + `:` + fmt.Sprintf("%d", issue.Line) + `:` + fmt.Sprintf("%d", issue.Column) + `</div>
                <h3>` + issue.Message + ` <span class="severity-badge ` + issue.Severity.String() + `">` + strings.ToUpper(issue.Severity.String()) + `</span></h3>
                <p><strong>규칙:</strong> ` + issue.RuleID + `</p>
                <p><strong>카테고리:</strong> ` + issue.Category + `</p>`)

			if issue.Description != "" {
				html.WriteString(`<p><strong>설명:</strong> ` + issue.Description + `</p>`)
			}

			if issue.Suggestion != "" {
				html.WriteString(`<p><strong>💡 권장사항:</strong> ` + issue.Suggestion + `</p>`)
			}

			if issue.CodeSnippet != "" {
				html.WriteString(`<div class="code-snippet">` + issue.CodeSnippet + `</div>`)
			}

			html.WriteString(`</div>`)
		}

		html.WriteString(`</div>`)
	}

	html.WriteString(`
    </div>
</body>
</html>`)

	return html.String()
}

func (r *HTMLReporter) writeToFile(content string, filename string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}