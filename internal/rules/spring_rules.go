package rules

import (
	"regexp"
	"strings"

	"code-quality-checker/internal/config"
	"code-quality-checker/internal/parser"
	"code-quality-checker/internal/types"
)

// SpringValidationRule @Valid 어노테이션 누락 검사
type SpringValidationRule struct {
	config config.RuleConfig
}

func NewSpringValidationRule(cfg config.RuleConfig) Rule {
	return &SpringValidationRule{config: cfg}
}

func (r *SpringValidationRule) ID() string                 { return r.config.ID }
func (r *SpringValidationRule) Name() string               { return r.config.Name }
func (r *SpringValidationRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *SpringValidationRule) Category() string          { return r.config.Category }
func (r *SpringValidationRule) Description() string       { return r.config.Description }

func (r *SpringValidationRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// Controller 클래스인지 확인
	if !r.isController(file.Content) {
		return issues
	}

	// @RequestBody 패턴 찾기
	requestBodyRegex := regexp.MustCompile(`@RequestBody\s+(\w+\s+\w+)`)
	matches := requestBodyRegex.FindAllStringSubmatch(file.Content, -1)
	indices := requestBodyRegex.FindAllStringIndex(file.Content, -1)

	for i, match := range matches {
		if len(match) > 1 {
			lineNum := getLineNumberFromPosition(file.Content, indices[i][0])
			
			// 해당 라인 주변에 @Valid가 있는지 확인
			if !r.hasValidAnnotation(file.Content, lineNum) {
				issues = append(issues, types.Issue{
					RuleID:      r.ID(),
					File:        file.Path,
					Line:        lineNum,
					Column:      getColumnFromPosition(file.Content, indices[i][0]),
					Severity:    r.Severity(),
					Category:    r.Category(),
					Message:     "@RequestBody 매개변수에 @Valid 어노테이션이 누락되었습니다",
					Description: "입력값 검증이 없으면 보안 취약점이 발생할 수 있습니다",
					Suggestion:  "@Valid 어노테이션을 추가하여 입력값을 검증하세요",
					CodeSnippet: strings.TrimSpace(r.getCodeSnippet(file, lineNum)),
				})
			}
		}
	}

	return issues
}

func (r *SpringValidationRule) isController(content string) bool {
	controllerPatterns := []string{
		"@Controller",
		"@RestController",
	}
	
	for _, pattern := range controllerPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

func (r *SpringValidationRule) hasValidAnnotation(content string, lineNum int) bool {
	lines := strings.Split(content, "\n")
	start := max(0, lineNum-2)
	end := min(len(lines), lineNum+2)
	
	for i := start; i < end; i++ {
		if strings.Contains(lines[i], "@Valid") {
			return true
		}
	}
	return false
}

func (r *SpringValidationRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return file.Lines[line-1]
}

// SpringTransactionalRule @Transactional 관련 검사
type SpringTransactionalRule struct {
	config config.RuleConfig
}

func NewSpringTransactionalRule(cfg config.RuleConfig) Rule {
	return &SpringTransactionalRule{config: cfg}
}

func (r *SpringTransactionalRule) ID() string                 { return r.config.ID }
func (r *SpringTransactionalRule) Name() string               { return r.config.Name }
func (r *SpringTransactionalRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *SpringTransactionalRule) Category() string          { return r.config.Category }
func (r *SpringTransactionalRule) Description() string       { return r.config.Description }

func (r *SpringTransactionalRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// private 메소드에 @Transactional 사용 검사
	privateTransactionalRegex := regexp.MustCompile(`@Transactional[^\n]*\n[^\n]*private\s+\w+\s+(\w+)\s*\(`)
	matches := privateTransactionalRegex.FindAllStringSubmatch(file.Content, -1)
	indices := privateTransactionalRegex.FindAllStringIndex(file.Content, -1)

	for i, match := range matches {
		if len(match) > 1 {
			lineNum := getLineNumberFromPosition(file.Content, indices[i][0])
			
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, indices[i][0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "private 메소드에 @Transactional 어노테이션이 사용되었습니다",
				Description: "private 메소드는 프록시가 작동하지 않아 트랜잭션이 적용되지 않습니다",
				Suggestion:  "메소드를 public으로 변경하거나 클래스 레벨에서 @Transactional을 사용하세요",
				CodeSnippet: strings.TrimSpace(r.getCodeSnippet(file, lineNum)),
			})
		}
	}

	// rollbackFor 누락 검사
	transactionalRegex := regexp.MustCompile(`@Transactional`)
	rollbackMatches := transactionalRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range rollbackMatches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		line := r.getCodeSnippet(file, lineNum)
		
		// rollbackFor가 있는지 확인
		if !strings.Contains(line, "rollbackFor") && r.hasThrowsException(file.Content, lineNum) {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, match[0]),
				Severity:    config.SeverityMedium,
				Category:    "reliability",
				Message:     "@Transactional에 rollbackFor 설정이 누락되었습니다",
				Description: "체크드 예외 발생 시 롤백되지 않을 수 있습니다",
				Suggestion:  "@Transactional(rollbackFor = Exception.class)를 사용하세요",
				CodeSnippet: strings.TrimSpace(line),
			})
		}
	}

	return issues
}

