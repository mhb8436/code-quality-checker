package rules

import (
	"fmt"
	"regexp"
	"strconv"
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

	// 데이터 변경 메소드 검사 - 복잡한 트랜잭션이 필요한 경우만 체크
	for _, method := range javaClass.Methods {
		if r.isDataChangeMethod(method.Name) && !r.hasTransactionalAnnotation(method.Annotations) {
			// 메소드 복잡도 분석
			complexity := r.analyzeMethodComplexity(file, method)
			
			if complexity.requiresTransaction {
				issues = append(issues, types.Issue{
					RuleID:      r.ID(),
					File:        file.Path,
					Line:        method.Line,
					Column:      method.Column,
					Severity:    r.Severity(),
					Category:    r.Category(),
					Message:     r.generateTransactionalMessage(method.Name, complexity),
					Description: "복잡한 데이터 변경 작업에는 트랜잭션이 필요합니다",
					Suggestion:  "@Transactional 어노테이션을 메소드에 추가하세요",
					CodeSnippet: r.getCodeSnippet(file, method.Line),
				})
			}
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

// MethodComplexity 메소드 복잡도 분석 결과
type MethodComplexity struct {
	requiresTransaction bool
	reason             string
	repositoryCalls    int
	conditionalLogic   bool
	multipleOperations bool
	externalCalls      bool
}

// analyzeMethodComplexity 메소드의 트랜잭션 필요성 분석
func (r *TransactionalRule) analyzeMethodComplexity(file *parser.ParsedFile, method parser.JavaMethod) MethodComplexity {
	methodBody := r.extractMethodBody(file, method)
	
	complexity := MethodComplexity{
		requiresTransaction: false,
		reason:             "",
	}
	
	// 1. Repository/DAO 호출 횟수 체크
	repositoryPatterns := []string{
		`\w+Repository\.\w+\(`,
		`\w+DAO\.\w+\(`,
		`\w+Mapper\.\w+\(`,
	}
	
	for _, pattern := range repositoryPatterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllString(methodBody, -1)
		complexity.repositoryCalls += len(matches)
	}
	
	// 2. 조건부 로직 검사 (if/else와 데이터 변경이 함께)
	if r.hasConditionalDataOperations(methodBody) {
		complexity.conditionalLogic = true
	}
	
	// 3. 여러 종류의 데이터 작업 검사
	if r.hasMultipleDataOperations(methodBody) {
		complexity.multipleOperations = true
	}
	
	// 4. 외부 시스템 호출 검사
	if r.hasExternalSystemCalls(methodBody) {
		complexity.externalCalls = true
	}
	
	// 트랜잭션 필요성 판단
	complexity.requiresTransaction, complexity.reason = r.determineTransactionNeed(complexity)
	
	return complexity
}

// extractMethodBody 메소드 본문 추출
func (r *TransactionalRule) extractMethodBody(file *parser.ParsedFile, method parser.JavaMethod) string {
	// 메소드 시작 위치 찾기
	methodPattern := regexp.QuoteMeta(method.Name) + `\s*\([^)]*\)\s*\{`
	methodRegex := regexp.MustCompile(methodPattern)
	
	match := methodRegex.FindStringIndex(file.Content)
	if match == nil {
		return ""
	}
	
	// 메소드 본문 추출 (중괄호 매칭)
	start := match[1] - 1 // '{' 위치
	braceCount := 1
	i := start + 1
	
	content := []rune(file.Content)
	for i < len(content) && braceCount > 0 {
		if content[i] == '{' {
			braceCount++
		} else if content[i] == '}' {
			braceCount--
		}
		i++
	}
	
	if braceCount == 0 {
		return string(content[start:i])
	}
	
	return ""
}

// hasConditionalDataOperations 조건부 데이터 작업 검사
func (r *TransactionalRule) hasConditionalDataOperations(methodBody string) bool {
	// if문과 데이터 변경 작업이 함께 있는지 검사
	ifPattern := `if\s*\([^)]+\)\s*\{[^}]*(?:save|update|delete|insert|remove)\([^}]*\}`
	matched, _ := regexp.MatchString(ifPattern, methodBody)
	return matched
}

// hasMultipleDataOperations 여러 종류의 데이터 작업 검사
func (r *TransactionalRule) hasMultipleDataOperations(methodBody string) bool {
	operations := []string{"save", "update", "delete", "insert", "remove"}
	foundOperations := make(map[string]bool)
	
	for _, op := range operations {
		pattern := `\w*` + op + `\w*\(`
		matched, _ := regexp.MatchString(`(?i)`+pattern, methodBody)
		if matched {
			foundOperations[op] = true
		}
	}
	
	// 2가지 이상의 다른 작업이 있으면 복잡한 트랜잭션
	return len(foundOperations) >= 2
}

// hasExternalSystemCalls 외부 시스템 호출 검사
func (r *TransactionalRule) hasExternalSystemCalls(methodBody string) bool {
	externalPatterns := []string{
		`restTemplate\.\w+\(`,
		`webClient\.\w+\(`,
		`\w*Client\.\w+\(`,
		`\w*Service\.\w+\(.*http`,
		`@FeignClient`,
		`kafka\w*\.\w+\(`,
		`jms\w*\.\w+\(`,
	}
	
	for _, pattern := range externalPatterns {
		matched, _ := regexp.MatchString(`(?i)`+pattern, methodBody)
		if matched {
			return true
		}
	}
	
	return false
}

// determineTransactionNeed 트랜잭션 필요성 최종 판단
func (r *TransactionalRule) determineTransactionNeed(complexity MethodComplexity) (bool, string) {
	reasons := []string{}
	
	// 2개 이상의 Repository 호출
	if complexity.repositoryCalls >= 2 {
		reasons = append(reasons, fmt.Sprintf("여러 테이블 작업(%d개 Repository 호출)", complexity.repositoryCalls))
	}
	
	// 조건부 데이터 작업
	if complexity.conditionalLogic {
		reasons = append(reasons, "조건부 데이터 변경 로직")
	}
	
	// 여러 종류의 데이터 작업
	if complexity.multipleOperations {
		reasons = append(reasons, "복합 데이터 작업(생성/수정/삭제)")
	}
	
	// 외부 시스템 호출과 DB 작업이 함께
	if complexity.externalCalls && complexity.repositoryCalls > 0 {
		reasons = append(reasons, "외부 시스템 연동과 DB 작업")
	}
	
	requiresTransaction := len(reasons) > 0
	reason := strings.Join(reasons, ", ")
	
	return requiresTransaction, reason
}

// generateTransactionalMessage 트랜잭션 누락 메시지 생성
func (r *TransactionalRule) generateTransactionalMessage(methodName string, complexity MethodComplexity) string {
	return fmt.Sprintf("메소드 '%s'에 @Transactional이 필요합니다: %s", methodName, complexity.reason)
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

	// 설정에서 임계값 가져오기 (기본값: 100)
	maxLines := r.getMaxLines()

	for _, method := range javaClass.Methods {
		methodLength := r.calculateMethodLength(file, method)
		
		if methodLength > maxLines {
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        method.Line,
				Column:      method.Column,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "메소드가 너무 깁니다 (" + method.Name + ": " + intToString(methodLength) + " 라인, 임계값: " + intToString(maxLines) + ")",
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

func (r *MethodLengthRule) getMaxLines() int {
	// 설정에서 max_lines 값 가져오기
	if maxLinesStr, exists := r.config.Custom["max_lines"]; exists {
		if maxLines, err := strconv.Atoi(maxLinesStr); err == nil && maxLines > 0 {
			return maxLines
		}
	}
	// 기본값: 100라인 (업계 표준)
	return 100
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
	return strconv.Itoa(i)
}

// ExceptionHandlingRule 예외 처리 검사
type ExceptionHandlingRule struct {
	config config.RuleConfig
}

func NewExceptionHandlingRule(cfg config.RuleConfig) Rule {
	return &ExceptionHandlingRule{config: cfg}
}

func (r *ExceptionHandlingRule) ID() string                 { return r.config.ID }
func (r *ExceptionHandlingRule) Name() string               { return r.config.Name }
func (r *ExceptionHandlingRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *ExceptionHandlingRule) Category() string          { return r.config.Category }
func (r *ExceptionHandlingRule) Description() string       { return r.config.Description }

func (r *ExceptionHandlingRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// printStackTrace 사용 검사
	printStackTraceRegex := regexp.MustCompile(`\.printStackTrace\(\)`)
	matches := printStackTraceRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range matches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        lineNum,
			Column:      getColumnFromPosition(file.Content, match[0]),
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "printStackTrace() 사용이 발견되었습니다",
			Description: "예외 스택트레이스가 콘솔에 노출되어 보안 위험이 있습니다",
			Suggestion:  "Logger를 사용하여 적절한 로깅을 하세요",
			CodeSnippet: r.getCodeSnippet(file, lineNum),
		})
	}

	// throw new Exception() without proper handling 검사
	throwRegex := regexp.MustCompile(`throw\s+new\s+Exception\s*\([^)]*\)`)
	throwMatches := throwRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range throwMatches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        lineNum,
			Column:      getColumnFromPosition(file.Content, match[0]),
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "일반적인 Exception 타입을 사용하고 있습니다",
			Description: "구체적인 예외 타입을 사용하는 것이 좋습니다",
			Suggestion:  "구체적인 예외 클래스(BusinessException 등)를 정의하여 사용하세요",
			CodeSnippet: r.getCodeSnippet(file, lineNum),
		})
	}

	// Controller에 @ControllerAdvice 없는 경우 검사
	javaClass, ok := file.AST.(*parser.JavaClass)
	if ok && r.isController(javaClass) && !r.hasGlobalExceptionHandler(file.Content) {
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        1,
			Column:      1,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "전역 예외 처리기(@ControllerAdvice)가 없습니다",
			Description: "일관된 예외 처리를 위해 전역 예외 처리기가 필요합니다",
			Suggestion:  "@ControllerAdvice 클래스를 생성하여 전역 예외 처리를 구현하세요",
			CodeSnippet: "",
		})
	}

	return issues
}

