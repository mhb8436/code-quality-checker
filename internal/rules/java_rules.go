package rules

import (
	"regexp"
	"strings"

	"code-quality-checker/internal/config"
	"code-quality-checker/internal/parser"
	"code-quality-checker/internal/types"
)

// TransactionalRule @Transactional 어노테이션 누락 검사
type TransactionalRule struct {
	config config.RuleConfig
}

func NewTransactionalRule(cfg config.RuleConfig) Rule {
	return &TransactionalRule{config: cfg}
}

func (r *TransactionalRule) ID() string                 { return r.config.ID }
func (r *TransactionalRule) Name() string               { return r.config.Name }
func (r *TransactionalRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *TransactionalRule) Category() string          { return r.config.Category }
func (r *TransactionalRule) Description() string       { return r.config.Description }

func (r *TransactionalRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	javaClass, ok := file.AST.(*parser.JavaClass)
	if !ok {
		return issues
	}

	// @Service 어노테이션이 있는지 확인
	hasServiceAnnotation := false
	for _, annotation := range javaClass.Annotations {
		if strings.Contains(annotation, "@Service") {
			hasServiceAnnotation = true
			break
		}
	}

	if !hasServiceAnnotation {
		return issues
	}

	// 데이터 변경 메소드 검사
	for _, method := range javaClass.Methods {
		if r.isDataChangeMethod(method.Name) && !r.hasTransactionalAnnotation(method.Annotations) {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        method.Line,
				Column:      method.Column,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "Service 클래스의 데이터 변경 메소드에 @Transactional 어노테이션이 누락되었습니다",
				Description: "트랜잭션 미적용 시 데이터 일관성 및 원자성 보장이 어렵습니다",
				Suggestion:  "@Transactional 어노테이션을 메소드에 추가하세요",
				CodeSnippet: r.getCodeSnippet(file, method.Line),
			})
		}
	}

	return issues
}

func (r *TransactionalRule) isDataChangeMethod(methodName string) bool {
	dataChangePatterns := []string{
		"insert", "update", "delete", "save", "modify", "remove", "create", "add", "set",
	}

	methodLower := strings.ToLower(methodName)
	for _, pattern := range dataChangePatterns {
		if strings.Contains(methodLower, pattern) {
			return true
		}
	}
	return false
}

func (r *TransactionalRule) hasTransactionalAnnotation(annotations []string) bool {
	for _, annotation := range annotations {
		if strings.Contains(annotation, "@Transactional") {
			return true
		}
	}
	return false
}

func (r *TransactionalRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// SystemOutRule System.out.println 사용 검사
type SystemOutRule struct {
	config config.RuleConfig
}

func NewSystemOutRule(cfg config.RuleConfig) Rule {
	return &SystemOutRule{config: cfg}
}

func (r *SystemOutRule) ID() string                 { return r.config.ID }
func (r *SystemOutRule) Name() string               { return r.config.Name }
func (r *SystemOutRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *SystemOutRule) Category() string          { return r.config.Category }
func (r *SystemOutRule) Description() string       { return r.config.Description }

func (r *SystemOutRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	systemOutRegex := regexp.MustCompile(`System\.out\.(print|println)`)
	matches := systemOutRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range matches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        lineNum,
			Column:      getColumnFromPosition(file.Content, match[0]),
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "System.out.println 사용이 발견되었습니다",
			Description: "프로덕션 환경에서 불필요한 정보 노출 위험이 있습니다",
			Suggestion:  "Logger를 사용하여 로깅하세요",
			CodeSnippet: r.getCodeSnippet(file, lineNum),
		})
	}

	return issues
}

func (r *SystemOutRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// LayerArchitectureRule 레이어 아키텍처 위반 검사
type LayerArchitectureRule struct {
	config config.RuleConfig
}

func NewLayerArchitectureRule(cfg config.RuleConfig) Rule {
	return &LayerArchitectureRule{config: cfg}
}

func (r *LayerArchitectureRule) ID() string                 { return r.config.ID }
func (r *LayerArchitectureRule) Name() string               { return r.config.Name }
func (r *LayerArchitectureRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *LayerArchitectureRule) Category() string          { return r.config.Category }
func (r *LayerArchitectureRule) Description() string       { return r.config.Description }

func (r *LayerArchitectureRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	javaClass, ok := file.AST.(*parser.JavaClass)
	if !ok {
		return issues
	}

	// Controller에서 DAO 직접 의존 검사
	if r.isController(javaClass) {
		issues = append(issues, r.checkControllerDAODependency(file, javaClass)...)
	}

	return issues
}

func (r *LayerArchitectureRule) isController(class *parser.JavaClass) bool {
	for _, annotation := range class.Annotations {
		if strings.Contains(annotation, "@Controller") || strings.Contains(annotation, "@RestController") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(class.Name), "controller")
}

func (r *LayerArchitectureRule) checkControllerDAODependency(file *parser.ParsedFile, class *parser.JavaClass) []types.Issue {
	var issues []types.Issue

	for _, field := range class.Fields {
		if r.isDAOField(field) {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        field.Line,
				Column:      0,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "Controller에서 DAO를 직접 의존하고 있습니다",
				Description: "레이어 아키텍처 위반으로 유지보수성이 저하됩니다",
				Suggestion:  "Service 레이어를 통해 데이터에 접근하세요",
				CodeSnippet: r.getCodeSnippet(file, field.Line),
			})
		}
	}

	return issues
}

