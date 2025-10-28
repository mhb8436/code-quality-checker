package rules

import (
	"regexp"
	"strings"

	"code-quality-checker/internal/config"
	"code-quality-checker/internal/parser"
	"code-quality-checker/internal/types"
)

// InnerHTMLXSSRule innerHTML XSS 취약점 검사
type InnerHTMLXSSRule struct {
	config config.RuleConfig
}

func NewInnerHTMLXSSRule(cfg config.RuleConfig) Rule {
	return &InnerHTMLXSSRule{config: cfg}
}

func (r *InnerHTMLXSSRule) ID() string                 { return r.config.ID }
func (r *InnerHTMLXSSRule) Name() string               { return r.config.Name }
func (r *InnerHTMLXSSRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *InnerHTMLXSSRule) Category() string          { return r.config.Category }
func (r *InnerHTMLXSSRule) Description() string       { return r.config.Description }

func (r *InnerHTMLXSSRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// innerHTML 사용 패턴 검사
	innerHTMLRegex := regexp.MustCompile(`\.innerHTML\s*=\s*[^;]+`)
	matches := innerHTMLRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range matches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		line := getLineContent(file, lineNum)
		
		// 안전한 패턴 제외 (escapeHtml, textContent 등)
		if r.isSafePattern(line) {
			continue
		}

		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        lineNum,
			Column:      getColumnFromPosition(file.Content, match[0]),
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "innerHTML 사용으로 인한 XSS 취약점 위험",
			Description: "사용자 입력을 innerHTML에 직접 할당하면 XSS 공격에 취약합니다",
			Suggestion:  "textContent를 사용하거나 입력값을 이스케이프 처리하세요",
			CodeSnippet: strings.TrimSpace(line),
		})
	}

	return issues
}

func (r *InnerHTMLXSSRule) isSafePattern(line string) bool {
	safePatterns := []string{
		"escapeHtml", "sanitize", "textContent", "createTextNode",
	}
	
	for _, pattern := range safePatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

// MemoryLeakRule 메모리 누수 검사
type MemoryLeakRule struct {
	config config.RuleConfig
}

func NewMemoryLeakRule(cfg config.RuleConfig) Rule {
	return &MemoryLeakRule{config: cfg}
}

func (r *MemoryLeakRule) ID() string                 { return r.config.ID }
func (r *MemoryLeakRule) Name() string               { return r.config.Name }
func (r *MemoryLeakRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *MemoryLeakRule) Category() string          { return r.config.Category }
func (r *MemoryLeakRule) Description() string       { return r.config.Description }

func (r *MemoryLeakRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// 이벤트 리스너 추가 패턴
	addEventRegex := regexp.MustCompile(`addEventListener\s*\(\s*['"][^'"]+['"]`)
	addMatches := addEventRegex.FindAllStringIndex(file.Content, -1)

	// 이벤트 리스너 제거 패턴
	removeEventRegex := regexp.MustCompile(`removeEventListener\s*\(\s*['"][^'"]+['"]`)
	removeMatches := removeEventRegex.FindAllStringIndex(file.Content, -1)

	// setInterval/setTimeout 패턴
	intervalRegex := regexp.MustCompile(`setInterval\s*\(`)
	intervalMatches := intervalRegex.FindAllStringIndex(file.Content, -1)

	timeoutRegex := regexp.MustCompile(`setTimeout\s*\(`)
	timeoutMatches := timeoutRegex.FindAllStringIndex(file.Content, -1)

	// clearInterval/clearTimeout 패턴
	clearIntervalRegex := regexp.MustCompile(`clearInterval\s*\(`)
	clearIntervalMatches := clearIntervalRegex.FindAllStringIndex(file.Content, -1)

	clearTimeoutRegex := regexp.MustCompile(`clearTimeout\s*\(`)
	clearTimeoutMatches := clearTimeoutRegex.FindAllStringIndex(file.Content, -1)

	// 이벤트 리스너 누수 검사
	if len(addMatches) > len(removeMatches) {
		for _, match := range addMatches[:len(addMatches)-len(removeMatches)] {
			lineNum := getLineNumberFromPosition(file.Content, match[0])
			
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, match[0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "이벤트 리스너가 제거되지 않아 메모리 누수 위험이 있습니다",
				Description: "addEventListener 후 removeEventListener가 호출되지 않습니다",
				Suggestion:  "컴포넌트 해제 시 removeEventListener를 호출하세요",
				CodeSnippet: getLineContent(file, lineNum),
			})
		}
	}

	// 타이머 누수 검사
	totalTimers := len(intervalMatches) + len(timeoutMatches)
	totalClears := len(clearIntervalMatches) + len(clearTimeoutMatches)
	
	if totalTimers > totalClears {
		for _, match := range intervalMatches {
			lineNum := getLineNumberFromPosition(file.Content, match[0])
			
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, match[0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "타이머가 정리되지 않아 메모리 누수 위험이 있습니다",
				Description: "setInterval/setTimeout 후 clear 함수가 호출되지 않습니다",
				Suggestion:  "컴포넌트 해제 시 clearInterval/clearTimeout을 호출하세요",
				CodeSnippet: getLineContent(file, lineNum),
			})
		}
	}

	return issues
}