func (r *ExceptionHandlingRule) isController(class *parser.JavaClass) bool {
	for _, annotation := range class.Annotations {
		if strings.Contains(annotation, "@Controller") || strings.Contains(annotation, "@RestController") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(class.Name), "controller")
}

func (r *ExceptionHandlingRule) hasGlobalExceptionHandler(content string) bool {
	return strings.Contains(content, "@ControllerAdvice") || strings.Contains(content, "@RestControllerAdvice")
}

func (r *ExceptionHandlingRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// InputValidationRule 입력 검증 검사
type InputValidationRule struct {
	config config.RuleConfig
}

func NewInputValidationRule(cfg config.RuleConfig) Rule {
	return &InputValidationRule{config: cfg}
}

func (r *InputValidationRule) ID() string                 { return r.config.ID }
func (r *InputValidationRule) Name() string               { return r.config.Name }
func (r *InputValidationRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *InputValidationRule) Category() string          { return r.config.Category }
func (r *InputValidationRule) Description() string       { return r.config.Description }

func (r *InputValidationRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	javaClass, ok := file.AST.(*parser.JavaClass)
	if !ok {
		return issues
	}

	// Controller 클래스인지 확인
	if !r.isController(javaClass) {
		return issues
	}

	// BenefitValidation 커스텀 검증 로직 사용 검사
	benefitValidationRegex := regexp.MustCompile(`BenefitValidation\.(isEmpty|isNull|isValid)`)
	matches := benefitValidationRegex.FindAllStringIndex(file.Content, -1)

	for _, match := range matches {
		lineNum := getLineNumberFromPosition(file.Content, match[0])
		issues = append(issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        lineNum,
			Column:      getColumnFromPosition(file.Content, match[0]),
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "커스텀 검증 로직 대신 Bean Validation 표준을 사용하세요",
			Description: "표준 검증 미적용 시 SQL인젝션, XSS 등 보안 취약점 위험이 증가합니다",
			Suggestion:  "@Valid, @NotNull, @Size 등 Bean Validation 어노테이션을 사용하세요",
			CodeSnippet: r.getCodeSnippet(file, lineNum),
		})
	}

	// @Valid 어노테이션 누락 검사
	for _, method := range javaClass.Methods {
		if r.isControllerMethod(method) && !r.hasValidAnnotation(method.Parameters, file.Content) {
			// RequestBody가 있는지 확인
			if r.hasRequestBodyParameter(method.Parameters, file.Content) {
				issues = append(issues, types.Issue{
					RuleID:      r.ID(),
					File:        file.Path,
					Line:        method.Line,
					Column:      method.Column,
					Severity:    r.Severity(),
					Category:    r.Category(),
					Message:     "@RequestBody 파라미터에 @Valid 어노테이션이 누락되었습니다",
					Description: "입력 검증이 누락되어 잘못된 데이터가 처리될 수 있습니다",
					Suggestion:  "@RequestBody @Valid 를 사용하여 자동 검증을 적용하세요",
					CodeSnippet: r.getCodeSnippet(file, method.Line),
				})
			}
		}
	}

	return issues
}