func (r *SpringTransactionalRule) hasThrowsException(content string, lineNum int) bool {
	// 해당 라인 근처에 throws Exception이 있는지 확인
	lines := strings.Split(content, "\n")
	start := max(0, lineNum-1)
	end := min(len(lines), lineNum+5)
	
	for i := start; i < end; i++ {
		if strings.Contains(lines[i], "throws Exception") {
			return true
		}
	}
	return false
}

func (r *SpringTransactionalRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return file.Lines[line-1]
}

// SpringSecurityRule Spring Security 어노테이션 검사
type SpringSecurityRule struct {
	config config.RuleConfig
}

func NewSpringSecurityRule(cfg config.RuleConfig) Rule {
	return &SpringSecurityRule{config: cfg}
}

func (r *SpringSecurityRule) ID() string                 { return r.config.ID }
func (r *SpringSecurityRule) Name() string               { return r.config.Name }
func (r *SpringSecurityRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *SpringSecurityRule) Category() string          { return r.config.Category }
func (r *SpringSecurityRule) Description() string       { return r.config.Description }

func (r *SpringSecurityRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// Controller 클래스인지 확인
	if !r.isController(file.Content) {
		return issues
	}

	// 민감한 메소드에 보안 어노테이션 누락 검사
	sensitiveMethodRegex := regexp.MustCompile(`public\s+\w+\s+(delete|remove|admin|update|modify|create|add)\w*\s*\([^)]*\)\s*(?:throws[^{]*)?\{`)
	matches := sensitiveMethodRegex.FindAllStringSubmatch(file.Content, -1)
	indices := sensitiveMethodRegex.FindAllStringIndex(file.Content, -1)

	for i, match := range matches {
		if len(match) > 1 {
			lineNum := getLineNumberFromPosition(file.Content, indices[i][0])
			
			// 해당 메소드에 보안 어노테이션이 있는지 확인
			if !r.hasSecurityAnnotation(file.Content, lineNum) {
				issues = append(issues, types.Issue{
					RuleID:      r.ID(),
					File:        file.Path,
					Line:        lineNum,
					Column:      getColumnFromPosition(file.Content, indices[i][0]),
					Severity:    r.Severity(),
					Category:    r.Category(),
					Message:     "민감한 메소드에 보안 어노테이션이 누락되었습니다: " + match[1],
					Description: "삭제, 수정, 관리자 기능에는 적절한 권한 검사가 필요합니다",
					Suggestion:  "@PreAuthorize(\"hasRole('ADMIN')\") 등의 보안 어노테이션을 추가하세요",
					CodeSnippet: strings.TrimSpace(r.getCodeSnippet(file, lineNum)),
				})
			}
		}
	}

	// @Secured 사용 시 @PreAuthorize 권장
	securedAnnotationRegex := regexp.MustCompile(`@Secured`)
	securedMatches := securedAnnotationRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range securedMatches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        lineNum,
			Column:      getColumnFromPosition(file.Content, match[0]),
			Severity:    config.SeverityMedium,
			Category:    "best-practices",
			Message:     "@Secured 대신 @PreAuthorize 사용을 권장합니다",
			Description: "@PreAuthorize는 SpEL을 지원하여 더 유연한 보안 설정이 가능합니다",
			Suggestion:  "@PreAuthorize(\"hasRole('ROLE_NAME')\")로 변경하세요",
			CodeSnippet: strings.TrimSpace(r.getCodeSnippet(file, lineNum)),
		})
	}

	return issues
}