func (r *LayerArchitectureRule) isDAOField(field parser.JavaField) bool {
	fieldTypeLower := strings.ToLower(field.Type)
	return strings.Contains(fieldTypeLower, "dao") || 
		   strings.Contains(fieldTypeLower, "repository") ||
		   strings.Contains(fieldTypeLower, "mapper")
}

func (r *LayerArchitectureRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// MagicNumberRule 매직 넘버 검사
type MagicNumberRule struct {
	config config.RuleConfig
}

func NewMagicNumberRule(cfg config.RuleConfig) Rule {
	return &MagicNumberRule{config: cfg}
}

func (r *MagicNumberRule) ID() string                 { return r.config.ID }
func (r *MagicNumberRule) Name() string               { return r.config.Name }
func (r *MagicNumberRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *MagicNumberRule) Category() string          { return r.config.Category }
func (r *MagicNumberRule) Description() string       { return r.config.Description }

func (r *MagicNumberRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// 매직 넘버 패턴 (정수 리터럴, 부동소수점 리터럴)
	magicNumberRegex := regexp.MustCompile(`\b((?:[1-9]\d{2,})|(?:\d+\.\d+))\b`)
	matches := magicNumberRegex.FindAllStringSubmatch(file.Content, -1)
	indices := magicNumberRegex.FindAllStringIndex(file.Content, -1)

	for i, match := range matches {
		if len(match) > 1 {
			number := match[1]
			
			// 제외할 숫자들 (0, 1, 2, 100 등 일반적인 숫자)
			if r.isExcludedNumber(number) {
				continue
			}

			lineNum := getLineNumberFromPosition(file.Content, indices[i][0])
			
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, indices[i][0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "매직 넘버가 발견되었습니다: " + number,
				Description: "하드코딩된 숫자는 코드 가독성을 저하시킵니다",
				Suggestion:  "의미있는 상수로 정의하세요",
				CodeSnippet: r.getCodeSnippet(file, lineNum),
			})
		}
	}

	return issues
}

func (r *MagicNumberRule) isExcludedNumber(number string) bool {
	excludedNumbers := []string{"0", "1", "2", "10", "100", "1000"}
	for _, excluded := range excludedNumbers {
		if number == excluded {
			return true
		}
	}
	return false
}

func (r *MagicNumberRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// MethodLengthRule 메소드 길이 검사
type MethodLengthRule struct {
	config config.RuleConfig
}

func NewMethodLengthRule(cfg config.RuleConfig) Rule {
	return &MethodLengthRule{config: cfg}
}

func (r *MethodLengthRule) ID() string                 { return r.config.ID }
func (r *MethodLengthRule) Name() string               { return r.config.Name }
func (r *MethodLengthRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *MethodLengthRule) Category() string          { return r.config.Category }
func (r *MethodLengthRule) Description() string       { return r.config.Description }

func (r *MethodLengthRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	javaClass, ok := file.AST.(*parser.JavaClass)
	if !ok {
		return issues
	}

	for _, method := range javaClass.Methods {
		methodLength := r.calculateMethodLength(file, method)
		
		if methodLength > 50 { // Java 메소드 길이 임계값
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        method.Line,
				Column:      method.Column,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "메소드가 너무 깁니다 (" + method.Name + ": " + intToString(methodLength) + " 라인)",
				Description: "긴 메소드는 가독성과 유지보수성을 저하시킵니다",
				Suggestion:  "메소드를 더 작은 단위로 분할하세요",
				CodeSnippet: r.getCodeSnippet(file, method.Line),
			})
		}
	}

	return issues
}

func (r *MethodLengthRule) calculateMethodLength(file *parser.ParsedFile, method parser.JavaMethod) int {
	// 간단한 방법: 메소드 시작부터 다음 메소드까지의 라인 수 계산
	// 실제로는 더 정교한 파싱이 필요하지만, 여기서는 근사치 사용
	methodBodyRegex := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(method.Name) + `\s*\([^)]*\)\s*\{.*?\}`)
	match := methodBodyRegex.FindString(file.Content)
	
	if match == "" {
		return 0
	}
	
	return strings.Count(match, "\n")
}

func (r *MethodLengthRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// 헬퍼 함수들
func getLineNumberFromPosition(content string, pos int) int {
	return strings.Count(content[:pos], "\n") + 1
}

func getColumnFromPosition(content string, pos int) int {
	lines := strings.Split(content[:pos], "\n")
	if len(lines) == 0 {
		return 1
	}
	return len(lines[len(lines)-1]) + 1
}

func intToString(i int) string {
	return strings.Trim(strings.Join(strings.Fields(string(rune(i))), ""), "[]")
}

// 나머지 규칙들을 위한 스텁
func NewExceptionHandlingRule(cfg config.RuleConfig) Rule {
	return &SystemOutRule{config: cfg} // 임시로 SystemOutRule 재사용
}

func NewInputValidationRule(cfg config.RuleConfig) Rule {
	return &SystemOutRule{config: cfg} // 임시로 SystemOutRule 재사용
}

func NewCyclomaticComplexityRule(cfg config.RuleConfig) Rule {
	return &SystemOutRule{config: cfg} // 임시로 SystemOutRule 재사용
}

func NewDuplicateCodeRule(cfg config.RuleConfig) Rule {
	return &SystemOutRule{config: cfg} // 임시로 SystemOutRule 재사용
}

func NewCodingConventionRule(cfg config.RuleConfig) Rule {
	return &SystemOutRule{config: cfg} // 임시로 SystemOutRule 재사용
}