func (r *InputValidationRule) isController(class *parser.JavaClass) bool {
	for _, annotation := range class.Annotations {
		if strings.Contains(annotation, "@Controller") || strings.Contains(annotation, "@RestController") {
			return true
		}
	}
	return strings.Contains(strings.ToLower(class.Name), "controller")
}

func (r *InputValidationRule) isControllerMethod(method parser.JavaMethod) bool {
	for _, annotation := range method.Annotations {
		if strings.Contains(annotation, "@RequestMapping") ||
			strings.Contains(annotation, "@GetMapping") ||
			strings.Contains(annotation, "@PostMapping") ||
			strings.Contains(annotation, "@PutMapping") ||
			strings.Contains(annotation, "@DeleteMapping") {
			return true
		}
	}
	return false
}

func (r *InputValidationRule) hasValidAnnotation(parameters []string, content string) bool {
	for _, param := range parameters {
		// 파라미터 주변에서 @Valid 찾기
		paramIndex := strings.Index(content, param)
		if paramIndex != -1 {
			// 파라미터 앞 100자 정도에서 @Valid 찾기
			startIndex := paramIndex - 100
			if startIndex < 0 {
				startIndex = 0
			}
			context := content[startIndex:paramIndex]
			if strings.Contains(context, "@Valid") {
				return true
			}
		}
	}
	return false
}