func (r *SpringSecurityRule) isController(content string) bool {
	controllerPatterns := []string{
		"@Controller",
		"@RestController",
	}
	
	for _, pattern := range controllerPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

func (r *SpringSecurityRule) hasSecurityAnnotation(content string, lineNum int) bool {
	lines := strings.Split(content, "\n")
	start := max(0, lineNum-5)
	end := min(len(lines), lineNum)
	
	securityAnnotations := []string{
		"@PreAuthorize",
		"@PostAuthorize",
		"@Secured",
		"@RolesAllowed",
	}
	
	for i := start; i < end; i++ {
		for _, annotation := range securityAnnotations {
			if strings.Contains(lines[i], annotation) {
				return true
			}
		}
	}
	return false
}

func (r *SpringSecurityRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return file.Lines[line-1]
}

// SpringDependencyInjectionRule 의존성 주입 검사
type SpringDependencyInjectionRule struct {
	config config.RuleConfig
}

func NewSpringDependencyInjectionRule(cfg config.RuleConfig) Rule {
	return &SpringDependencyInjectionRule{config: cfg}
}

func (r *SpringDependencyInjectionRule) ID() string                 { return r.config.ID }
func (r *SpringDependencyInjectionRule) Name() string               { return r.config.Name }
func (r *SpringDependencyInjectionRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *SpringDependencyInjectionRule) Category() string          { return r.config.Category }
func (r *SpringDependencyInjectionRule) Description() string       { return r.config.Description }

func (r *SpringDependencyInjectionRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// @Autowired 필드 주입 사용 검사
	autowiredFieldRegex := regexp.MustCompile(`@Autowired\s+private\s+\w+\s+(\w+);`)
	matches := autowiredFieldRegex.FindAllStringSubmatch(file.Content, -1)
	indices := autowiredFieldRegex.FindAllStringIndex(file.Content, -1)

	for i, match := range matches {
		if len(match) > 1 {
			lineNum := getLineNumberFromPosition(file.Content, indices[i][0])
			
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, indices[i][0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "필드 주입 대신 생성자 주입을 사용하세요: " + match[1],
				Description: "생성자 주입은 불변성을 보장하고 테스트하기 더 쉽습니다",
				Suggestion:  "final 필드와 생성자를 사용하거나 @RequiredArgsConstructor를 활용하세요",
				CodeSnippet: strings.TrimSpace(r.getCodeSnippet(file, lineNum)),
			})
		}
	}

	return issues
}

func (r *SpringDependencyInjectionRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return file.Lines[line-1]
}

// SpringExceptionHandlingRule 예외 처리 검사
type SpringExceptionHandlingRule struct {
	config config.RuleConfig
}

func NewSpringExceptionHandlingRule(cfg config.RuleConfig) Rule {
	return &SpringExceptionHandlingRule{config: cfg}
}

func (r *SpringExceptionHandlingRule) ID() string                 { return r.config.ID }
func (r *SpringExceptionHandlingRule) Name() string               { return r.config.Name }
func (r *SpringExceptionHandlingRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *SpringExceptionHandlingRule) Category() string          { return r.config.Category }
func (r *SpringExceptionHandlingRule) Description() string       { return r.config.Description }

func (r *SpringExceptionHandlingRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// 프로젝트에 @ControllerAdvice가 있는지 확인
	hasControllerAdvice := strings.Contains(file.Content, "@ControllerAdvice") || 
						  strings.Contains(file.Content, "@RestControllerAdvice")

	// Controller 클래스이면서 전역 예외 처리기가 없는 경우
	if r.isController(file.Content) && !hasControllerAdvice {
		// try-catch 없이 throws Exception만 있는 메소드 검사
		throwsExceptionRegex := regexp.MustCompile(`public\s+\w+\s+\w+\s*\([^)]*\)\s+throws\s+Exception`)
		matches := throwsExceptionRegex.FindAllStringIndex(file.Content, -1)

		if len(matches) > 0 {
			lineNum := getLineNumberFromPosition(file.Content, matches[0][0])
			
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, matches[0][0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "전역 예외 처리기(@ControllerAdvice)가 없습니다",
				Description: "일관된 예외 처리를 위해 전역 예외 처리기를 구현하세요",
				Suggestion:  "@ControllerAdvice를 사용한 전역 예외 처리 클래스를 생성하세요",
				CodeSnippet: strings.TrimSpace(r.getCodeSnippet(file, lineNum)),
			})
		}
	}

	return issues
}

func (r *SpringExceptionHandlingRule) isController(content string) bool {
	controllerPatterns := []string{
		"@Controller",
		"@RestController",
	}
	
	for _, pattern := range controllerPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

func (r *SpringExceptionHandlingRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return file.Lines[line-1]
}

// 헬퍼 함수
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}