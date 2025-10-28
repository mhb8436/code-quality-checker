package rules

import (
	"regexp"
	"strings"

	"code-quality-checker/internal/config"
	"code-quality-checker/internal/parser"
	"code-quality-checker/internal/types"
)

// CSSSelectorsRule CSS 셀렉터 효율성 검사
type CSSSelectorsRule struct {
	config config.RuleConfig
}

func NewCSSSelectorsRule(cfg config.RuleConfig) Rule {
	return &CSSSelectorsRule{config: cfg}
}

func (r *CSSSelectorsRule) ID() string                 { return r.config.ID }
func (r *CSSSelectorsRule) Name() string               { return r.config.Name }
func (r *CSSSelectorsRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *CSSSelectorsRule) Category() string          { return r.config.Category }
func (r *CSSSelectorsRule) Description() string       { return r.config.Description }

func (r *CSSSelectorsRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	cssData, ok := file.AST.(map[string]interface{})
	if !ok {
		return issues
	}

	selectors, exists := cssData["selectors"]
	if !exists {
		return issues
	}

	selectorList, ok := selectors.([]string)
	if !ok {
		return issues
	}

	for _, selector := range selectorList {
		selector = strings.TrimSpace(selector)
		lineNum := r.findLineNumber(file, selector)

		// 과도한 중첩 셀렉터 검사 (4단계 이상)
		if r.isOverlyNested(selector) {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      0,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "과도하게 중첩된 CSS 셀렉터입니다",
				Description: "깊은 중첩은 CSS 성능을 저하시키고 유지보수를 어렵게 합니다",
				Suggestion:  "셀렉터 중첩을 3단계 이하로 줄이세요",
				CodeSnippet: selector,
			})
		}

		// 전체 셀렉터(*) 남용 검사
		if r.hasUniversalSelector(selector) {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      0,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "전체 셀렉터(*) 사용이 발견되었습니다",
				Description: "전체 셀렉터는 모든 요소를 검사하여 성능을 저하시킵니다",
				Suggestion:  "더 구체적인 셀렉터를 사용하세요",
				CodeSnippet: selector,
			})
		}

		// 비효율적인 자손 셀렉터 검사
		if r.isInefficientDescendantSelector(selector) {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      0,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "비효율적인 자손 셀렉터입니다",
				Description: "태그명으로 시작하는 복잡한 셀렉터는 성능이 떨어집니다",
				Suggestion:  "클래스나 ID로 시작하는 셀렉터를 사용하세요",
				CodeSnippet: selector,
			})
		}

		// ID 셀렉터 과다 사용 검사
		if r.hasMultipleIds(selector) {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      0,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "여러 ID 셀렉터가 사용되었습니다",
				Description: "ID는 문서에서 고유해야 하므로 여러 ID 셀렉터는 불필요합니다",
				Suggestion:  "하나의 ID만 사용하거나 클래스 셀렉터를 사용하세요",
				CodeSnippet: selector,
			})
		}
	}

	// 중복 스타일 검사
	issues = append(issues, r.checkDuplicateStyles(file)...)

	return issues
}

func (r *CSSSelectorsRule) isOverlyNested(selector string) bool {
	// 공백으로 구분된 셀렉터의 깊이 계산
	parts := strings.Fields(selector)
	return len(parts) > 4
}

func (r *CSSSelectorsRule) hasUniversalSelector(selector string) bool {
	// * 셀렉터 사용 검사
	return strings.Contains(selector, "*")
}

func (r *CSSSelectorsRule) isInefficientDescendantSelector(selector string) bool {
	// 태그명으로 시작하고 복잡한 구조인지 검사
	parts := strings.Fields(selector)
	if len(parts) < 3 {
		return false
	}

	// 첫 번째 부분이 태그명인지 확인 (소문자로만 구성)
	firstPart := parts[0]
	if regexp.MustCompile(`^[a-z]+$`).MatchString(firstPart) && len(parts) > 3 {
		return true
	}

	return false
}

func (r *CSSSelectorsRule) hasMultipleIds(selector string) bool {
	// ID 셀렉터(#) 개수 확인
	idCount := strings.Count(selector, "#")
	return idCount > 1
}

func (r *CSSSelectorsRule) checkDuplicateStyles(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// CSS 규칙 블록 추출
	ruleRegex := regexp.MustCompile(`([^{}]+)\s*\{([^{}]*)\}`)
	matches := ruleRegex.FindAllStringSubmatch(file.Content, -1)

	styleBlocks := make(map[string][]string) // 스타일 -> 셀렉터 목록

	for _, match := range matches {
		if len(match) >= 3 {
			selector := strings.TrimSpace(match[1])
			styles := strings.TrimSpace(match[2])

			// 스타일 정규화 (공백, 세미콜론 정리)
			normalizedStyles := r.normalizeStyles(styles)

			if normalizedStyles != "" {
				styleBlocks[normalizedStyles] = append(styleBlocks[normalizedStyles], selector)
			}
		}
	}

	// 중복 스타일 블록 찾기
	for styles, selectors := range styleBlocks {
		if len(selectors) > 1 {
			// 첫 번째 발생 위치만 보고
			lineNum := r.findLineNumber(file, selectors[0])

			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      0,
				Severity:    config.SeverityMedium,
				Category:    "performance",
				Message:     "중복된 CSS 스타일이 발견되었습니다",
				Description: "동일한 스타일이 여러 셀렉터에 중복 정의되어 있습니다",
				Suggestion:  "공통 클래스를 만들어 중복을 제거하세요",
				CodeSnippet: strings.Join(selectors, ", ") + " { " + styles + " }",
			})
		}
	}

	return issues
}