func (r *InputValidationRule) hasRequestBodyParameter(parameters []string, content string) bool {
	for _, param := range parameters {
		paramIndex := strings.Index(content, param)
		if paramIndex != -1 {
			startIndex := paramIndex - 100
			if startIndex < 0 {
				startIndex = 0
			}
			context := content[startIndex:paramIndex]
			if strings.Contains(context, "@RequestBody") {
				return true
			}
		}
	}
	return false
}

func (r *InputValidationRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// CyclomaticComplexityRule 순환 복잡도 검사
type CyclomaticComplexityRule struct {
	config config.RuleConfig
}

func NewCyclomaticComplexityRule(cfg config.RuleConfig) Rule {
	return &CyclomaticComplexityRule{config: cfg}
}

func (r *CyclomaticComplexityRule) ID() string                 { return r.config.ID }
func (r *CyclomaticComplexityRule) Name() string               { return r.config.Name }
func (r *CyclomaticComplexityRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *CyclomaticComplexityRule) Category() string          { return r.config.Category }
func (r *CyclomaticComplexityRule) Description() string       { return r.config.Description }

func (r *CyclomaticComplexityRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	javaClass, ok := file.AST.(*parser.JavaClass)
	if !ok {
		return issues
	}

	for _, method := range javaClass.Methods {
		complexity := r.calculateComplexity(file, method)
		
		if complexity > 10 { // 순환 복잡도 임계값
			issues = append(issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        method.Line,
				Column:      method.Column,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     fmt.Sprintf("메소드 '%s'의 순환 복잡도가 너무 높습니다 (복잡도: %d)", method.Name, complexity),
				Description: "높은 순환 복잡도는 코드 이해도와 테스트 어려움을 증가시킵니다",
				Suggestion:  "메소드를 더 작은 단위로 분할하여 복잡도를 낮추세요",
				CodeSnippet: r.getCodeSnippet(file, method.Line),
			})
		}
	}

	return issues
}

