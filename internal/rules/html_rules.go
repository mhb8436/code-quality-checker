package rules

import (
	"regexp"
	"strings"

	"code-quality-checker/internal/config"
	"code-quality-checker/internal/parser"
	"code-quality-checker/internal/types"
)

// ImgAltRule img 태그 alt 속성 누락 검사
type ImgAltRule struct {
	config config.RuleConfig
}

func NewImgAltRule(cfg config.RuleConfig) Rule {
	return &ImgAltRule{config: cfg}
}

func (r *ImgAltRule) ID() string                 { return r.config.ID }
func (r *ImgAltRule) Name() string               { return r.config.Name }
func (r *ImgAltRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *ImgAltRule) Category() string          { return r.config.Category }
func (r *ImgAltRule) Description() string       { return r.config.Description }

func (r *ImgAltRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	htmlData, ok := file.AST.(map[string]interface{})
	if !ok {
		return issues
	}

	images, exists := htmlData["images"]
	if !exists {
		return issues
	}

	imageList, ok := images.([]map[string]string)
	if !ok {
		return issues
	}

	for _, img := range imageList {
		imgTag := img["tag"]
		alt, hasAlt := img["alt"]
		
		if !hasAlt || strings.TrimSpace(alt) == "" {
			lineNum := r.findLineNumber(file, imgTag)
			
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      0,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "img 태그에 alt 속성이 누락되었거나 비어있습니다",
				Description: "시각 장애인을 위한 대체 텍스트가 필요합니다",
				Suggestion:  "img 태그에 의미있는 alt 속성을 추가하세요",
				CodeSnippet: imgTag,
			})
		}
	}

	return issues
}

func (r *ImgAltRule) findLineNumber(file *parser.ParsedFile, tag string) int {
	for i, line := range file.Lines {
		if strings.Contains(line, tag) {
			return i + 1
		}
	}
	return 1
}

// AccessibilityRule 웹 접근성 검사
type AccessibilityRule struct {
	config config.RuleConfig
}

func NewAccessibilityRule(cfg config.RuleConfig) Rule {
	return &AccessibilityRule{config: cfg}
}

func (r *AccessibilityRule) ID() string                 { return r.config.ID }
func (r *AccessibilityRule) Name() string               { return r.config.Name }
func (r *AccessibilityRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *AccessibilityRule) Category() string          { return r.config.Category }
func (r *AccessibilityRule) Description() string       { return r.config.Description }

func (r *AccessibilityRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// 클릭 가능한 div 요소 검사 (onclick이 있는 div)
	clickableDivRegex := regexp.MustCompile(`<div[^>]*onclick[^>]*>`)
	matches := clickableDivRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range matches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        lineNum,
			Column:      getColumnFromPosition(file.Content, match[0]),
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "div 요소에 onclick이 사용되었습니다",
			Description: "키보드 접근성이 떨어지며 스크린 리더에서 인식하기 어렵습니다",
			Suggestion:  "button 요소를 사용하거나 적절한 ARIA 속성을 추가하세요",
			CodeSnippet: r.getCodeSnippet(file, lineNum),
		})
	}

	// aria-label 없는 버튼 검사
	buttonRegex := regexp.MustCompile(`<button[^>]*>`)
	buttonMatches := buttonRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range buttonMatches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		buttonText := r.getCodeSnippet(file, lineNum)
		
		// aria-label이 있는지 확인
		if !strings.Contains(buttonText, "aria-label") && !r.hasButtonText(buttonText) {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, match[0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "button 요소에 접근 가능한 텍스트가 없습니다",
				Description: "스크린 리더 사용자가 버튼의 목적을 알 수 없습니다",
				Suggestion:  "aria-label 속성이나 버튼 텍스트를 추가하세요",
				CodeSnippet: buttonText,
			})
		}
	}

	// form input 요소의 label 연결 검사
	inputRegex := regexp.MustCompile(`<input[^>]*>`)
	inputMatches := inputRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range inputMatches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		inputText := r.getCodeSnippet(file, lineNum)
		
		// aria-label 또는 aria-labelledby가 있는지 확인
		if !strings.Contains(inputText, "aria-label") && !strings.Contains(inputText, "aria-labelledby") {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, match[0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "input 요소에 레이블이 연결되지 않았습니다",
				Description: "사용자가 입력 필드의 목적을 알기 어렵습니다",
				Suggestion:  "label 요소를 사용하거나 aria-label 속성을 추가하세요",
				CodeSnippet: r.getCodeSnippet(file, lineNum),
			})
		}
	}

	return issues
}