// FunctionLengthRule JavaScript 함수 길이 검사
type FunctionLengthRule struct {
	config config.RuleConfig
}

func NewFunctionLengthRule(cfg config.RuleConfig) Rule {
	return &FunctionLengthRule{config: cfg}
}

func (r *FunctionLengthRule) ID() string                 { return r.config.ID }
func (r *FunctionLengthRule) Name() string               { return r.config.Name }
func (r *FunctionLengthRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *FunctionLengthRule) Category() string          { return r.config.Category }
func (r *FunctionLengthRule) Description() string       { return r.config.Description }

func (r *FunctionLengthRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	functions, ok := file.AST.([]parser.JSFunction)
	if !ok {
		return issues
	}

	for _, function := range functions {
		functionLength := r.calculateFunctionLength(file, function)
		
		if functionLength > 30 { // JavaScript 함수 길이 임계값
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        function.Line,
				Column:      function.Column,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "함수가 너무 깁니다 (" + function.Name + ": " + intToString(functionLength) + " 라인)",
				Description: "긴 함수는 가독성과 유지보수성을 저하시킵니다",
				Suggestion:  "함수를 더 작은 단위로 분할하세요",
				CodeSnippet: getLineContent(file, function.Line),
			})
		}
	}

	return issues
}

func (r *FunctionLengthRule) calculateFunctionLength(file *parser.ParsedFile, function parser.JSFunction) int {
	// 함수 시작 라인부터 닫는 브레이스까지의 라인 수 계산
	// 간단한 구현: 함수 이름으로 시작해서 다음 함수까지의 라인 수
	functionPattern := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(function.Name) + `.*?\{.*?\}`)
	match := functionPattern.FindString(file.Content)
	
	if match == "" {
		return 0
	}
	
	return strings.Count(match, "\n")
}

// ConsoleLogRule console.log 사용 검사
type ConsoleLogRule struct {
	config config.RuleConfig
}

func NewConsoleLogRule(cfg config.RuleConfig) Rule {
	return &ConsoleLogRule{config: cfg}
}

func (r *ConsoleLogRule) ID() string                 { return r.config.ID }
func (r *ConsoleLogRule) Name() string               { return r.config.Name }
func (r *ConsoleLogRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *ConsoleLogRule) Category() string          { return r.config.Category }
func (r *ConsoleLogRule) Description() string       { return r.config.Description }

func (r *ConsoleLogRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	consoleRegex := regexp.MustCompile(`console\.(log|warn|error|info|debug)`)
	matches := consoleRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range matches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        lineNum,
			Column:      getColumnFromPosition(file.Content, match[0]),
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "console.log 사용이 발견되었습니다",
			Description: "프로덕션 환경에서 console 출력은 성능에 영향을 줄 수 있습니다",
			Suggestion:  "적절한 로깅 라이브러리를 사용하거나 프로덕션에서 제거하세요",
			CodeSnippet: getLineContent(file, lineNum),
		})
	}

	return issues
}

// VarUsageRule var 키워드 사용 검사
type VarUsageRule struct {
	config config.RuleConfig
}

func NewVarUsageRule(cfg config.RuleConfig) Rule {
	return &VarUsageRule{config: cfg}
}

func (r *VarUsageRule) ID() string                 { return r.config.ID }
func (r *VarUsageRule) Name() string               { return r.config.Name }
func (r *VarUsageRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *VarUsageRule) Category() string          { return r.config.Category }
func (r *VarUsageRule) Description() string       { return r.config.Description }

func (r *VarUsageRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// var 키워드 사용 패턴
	varRegex := regexp.MustCompile(`\bvar\s+\w+`)
	matches := varRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range matches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		line := getLineContent(file, lineNum)
		
		// 주석 안의 var는 제외
		if strings.Contains(line, "//") && strings.Index(line, "//") < strings.Index(line, "var") {
			continue
		}
		
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        lineNum,
			Column:      getColumnFromPosition(file.Content, match[0]),
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "var 키워드 사용이 발견되었습니다",
			Description: "var는 호이스팅과 스코프 문제를 일으킬 수 있습니다",
			Suggestion:  "let 또는 const를 사용하세요",
			CodeSnippet: strings.TrimSpace(line),
		})
	}

	return issues
}

// 헬퍼 함수
func getLineContent(file *parser.ParsedFile, lineNum int) string {
	if lineNum <= 0 || lineNum > len(file.Lines) {
		return ""
	}
	return file.Lines[lineNum-1]
}