func (r *CyclomaticComplexityRule) calculateComplexity(file *parser.ParsedFile, method parser.JavaMethod) int {
	// 메소드 본문 추출
	methodBody := r.extractMethodBody(file, method)
	if methodBody == "" {
		return 1 // 기본 복잡도
	}

	complexity := 1 // 기본 경로 1개

	// 분기문 패턴들
	branchPatterns := []string{
		`\bif\s*\(`,          // if 문
		`\belse\s+if\s*\(`,   // else if 문  
		`\belse\b`,           // else 문
		`\bwhile\s*\(`,       // while 문
		`\bfor\s*\(`,         // for 문
		`\bdo\s*\{`,          // do-while 문
		`\bswitch\s*\(`,      // switch 문
		`\bcase\s+`,          // case 문
		`\bcatch\s*\(`,       // catch 문
		`\?\s*[^:]+\s*:`,     // 삼항연산자
		`\&\&`,               // 논리 AND
		`\|\|`,               // 논리 OR
	}

	for _, pattern := range branchPatterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllString(methodBody, -1)
		complexity += len(matches)
	}

	return complexity
}

func (r *CyclomaticComplexityRule) extractMethodBody(file *parser.ParsedFile, method parser.JavaMethod) string {
	// 메소드 시작 위치 찾기
	methodPattern := regexp.QuoteMeta(method.Name) + `\s*\([^)]*\)\s*\{`
	methodRegex := regexp.MustCompile(methodPattern)
	
	match := methodRegex.FindStringIndex(file.Content)
	if match == nil {
		return ""
	}

	// 메소드 본문 추출 (중괄호 매칭)
	start := match[1] - 1 // '{' 위치
	braceCount := 1
	i := start + 1

	content := []rune(file.Content)
	for i < len(content) && braceCount > 0 {
		if content[i] == '{' {
			braceCount++
		} else if content[i] == '}' {
			braceCount--
		}
		i++
	}

	if braceCount == 0 {
		return string(content[start:i])
	}

	return ""
}