func (r *AccessibilityRule) hasButtonText(buttonHTML string) bool {
	// 버튼 태그 사이의 텍스트 추출
	textRegex := regexp.MustCompile(`<button[^>]*>(.*?)</button>`)
	match := textRegex.FindStringSubmatch(buttonHTML)
	
	if len(match) > 1 {
		text := strings.TrimSpace(match[1])
		// HTML 태그 제거
		textWithoutTags := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
		return strings.TrimSpace(textWithoutTags) != ""
	}
	
	return false
}

func (r *AccessibilityRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// SEORule SEO 최적화 검사
type SEORule struct {
	config config.RuleConfig
}

func NewSEORule(cfg config.RuleConfig) Rule {
	return &SEORule{config: cfg}
}

func (r *SEORule) ID() string                 { return r.config.ID }
func (r *SEORule) Name() string               { return r.config.Name }
func (r *SEORule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *SEORule) Category() string          { return r.config.Category }
func (r *SEORule) Description() string       { return r.config.Description }

func (r *SEORule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// title 태그 검사
	if !r.hasTitle(file.Content) {
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        1,
			Column:      1,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "title 태그가 없습니다",
			Description: "페이지 제목은 SEO에 매우 중요합니다",
			Suggestion:  "<title> 태그를 head 영역에 추가하세요",
			CodeSnippet: "<title>페이지 제목</title>",
		})
	}

	// meta description 검사
	if !r.hasMetaDescription(file.Content) {
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        1,
			Column:      1,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "meta description이 없습니다",
			Description: "검색 결과에 표시될 페이지 설명이 필요합니다",
			Suggestion:  `<meta name="description" content="페이지 설명"> 태그를 추가하세요`,
			CodeSnippet: `<meta name="description" content="페이지 설명">`,
		})
	}

	// h1 태그 검사
	h1Count := r.countH1Tags(file.Content)
	if h1Count == 0 {
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        1,
			Column:      1,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "h1 태그가 없습니다",
			Description: "페이지의 주요 제목이 필요합니다",
			Suggestion:  "페이지의 주요 제목에 h1 태그를 사용하세요",
			CodeSnippet: "<h1>페이지 주제목</h1>",
		})
	} else if h1Count > 1 {
		h1Regex := regexp.MustCompile(`<h1[^>]*>`)
		matches := h1Regex.FindAllStringIndex(file.Content, -1)
		
		for i, match := range matches[1:] { // 첫 번째 h1은 제외
			lineNum := getLineNumberFromPosition(file.Content, match[0])
			
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, match[0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "h1 태그가 여러 개 사용되었습니다",
				Description: "페이지당 하나의 h1 태그만 사용하는 것이 좋습니다",
				Suggestion:  "추가 제목에는 h2, h3 등을 사용하세요",
				CodeSnippet: r.getCodeSnippet(file, lineNum),
			})
			
			if i >= 2 { // 최대 3개까지만 보고
				break
			}
		}
	}

	return issues
}

func (r *SEORule) hasTitle(content string) bool {
	titleRegex := regexp.MustCompile(`<title[^>]*>.*?</title>`)
	return titleRegex.MatchString(content)
}

func (r *SEORule) hasMetaDescription(content string) bool {
	metaDescRegex := regexp.MustCompile(`<meta[^>]*name\s*=\s*["']description["'][^>]*>`)
	return metaDescRegex.MatchString(content)
}

func (r *SEORule) countH1Tags(content string) int {
	h1Regex := regexp.MustCompile(`<h1[^>]*>`)
	matches := h1Regex.FindAllString(content, -1)
	return len(matches)
}

func (r *SEORule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}