func (r *CSSSelectorsRule) normalizeStyles(styles string) string {
	// 스타일 정규화: 공백 제거, 정렬, 세미콜론 정리
	styles = regexp.MustCompile(`\s+`).ReplaceAllString(styles, " ")
	styles = strings.TrimSpace(styles)
	styles = strings.Trim(styles, ";")

	if styles == "" {
		return ""
	}

	// 각 속성을 분리하고 정렬
	properties := strings.Split(styles, ";")
	var normalizedProps []string

	for _, prop := range properties {
		prop = strings.TrimSpace(prop)
		if prop != "" {
			normalizedProps = append(normalizedProps, prop)
		}
	}

	return strings.Join(normalizedProps, ";")
}

func (r *CSSSelectorsRule) findLineNumber(file *parser.ParsedFile, text string) int {
	for i, line := range file.Lines {
		if strings.Contains(line, text) {
			return i + 1
		}
	}
	return 1
}

// ResponsiveDesignRule 반응형 디자인 검사
type ResponsiveDesignRule struct {
	config config.RuleConfig
}

func NewResponsiveDesignRule(cfg config.RuleConfig) Rule {
	return &ResponsiveDesignRule{config: cfg}
}

func (r *ResponsiveDesignRule) ID() string                 { return r.config.ID }
func (r *ResponsiveDesignRule) Name() string               { return r.config.Name }
func (r *ResponsiveDesignRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *ResponsiveDesignRule) Category() string          { return r.config.Category }
func (r *ResponsiveDesignRule) Description() string       { return r.config.Description }

func (r *ResponsiveDesignRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// 미디어 쿼리 사용 여부 검사
	hasMediaQueries := r.hasMediaQueries(file.Content)
	hasFixedWidths := r.hasFixedWidths(file.Content)

	if hasFixedWidths && !hasMediaQueries {
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        1,
			Column:      1,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "고정 너비를 사용하지만 미디어 쿼리가 없습니다",
			Description: "반응형 디자인을 위해 미디어 쿼리가 필요합니다",
			Suggestion:  "@media 쿼리를 추가하여 다양한 화면 크기에 대응하세요",
			CodeSnippet: "@media (max-width: 768px) { /* 모바일 스타일 */ }",
		})
	}

	// 고정 단위 사용 검사
	fixedUnitIssues := r.checkFixedUnits(file)
	issues = append(issues, fixedUnitIssues...)

	// flex/grid 사용 권장
	if !r.hasFlexOrGrid(file.Content) && r.hasLayoutProperties(file.Content) {
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        1,
			Column:      1,
			Severity:    config.SeverityLow,
			Category:    r.Category(),
			Message:     "모던 레이아웃 기법이 사용되지 않았습니다",
			Description: "Flexbox나 Grid를 사용하면 더 유연한 레이아웃을 만들 수 있습니다",
			Suggestion:  "display: flex 또는 display: grid를 고려해보세요",
			CodeSnippet: "display: flex; /* 또는 */ display: grid;",
		})
	}

	return issues
}

func (r *ResponsiveDesignRule) hasMediaQueries(content string) bool {
	mediaQueryRegex := regexp.MustCompile(`@media\s*\([^)]+\)`)
	return mediaQueryRegex.MatchString(content)
}

func (r *ResponsiveDesignRule) hasFixedWidths(content string) bool {
	fixedWidthRegex := regexp.MustCompile(`width\s*:\s*\d+px`)
	return fixedWidthRegex.MatchString(content)
}

func (r *ResponsiveDesignRule) hasViewportMeta(content string) bool {
	viewportRegex := regexp.MustCompile(`<meta[^>]*name\s*=\s*["']viewport["']`)
	return viewportRegex.MatchString(content)
}

func (r *ResponsiveDesignRule) hasFlexOrGrid(content string) bool {
	flexGridRegex := regexp.MustCompile(`display\s*:\s*(flex|grid)`)
	return flexGridRegex.MatchString(content)
}

func (r *ResponsiveDesignRule) hasLayoutProperties(content string) bool {
	layoutRegex := regexp.MustCompile(`(width|height|margin|padding|position)\s*:`)
	return layoutRegex.MatchString(content)
}

func (r *ResponsiveDesignRule) checkFixedUnits(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// px 단위 과다 사용 검사
	pxRegex := regexp.MustCompile(`:\s*\d+px`)
	matches := pxRegex.FindAllStringIndex(file.Content, -1)

	pxCount := len(matches)
	if pxCount > 10 { // 임계값: 10개 이상
		// 처음 몇 개만 보고
		for i, match := range matches[:3] {
			lineNum := getLineNumberFromPosition(file.Content, match[0])

			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, match[0]),
				Severity:    config.SeverityLow,
				Category:    r.Category(),
				Message:     "px 단위를 과도하게 사용하고 있습니다",
				Description: "고정 단위는 반응형 디자인에 제한적입니다",
				Suggestion:  "em, rem, %, vw, vh 등 상대 단위 사용을 고려하세요",
				CodeSnippet: r.getCodeSnippet(file, lineNum),
			})

			if i >= 2 { // 최대 3개까지만
				break
			}
		}
	}

	return issues
}

func (r *ResponsiveDesignRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}