func (r *CyclomaticComplexityRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// DuplicateCodeRule 중복 코드 검사
type DuplicateCodeRule struct {
	config config.RuleConfig
}

func NewDuplicateCodeRule(cfg config.RuleConfig) Rule {
	return &DuplicateCodeRule{config: cfg}
}

func (r *DuplicateCodeRule) ID() string                 { return r.config.ID }
func (r *DuplicateCodeRule) Name() string               { return r.config.Name }
func (r *DuplicateCodeRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *DuplicateCodeRule) Category() string          { return r.config.Category }
func (r *DuplicateCodeRule) Description() string       { return r.config.Description }

func (r *DuplicateCodeRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	// 공통 패턴들 검사
	duplicatePatterns := []struct {
		pattern     string
		description string
		suggestion  string
	}{
		{
			pattern:     `responseBody\.put\(.*?\);`,
			description: "API 응답 생성 패턴이 중복되고 있습니다",
			suggestion:  "공통 응답 클래스(ApiResponse)를 만들어 사용하세요",
		},
		{
			pattern:     `cdService\.selectCdList\([^)]+\)`,
			description: "코드 목록 조회가 반복되고 있습니다",
			suggestion:  "캐싱을 적용하거나 공통 메소드로 추출하세요",
		},
		{
			pattern:     `if\s*\([^)]*==\s*null[^)]*\)\s*\{[^}]*throw[^}]*\}`,
			description: "null 체크 후 예외 발생 패턴이 중복됩니다",
			suggestion:  "공통 검증 메소드를 만들어 사용하세요",
		},
		{
			pattern:     `logger\.(info|debug|error)\([^)]*\);\s*return`,
			description: "로깅 후 return 패턴이 반복됩니다",
			suggestion:  "공통 로깅 유틸리티를 만들어 사용하세요",
		},
	}

	for _, dp := range duplicatePatterns {
		regex := regexp.MustCompile(dp.pattern)
		matches := regex.FindAllStringIndex(file.Content, -1)

		if len(matches) >= 3 { // 3번 이상 반복되면 중복으로 간주
			for _, match := range matches {
				lineNum := getLineNumberFromPosition(file.Content, match[0])
				issues = append(issues, types.Issue{
					RuleID:      r.ID(),
					File:        file.Path,
					Line:        lineNum,
					Column:      getColumnFromPosition(file.Content, match[0]),
					Severity:    r.Severity(),
					Category:    r.Category(),
					Message:     fmt.Sprintf("중복 코드 패턴이 발견되었습니다 (%d회 반복)", len(matches)),
					Description: dp.description,
					Suggestion:  dp.suggestion,
					CodeSnippet: r.getCodeSnippet(file, lineNum),
				})
			}
		}
	}

	// 동일한 라인 블록 검사 (5라인 이상)
	r.checkDuplicateBlocks(file, &issues)

	return issues
}

func (r *DuplicateCodeRule) checkDuplicateBlocks(file *parser.ParsedFile, issues *[]types.Issue) {
	blockSize := 5 // 최소 5라인 블록
	blocks := make(map[string][]int) // 정규화된 블록 -> 라인 번호들

	for i := 0; i <= len(file.Lines)-blockSize; i++ {
		block := r.normalizeBlock(file.Lines[i : i+blockSize])
		if block != "" {
			blocks[block] = append(blocks[block], i+1)
		}
	}

	for _, lines := range blocks {
		if len(lines) >= 2 { // 2번 이상 나타나면 중복
			for _, lineNum := range lines {
				*issues = append(*issues, types.Issue{
					RuleID:      r.ID(),
					File:        file.Path,
					Line:        lineNum,
					Column:      1,
					Severity:    r.Severity(),
					Category:    r.Category(),
					Message:     fmt.Sprintf("중복된 코드 블록이 발견되었습니다 (%d개 위치에서 반복)", len(lines)),
					Description: "동일한 코드 블록이 여러 곳에서 반복되고 있습니다",
					Suggestion:  "공통 메소드로 추출하여 중복을 제거하세요",
					CodeSnippet: r.getCodeSnippet(file, lineNum),
				})
			}
		}
	}
}

func (r *DuplicateCodeRule) normalizeBlock(lines []string) string {
	var normalized []string
	
	for _, line := range lines {
		// 공백 제거 및 정규화
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			continue // 빈 라인, 주석 제외
		}
		
		// 변수명, 문자열 등을 플레이스홀더로 변경하여 구조적 유사성 검사
		normalized = append(normalized, r.normalizeCodeLine(trimmed))
	}
	
	if len(normalized) < 3 { // 실제 코드가 3라인 미만이면 제외
		return ""
	}
	
	return strings.Join(normalized, "\n")
}

func (r *DuplicateCodeRule) normalizeCodeLine(line string) string {
	// 문자열 리터럴을 플레이스홀더로 변경
	stringRegex := regexp.MustCompile(`"[^"]*"`)
	line = stringRegex.ReplaceAllString(line, `"STRING"`)
	
	// 숫자를 플레이스홀더로 변경
	numberRegex := regexp.MustCompile(`\b\d+\b`)
	line = numberRegex.ReplaceAllString(line, "NUM")
	
	// 변수명을 단순화 (camelCase, snake_case 등)
	variableRegex := regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)
	line = variableRegex.ReplaceAllString(line, "VAR")
	
	return line
}

