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
        body { font-family: Arial, sans-serif; margin: 0; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: #2c3e50; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
        .tabs { background: white; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .tab-buttons { display: flex; border-bottom: 1px solid #ddd; }
        .tab-button { padding: 15px 20px; background: none; border: none; cursor: pointer; font-size: 16px; border-bottom: 3px solid transparent; transition: all 0.3s; }
        .tab-button.active { background-color: #3498db; color: white; border-bottom-color: #2980b9; }
        .tab-button:hover { background-color: #ecf0f1; }
        .tab-button.active:hover { background-color: #2980b9; }
        .tab-content { padding: 20px; min-height: 400px; }
        .tab-pane { display: none; }
        .tab-pane.active { display: block; }
        .stats { display: flex; gap: 20px; flex-wrap: wrap; margin-bottom: 20px; }
        .stat-card { background: #ecf0f1; padding: 15px; border-radius: 8px; flex: 1; min-width: 200px; text-align: center; }
        .severity-badge { display: inline-block; padding: 4px 8px; border-radius: 4px; color: white; font-size: 12px; font-weight: bold; }
        .critical { background-color: #e74c3c; }
        .high { background-color: #f39c12; }
        .medium { background-color: #3498db; }
        .low { background-color: #27ae60; }
        .rule-nav { background: #f8f9fa; padding: 15px; border-radius: 8px; margin-bottom: 20px; }
        .rule-nav h3 { margin-top: 0; margin-bottom: 10px; }
        .rule-buttons { display: flex; flex-wrap: wrap; gap: 8px; }
        .rule-button { padding: 8px 12px; background: #3498db; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 14px; }
        .rule-button:hover { background: #2980b9; }
        .issue { border-left: 4px solid #e74c3c; margin-bottom: 15px; padding: 15px; background: #fafafa; border-radius: 4px; }
        .issue.critical { border-left-color: #e74c3c; }
        .issue.high { border-left-color: #f39c12; }
        .issue.medium { border-left-color: #3498db; }
        .issue.low { border-left-color: #27ae60; }
        .code-snippet { background: #2c3e50; color: #ecf0f1; padding: 10px; border-radius: 4px; font-family: monospace; margin-top: 10px; }
        .file-path { color: #7f8c8d; font-family: monospace; font-size: 14px; }
        .collapsible { cursor: pointer; padding: 10px; background: #e8f4f8; border: 1px solid #d4e6ea; border-radius: 4px; margin-bottom: 5px; }
        .collapsible:hover { background: #d4e6ea; }
        .collapsible.active { background: #3498db; color: white; }
        .collapsible-content { display: none; padding: 15px; border: 1px solid #ddd; border-top: none; }
        h1, h2, h3 { margin-top: 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔍 Code Quality Report</h1>
            <p>분석 완료 시간: ` + result.EndTime.Format("2006-01-02 15:04:05") + `</p>
            <p>분석 시간: ` + fmt.Sprintf("%.2f초", result.Duration.Seconds()) + `</p>
        </div>

        <div class="tabs">
            <div class="tab-buttons">
                <button class="tab-button active" onclick="showTab('overview')">전체 요약</button>
                <button class="tab-button" onclick="showTab('rules')">규칙별</button>
                <button class="tab-button" onclick="showTab('severity')">심각도별</button>
                <button class="tab-button" onclick="showTab('files')">파일별</button>
            </div>
            
            <div class="tab-content">
                ` + r.generateOverviewTab(result) + `
                ` + r.generateRulesByTab(result) + `
                ` + r.generateSeverityTab(result) + `
                ` + r.generateFilesTab(result) + `
            </div>
        </div>
    </div>

    <script>
        function showTab(tabName) {
            // 모든 탭 버튼 비활성화
            var buttons = document.querySelectorAll('.tab-button');
            buttons.forEach(btn => btn.classList.remove('active'));
            
            // 모든 탭 패널 숨기기
            var panes = document.querySelectorAll('.tab-pane');
            panes.forEach(pane => pane.classList.remove('active'));
            
            // 선택된 탭 활성화
            event.target.classList.add('active');
            document.getElementById(tabName + '-tab').classList.add('active');
        }
        
        function scrollToRule(ruleId) {
            var element = document.getElementById('rule-' + ruleId);
            if (element) {
                element.scrollIntoView({ behavior: 'smooth', block: 'start' });
                element.style.backgroundColor = '#fff3cd';
                setTimeout(() => {
                    element.style.backgroundColor = '';
                }, 2000);
            }
        }
        
        function toggleCollapsible(element) {
            element.classList.toggle('active');
            var content = element.nextElementSibling;
            if (content.style.display === 'block') {
                content.style.display = 'none';
            } else {
                content.style.display = 'block';
            }
        }
    </script>
</body>
</html>`)

	return html.String()
}

func (r *HTMLReporter) generateOverviewTab(result *types.AnalysisResult) string {
	var html strings.Builder
	
	html.WriteString(`<div id="overview-tab" class="tab-pane active">
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

	html.WriteString(`</div>`)

	// 언어별 통계
	if len(result.Summary.LanguageCount) > 0 {
		html.WriteString(`<h3>💻 언어별 파일 수</h3><div class="stats">`)
		for language, count := range result.Summary.LanguageCount {
			html.WriteString(`
			<div class="stat-card">
				<h3>` + fmt.Sprintf("%d", count) + `</h3>
				<p>` + language + `</p>
			</div>`)
		}
		html.WriteString(`</div>`)
	}

	html.WriteString(`</div>`)
	return html.String()
}

func (r *HTMLReporter) generateRulesByTab(result *types.AnalysisResult) string {
	var html strings.Builder
	
	html.WriteString(`<div id="rules-tab" class="tab-pane">
		<h2>📋 규칙별 분석</h2>`)

	// 규칙별 네비게이션
	issuesByRule := r.groupIssuesByRule(result.Issues)
	if len(issuesByRule) > 0 {
		html.WriteString(`<div class="rule-nav">
			<h3>규칙 선택 (섹션 이동)</h3>
			<div class="rule-buttons">`)
		
		for ruleID, issues := range issuesByRule {
			html.WriteString(`<button class="rule-button" onclick="scrollToRule('` + ruleID + `')">` + ruleID + ` (` + fmt.Sprintf("%d", len(issues)) + `)</button>`)
		}
		
		html.WriteString(`</div></div>`)

		// 규칙별 이슈 표시
		for ruleID, issues := range issuesByRule {
			html.WriteString(`<div id="rule-` + ruleID + `" class="collapsible" onclick="toggleCollapsible(this)">
				<h3>` + ruleID + ` (` + fmt.Sprintf("%d", len(issues)) + `개 이슈)</h3>
			</div>
			<div class="collapsible-content">`)

			for _, issue := range issues {
				html.WriteString(`
				<div class="issue ` + issue.Severity.String() + `">
					<div class="file-path">` + issue.File + `:` + fmt.Sprintf("%d", issue.Line) + `:` + fmt.Sprintf("%d", issue.Column) + `</div>
					<h4>` + issue.Message + ` <span class="severity-badge ` + issue.Severity.String() + `">` + strings.ToUpper(issue.Severity.String()) + `</span></h4>
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
	} else {
		html.WriteString(`<p>✅ 발견된 이슈가 없습니다!</p>`)
	}

	html.WriteString(`</div>`)
	return html.String()
}

func (r *HTMLReporter) generateSeverityTab(result *types.AnalysisResult) string {
	var html strings.Builder
	
	html.WriteString(`<div id="severity-tab" class="tab-pane">
		<h2>⚠️ 심각도별 분석</h2>`)

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

		html.WriteString(`<div class="collapsible" onclick="toggleCollapsible(this)">
			<h3><span class="severity-badge ` + severity.String() + `">` + strings.ToUpper(severity.String()) + `</span> (` + fmt.Sprintf("%d", len(issues)) + `개 이슈)</h3>
		</div>
		<div class="collapsible-content">`)

		for _, issue := range issues {
			html.WriteString(`
			<div class="issue ` + issue.Severity.String() + `">
				<div class="file-path">` + issue.File + `:` + fmt.Sprintf("%d", issue.Line) + `:` + fmt.Sprintf("%d", issue.Column) + `</div>
				<h4>` + issue.Message + `</h4>
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

	html.WriteString(`</div>`)
	return html.String()
}

func (r *HTMLReporter) generateFilesTab(result *types.AnalysisResult) string {
	var html strings.Builder
	
	html.WriteString(`<div id="files-tab" class="tab-pane">
		<h2>📁 파일별 분석</h2>`)

	issuesByFile := r.groupIssuesByFile(result.Issues)
	if len(issuesByFile) > 0 {
		for file, issues := range issuesByFile {
			html.WriteString(`<div class="collapsible" onclick="toggleCollapsible(this)">
				<h3>` + file + ` (` + fmt.Sprintf("%d", len(issues)) + `개 이슈)</h3>
			</div>
			<div class="collapsible-content">`)

			for _, issue := range issues {
				html.WriteString(`
				<div class="issue ` + issue.Severity.String() + `">
					<div class="file-path">Line ` + fmt.Sprintf("%d", issue.Line) + `, Column ` + fmt.Sprintf("%d", issue.Column) + `</div>
					<h4>` + issue.Message + ` <span class="severity-badge ` + issue.Severity.String() + `">` + strings.ToUpper(issue.Severity.String()) + `</span></h4>
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
	} else {
		html.WriteString(`<p>✅ 발견된 이슈가 없습니다!</p>`)
	}

	html.WriteString(`</div>`)
	return html.String()
}

func (r *HTMLReporter) groupIssuesByRule(issues []types.Issue) map[string][]types.Issue {
	grouped := make(map[string][]types.Issue)
	for _, issue := range issues {
		grouped[issue.RuleID] = append(grouped[issue.RuleID], issue)
	}
	return grouped
}

func (r *HTMLReporter) groupIssuesByFile(issues []types.Issue) map[string][]types.Issue {
	grouped := make(map[string][]types.Issue)
	for _, issue := range issues {
		grouped[issue.File] = append(grouped[issue.File], issue)
	}
	return grouped
}

func (r *HTMLReporter) groupIssuesBySeverity(issues []types.Issue) map[config.Severity][]types.Issue {
	grouped := make(map[config.Severity][]types.Issue)
	
	for _, issue := range issues {
		grouped[issue.Severity] = append(grouped[issue.Severity], issue)
	}
	
	return grouped
}

func (r *HTMLReporter) writeToFile(content string, filename string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}