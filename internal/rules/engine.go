package rules

import (
	"code-quality-checker/internal/config"
	"code-quality-checker/internal/parser"
	"code-quality-checker/internal/types"
)

// Rule 규칙 인터페이스
type Rule interface {
	ID() string
	Name() string
	Severity() config.Severity
	Category() string
	Description() string
	Check(file *parser.ParsedFile) []types.Issue
}

// Engine 규칙 엔진
type Engine struct {
	config *config.Config
	rules  map[string][]Rule // 언어별 규칙
}

// NewEngine 새로운 규칙 엔진 생성
func NewEngine(cfg *config.Config) *Engine {
	engine := &Engine{
		config: cfg,
		rules:  make(map[string][]Rule),
	}

	// 언어별 규칙 초기화
	engine.initializeRules()

	return engine
}

// initializeRules 규칙 초기화
func (e *Engine) initializeRules() {
	// Java 규칙 등록
	e.registerJavaRules()
	
	// JavaScript 규칙 등록
	e.registerJavaScriptRules()
	
	// HTML 규칙 등록
	e.registerHTMLRules()
	
	// CSS 규칙 등록
	e.registerCSSRules()
}

// CheckFile 파일 검사
func (e *Engine) CheckFile(file *parser.ParsedFile, language string) []types.Issue {
	var allIssues []types.Issue

	rules, exists := e.rules[language]
	if !exists {
		return allIssues
	}

	// 각 규칙 실행
	for _, rule := range rules {
		issues := rule.Check(file)
		allIssues = append(allIssues, issues...)
	}

	return allIssues
}

// registerJavaRules Java 규칙 등록
func (e *Engine) registerJavaRules() {
	javaRules := e.config.GetRulesForLanguage("java")
	var rules []Rule

	for _, ruleConfig := range javaRules {
		switch ruleConfig.ID {
		case "java-transactional-missing":
			rules = append(rules, NewTransactionalRule(ruleConfig))
		case "java-system-out":
			rules = append(rules, NewSystemOutRule(ruleConfig))
		case "java-layer-architecture":
			rules = append(rules, NewLayerArchitectureRule(ruleConfig))
		case "java-exception-handling":
			rules = append(rules, NewExceptionHandlingRule(ruleConfig))
		case "java-input-validation":
			rules = append(rules, NewInputValidationRule(ruleConfig))
		case "java-magic-number":
			rules = append(rules, NewMagicNumberRule(ruleConfig))
		case "java-method-length":
			rules = append(rules, NewMethodLengthRule(ruleConfig))
		case "java-cyclomatic-complexity":
			rules = append(rules, NewCyclomaticComplexityRule(ruleConfig))
		case "java-duplicate-code":
			rules = append(rules, NewDuplicateCodeRule(ruleConfig))
		case "java-coding-conventions":
			rules = append(rules, NewCodingConventionRule(ruleConfig))
		// Spring Framework 규칙들
		case "spring-validation-missing":
			rules = append(rules, NewSpringValidationRule(ruleConfig))
		case "spring-transactional-private":
			rules = append(rules, NewSpringTransactionalRule(ruleConfig))
		case "spring-transactional-rollback":
			rules = append(rules, NewSpringTransactionalRule(ruleConfig))
		case "spring-security-missing":
			rules = append(rules, NewSpringSecurityRule(ruleConfig))
		case "spring-secured-deprecated":
			rules = append(rules, NewSpringSecurityRule(ruleConfig))
		case "spring-field-injection":
			rules = append(rules, NewSpringDependencyInjectionRule(ruleConfig))
		case "spring-controller-advice-missing":
			rules = append(rules, NewSpringExceptionHandlingRule(ruleConfig))
		}
	}

	e.rules["java"] = rules
}

// registerJavaScriptRules JavaScript 규칙 등록
func (e *Engine) registerJavaScriptRules() {
	jsRules := e.config.GetRulesForLanguage("javascript")
	var rules []Rule

	for _, ruleConfig := range jsRules {
		switch ruleConfig.ID {
		case "js-innerHTML-xss":
			rules = append(rules, NewInnerHTMLXSSRule(ruleConfig))
		case "js-memory-leak":
			rules = append(rules, NewMemoryLeakRule(ruleConfig))
		case "js-function-length":
			rules = append(rules, NewFunctionLengthRule(ruleConfig))
		case "js-console-log":
			rules = append(rules, NewConsoleLogRule(ruleConfig))
		case "js-var-usage":
			rules = append(rules, NewVarUsageRule(ruleConfig))
		}
	}

	e.rules["javascript"] = rules
	e.rules["typescript"] = rules // TypeScript도 같은 규칙 적용
}

// registerHTMLRules HTML 규칙 등록
func (e *Engine) registerHTMLRules() {
	htmlRules := e.config.GetRulesForLanguage("html")
	var rules []Rule

	for _, ruleConfig := range htmlRules {
		switch ruleConfig.ID {
		case "html-img-alt":
			rules = append(rules, NewImgAltRule(ruleConfig))
		case "html-accessibility":
			rules = append(rules, NewAccessibilityRule(ruleConfig))
		case "html-seo":
			rules = append(rules, NewSEORule(ruleConfig))
		}
	}

	e.rules["html"] = rules
}

// registerCSSRules CSS 규칙 등록
func (e *Engine) registerCSSRules() {
	cssRules := e.config.GetRulesForLanguage("css")
	var rules []Rule

	for _, ruleConfig := range cssRules {
		switch ruleConfig.ID {
		case "css-selectors":
			rules = append(rules, NewCSSSelectorsRule(ruleConfig))
		case "css-responsive-design":
			rules = append(rules, NewResponsiveDesignRule(ruleConfig))
		}
	}

	e.rules["css"] = rules
}