func (r *DuplicateCodeRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}

// CodingConventionRule 코딩 컨벤션 검사
type CodingConventionRule struct {
	config config.RuleConfig
}

func NewCodingConventionRule(cfg config.RuleConfig) Rule {
	return &CodingConventionRule{config: cfg}
}

func (r *CodingConventionRule) ID() string                 { return r.config.ID }
func (r *CodingConventionRule) Name() string               { return r.config.Name }
func (r *CodingConventionRule) Severity() config.Severity { return config.ParseSeverity(r.config.Severity) }
func (r *CodingConventionRule) Category() string          { return r.config.Category }
func (r *CodingConventionRule) Description() string       { return r.config.Description }

func (r *CodingConventionRule) Check(file *parser.ParsedFile) []types.Issue {
	var issues []types.Issue

	javaClass, ok := file.AST.(*parser.JavaClass)
	if !ok {
		return issues
	}

	// @Resource vs @Autowired 혼용 검사
	r.checkAnnotationConsistency(file, &issues)

	// 네이밍 컨벤션 검사
	r.checkNamingConvention(javaClass, file, &issues)

	// 코드 스타일 검사
	r.checkCodeStyle(file, &issues)

	return issues
}

func (r *CodingConventionRule) checkAnnotationConsistency(file *parser.ParsedFile, issues *[]types.Issue) {
	hasResource := strings.Contains(file.Content, "@Resource")
	hasAutowired := strings.Contains(file.Content, "@Autowired")

	if hasResource && hasAutowired {
		// @Resource와 @Autowired가 모두 사용된 경우
		resourceRegex := regexp.MustCompile(`@Resource`)
		autowiredRegex := regexp.MustCompile(`@Autowired`)

		resourceMatches := resourceRegex.FindAllStringIndex(file.Content, -1)
		autowiredMatches := autowiredRegex.FindAllStringIndex(file.Content, -1)

		// @Resource 사용 위치에 경고
		for _, match := range resourceMatches {
			lineNum := getLineNumberFromPosition(file.Content, match[0])
			*issues = append(*issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, match[0]),
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "@Resource와 @Autowired가 혼용되고 있습니다",
				Description: "일관되지 않은 어노테이션 사용은 코드 품질을 저하시킵니다",
				Suggestion:  "@Autowired로 통일하여 사용하세요 (Spring 권장사항)",
				CodeSnippet: r.getCodeSnippet(file, lineNum),
			})
		}

		// @Autowired 사용 위치에도 정보성 메시지
		for _, match := range autowiredMatches {
			lineNum := getLineNumberFromPosition(file.Content, match[0])
			*issues = append(*issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        lineNum,
				Column:      getColumnFromPosition(file.Content, match[0]),
				Severity:    config.SeverityLow, // 정보성 메시지
				Category:    r.Category(),
				Message:     "동일 클래스에서 @Resource와 @Autowired가 혼용되고 있습니다",
				Description: "의존성 주입 어노테이션을 통일하는 것이 좋습니다",
				Suggestion:  "프로젝트 전체에서 @Autowired로 통일하세요",
				CodeSnippet: r.getCodeSnippet(file, lineNum),
			})
		}
	}
}

func (r *CodingConventionRule) checkNamingConvention(javaClass *parser.JavaClass, file *parser.ParsedFile, issues *[]types.Issue) {
	// 클래스명 PascalCase 검사
	if !r.isPascalCase(javaClass.Name) {
		*issues = append(*issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        1,
			Column:      1,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "클래스명이 PascalCase 규칙을 따르지 않습니다: " + javaClass.Name,
			Description: "Java 네이밍 컨벤션에 따라 클래스명은 PascalCase를 사용해야 합니다",
			Suggestion:  "클래스명을 PascalCase로 변경하세요",
			CodeSnippet: "class " + javaClass.Name,
		})
	}

	// 메소드명 camelCase 검사
	for _, method := range javaClass.Methods {
		if !r.isCamelCase(method.Name) && !r.isSpecialMethod(method.Name) {
			*issues = append(*issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        method.Line,
				Column:      method.Column,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "메소드명이 camelCase 규칙을 따르지 않습니다: " + method.Name,
				Description: "Java 네이밍 컨벤션에 따라 메소드명은 camelCase를 사용해야 합니다",
				Suggestion:  "메소드명을 camelCase로 변경하세요",
				CodeSnippet: r.getCodeSnippet(file, method.Line),
			})
		}
	}

	// 필드명 camelCase 검사
	for _, field := range javaClass.Fields {
		if !r.isCamelCase(field.Name) && !r.isConstant(field) {
			*issues = append(*issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        field.Line,
				Column:      1,
				Severity:    r.Severity(),
				Category:    r.Category(),
				Message:     "필드명이 camelCase 규칙을 따르지 않습니다: " + field.Name,
				Description: "Java 네이밍 컨벤션에 따라 필드명은 camelCase를 사용해야 합니다",
				Suggestion:  "필드명을 camelCase로 변경하세요",
				CodeSnippet: r.getCodeSnippet(file, field.Line),
			})
		}
	}
}

func (r *CodingConventionRule) checkCodeStyle(file *parser.ParsedFile, issues *[]types.Issue) {
	// 탭과 스페이스 혼용 검사
	hasTab := strings.Contains(file.Content, "\t")
	hasSpaceIndent := regexp.MustCompile(`^\s{4,}`).MatchString(file.Content)

	if hasTab && hasSpaceIndent {
		*issues = append(*issues, types.Issue{
			RuleID:      r.ID(),
			File:        file.Path,
			Line:        1,
			Column:      1,
			Severity:    r.Severity(),
			Category:    r.Category(),
			Message:     "탭과 스페이스가 혼용되고 있습니다",
			Description: "일관된 들여쓰기를 사용해야 코드 가독성이 향상됩니다",
			Suggestion:  "탭 또는 스페이스 중 하나로 통일하세요",
			CodeSnippet: "",
		})
	}

	// 긴 라인 검사 (120자 초과)
	for i, line := range file.Lines {
		if len(line) > 120 {
			*issues = append(*issues, types.Issue{
				RuleID:      r.ID(),
				File:        file.Path,
				Line:        i + 1,
				Column:      121,
				Severity:    config.SeverityLow,
				Category:    r.Category(),
				Message:     fmt.Sprintf("라인이 너무 깁니다 (%d자)", len(line)),
				Description: "긴 라인은 가독성을 저하시킵니다",
				Suggestion:  "라인을 120자 이하로 분할하세요",
				CodeSnippet: r.getCodeSnippet(file, i+1),
			})
		}
	}
}

func (r *CodingConventionRule) isPascalCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	// 첫 글자가 대문자이고, 언더스코어나 다른 특수문자가 없어야 함
	return name[0] >= 'A' && name[0] <= 'Z' && !strings.Contains(name, "_")
}

func (r *CodingConventionRule) isCamelCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	// 첫 글자가 소문자이고, 언더스코어가 없어야 함
	return name[0] >= 'a' && name[0] <= 'z' && !strings.Contains(name, "_")
}

func (r *CodingConventionRule) isSpecialMethod(name string) bool {
	// 생성자, getter/setter, toString 등 특별한 메소드들
	specialMethods := []string{"toString", "hashCode", "equals", "main"}
	for _, special := range specialMethods {
		if name == special {
			return true
		}
	}
	// getter/setter 패턴
	return strings.HasPrefix(name, "get") || strings.HasPrefix(name, "set") || strings.HasPrefix(name, "is")
}

func (r *CodingConventionRule) isConstant(field parser.JavaField) bool {
	// static final 필드는 상수로 간주
	return field.IsStatic && field.IsFinal
}

func (r *CodingConventionRule) getCodeSnippet(file *parser.ParsedFile, line int) string {
	if line <= 0 || line > len(file.Lines) {
		return ""
	}
	return strings.TrimSpace(file.Lines[line-1])
}