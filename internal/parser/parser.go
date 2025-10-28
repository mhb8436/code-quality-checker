package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ParsedFile 파싱된 파일 정보
type ParsedFile struct {
	Path     string
	Language string
	Content  string
	Lines    []string
	Tokens   []Token
	AST      interface{} // 언어별로 다른 AST 구조
}

// Token 토큰 정보
type Token struct {
	Type     string
	Value    string
	Line     int
	Column   int
	StartPos int
	EndPos   int
}

// JavaClass Java 클래스 정보
type JavaClass struct {
	Name        string
	Annotations []string
	Methods     []JavaMethod
	Fields      []JavaField
	Imports     []string
	Package     string
}

// JavaMethod Java 메소드 정보  
type JavaMethod struct {
	Name         string
	Annotations  []string
	Parameters   []string
	ReturnType   string
	Line         int
	Column       int
	Body         string
	IsPublic     bool
	IsPrivate    bool
	IsProtected  bool
	IsStatic     bool
}

// JavaField Java 필드 정보
type JavaField struct {
	Name        string
	Type        string
	Annotations []string
	Line        int
	IsStatic    bool
	IsFinal     bool
}

// JSFunction JavaScript 함수 정보
type JSFunction struct {
	Name       string
	Parameters []string
	Line       int
	Column     int
	Body       string
	IsArrow    bool
	IsAsync    bool
}

// ParseFile 파일 파싱
func ParseFile(filePath, language string) (*ParsedFile, error) {
	content, err := readFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(content, "\n")

	parsed := &ParsedFile{
		Path:     filePath,
		Language: language,
		Content:  content,
		Lines:    lines,
	}

	// 언어별 파싱
	switch language {
	case "java":
		parsed.AST, err = parseJava(content, lines)
	case "javascript", "typescript":
		parsed.AST, err = parseJavaScript(content, lines)
	case "html":
		parsed.AST, err = parseHTML(content, lines)
	case "css":
		parsed.AST, err = parseCSS(content, lines)
	default:
		// 기본적으로 텍스트 파싱
		parsed.Tokens = tokenizeText(content)
	}

	if err != nil {
		return nil, fmt.Errorf("언어별 파싱 실패: %w", err)
	}

	return parsed, nil
}

// readFile 파일 읽기
func readFile(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	return content.String(), scanner.Err()
}

// parseJava Java 파일 파싱
func parseJava(content string, lines []string) (*JavaClass, error) {
	class := &JavaClass{}

	// 패키지 추출
	packageRegex := regexp.MustCompile(`package\s+([a-zA-Z0-9_.]+);`)
	if match := packageRegex.FindStringSubmatch(content); len(match) > 1 {
		class.Package = match[1]
	}

	// import 추출
	importRegex := regexp.MustCompile(`import\s+([a-zA-Z0-9_.*]+);`)
	imports := importRegex.FindAllStringSubmatch(content, -1)
	for _, imp := range imports {
		if len(imp) > 1 {
			class.Imports = append(class.Imports, imp[1])
		}
	}

	// 클래스명 추출
	classRegex := regexp.MustCompile(`(?:public\s+)?class\s+(\w+)`)
	if match := classRegex.FindStringSubmatch(content); len(match) > 1 {
		class.Name = match[1]
	}

	// 클래스 어노테이션 추출
	class.Annotations = extractAnnotations(content, 0)

	// 메소드 추출
	class.Methods = extractJavaMethods(content, lines)

	// 필드 추출
	class.Fields = extractJavaFields(content, lines)

	return class, nil
}

// extractJavaMethods Java 메소드 추출
func extractJavaMethods(content string, lines []string) []JavaMethod {
	var methods []JavaMethod

	// 메소드 패턴: (접근제한자)? (기타제한자)* 리턴타입 메소드명(파라미터) {
	methodRegex := regexp.MustCompile(`(?m)^\s*(?:(public|private|protected)\s+)?(?:(static|final|abstract|synchronized)\s+)*(\w+(?:<[^>]+>)?)\s+(\w+)\s*\(([^)]*)\)\s*(?:throws\s+[^{]+)?\s*\{`)

	matches := methodRegex.FindAllStringSubmatch(content, -1)
	indices := methodRegex.FindAllStringIndex(content, -1)

	for i, match := range matches {
		if len(match) >= 5 {
			method := JavaMethod{
				Name:       match[4],
				ReturnType: match[3],
			}

			// 접근 제한자 설정
			if match[1] == "public" {
				method.IsPublic = true
			} else if match[1] == "private" {
				method.IsPrivate = true
			} else if match[1] == "protected" {
				method.IsProtected = true
			}

			// static 여부
			if strings.Contains(match[2], "static") {
				method.IsStatic = true
			}

			// 파라미터 파싱
			if match[5] != "" {
				params := strings.Split(match[5], ",")
				for _, param := range params {
					method.Parameters = append(method.Parameters, strings.TrimSpace(param))
				}
			}

			// 라인 번호 계산
			if i < len(indices) {
				lineNum := getLineNumber(content, indices[i][0])
				method.Line = lineNum
				
				// 메소드 이전 어노테이션 추출
				method.Annotations = extractAnnotations(content, indices[i][0])
			}

			methods = append(methods, method)
		}
	}

	return methods
}

// extractJavaFields Java 필드 추출
func extractJavaFields(content string, lines []string) []JavaField {
	var fields []JavaField

	// 필드 패턴: (접근제한자)? (기타제한자)* 타입 필드명;
	fieldRegex := regexp.MustCompile(`(?m)^\s*(?:(public|private|protected)\s+)?(?:(static|final)\s+)*(\w+(?:<[^>]+>)?)\s+(\w+)\s*(?:=\s*[^;]+)?;`)

	matches := fieldRegex.FindAllStringSubmatch(content, -1)
	indices := fieldRegex.FindAllStringIndex(content, -1)

	for i, match := range matches {
		if len(match) >= 5 {
			field := JavaField{
				Name: match[4],
				Type: match[3],
			}

			// static, final 여부
			if strings.Contains(match[2], "static") {
				field.IsStatic = true
			}
			if strings.Contains(match[2], "final") {
				field.IsFinal = true
			}

			// 라인 번호 계산
			if i < len(indices) {
				field.Line = getLineNumber(content, indices[i][0])
				field.Annotations = extractAnnotations(content, indices[i][0])
			}

			fields = append(fields, field)
		}
	}

	return fields
}

// extractAnnotations 어노테이션 추출
func extractAnnotations(content string, beforePos int) []string {
	var annotations []string

	// beforePos 이전의 내용에서 어노테이션 찾기
	beforeContent := content[:beforePos]
	lines := strings.Split(beforeContent, "\n")

	// 뒤에서부터 어노테이션 찾기
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "@") {
			annotations = append([]string{line}, annotations...) // 앞에 추가
		} else if line != "" && !strings.HasPrefix(line, "//") && !strings.HasPrefix(line, "*") {
			// 어노테이션이 아닌 코드가 나오면 중단
			break
		}
	}

	return annotations
}

// parseJavaScript JavaScript 파일 파싱  
func parseJavaScript(content string, lines []string) ([]JSFunction, error) {
	var functions []JSFunction

	// 함수 패턴들
	patterns := []string{
		`function\s+(\w+)\s*\(([^)]*)\)\s*\{`,           // function name() {}
		`(\w+)\s*:\s*function\s*\(([^)]*)\)\s*\{`,      // name: function() {}
		`(\w+)\s*=\s*function\s*\(([^)]*)\)\s*\{`,      // name = function() {}
		`(\w+)\s*=\s*\(([^)]*)\)\s*=>\s*\{`,            // name = () => {}
		`const\s+(\w+)\s*=\s*\(([^)]*)\)\s*=>\s*\{`,    // const name = () => {}
		`let\s+(\w+)\s*=\s*\(([^)]*)\)\s*=>\s*\{`,      // let name = () => {}
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(content, -1)
		indices := regex.FindAllStringIndex(content, -1)

		for i, match := range matches {
			if len(match) >= 3 {
				function := JSFunction{
					Name: match[1],
				}

				// 파라미터 파싱
				if match[2] != "" {
					params := strings.Split(match[2], ",")
					for _, param := range params {
						function.Parameters = append(function.Parameters, strings.TrimSpace(param))
					}
				}

				// 화살표 함수 여부
				if strings.Contains(pattern, "=>") {
					function.IsArrow = true
				}

				// 라인 번호 계산
				if i < len(indices) {
					function.Line = getLineNumber(content, indices[i][0])
				}

				functions = append(functions, function)
			}
		}
	}

	return functions, nil
}

// parseHTML HTML 파일 파싱
func parseHTML(content string, lines []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	// 기본적인 HTML 요소 추출
	result["images"] = extractHTMLImages(content)
	result["forms"] = extractHTMLForms(content)
	result["scripts"] = extractHTMLScripts(content)
	
	return result, nil
}

// parseCSS CSS 파일 파싱
func parseCSS(content string, lines []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	// CSS 선택자 추출
	result["selectors"] = extractCSSSelectors(content)
	
	return result, nil
}

// 헬퍼 함수들
func extractHTMLImages(content string) []map[string]string {
	var images []map[string]string
	imgRegex := regexp.MustCompile(`<img[^>]*>`)
	matches := imgRegex.FindAllString(content, -1)
	
	for _, match := range matches {
		img := make(map[string]string)
		img["tag"] = match
		
		// src 속성 추출
		srcRegex := regexp.MustCompile(`src\s*=\s*["']([^"']*)["']`)
		if srcMatch := srcRegex.FindStringSubmatch(match); len(srcMatch) > 1 {
			img["src"] = srcMatch[1]
		}
		
		// alt 속성 추출
		altRegex := regexp.MustCompile(`alt\s*=\s*["']([^"']*)["']`)
		if altMatch := altRegex.FindStringSubmatch(match); len(altMatch) > 1 {
			img["alt"] = altMatch[1]
		}
		
		images = append(images, img)
	}
	
	return images
}

func extractHTMLForms(content string) []string {
	formRegex := regexp.MustCompile(`<form[^>]*>`)
	return formRegex.FindAllString(content, -1)
}

func extractHTMLScripts(content string) []string {
	scriptRegex := regexp.MustCompile(`<script[^>]*>[\s\S]*?</script>`)
	return scriptRegex.FindAllString(content, -1)
}

func extractCSSSelectors(content string) []string {
	selectorRegex := regexp.MustCompile(`([^{}]+)\s*\{`)
	matches := selectorRegex.FindAllStringSubmatch(content, -1)
	
	var selectors []string
	for _, match := range matches {
		if len(match) > 1 {
			selector := strings.TrimSpace(match[1])
			if selector != "" && !strings.HasPrefix(selector, "@") {
				selectors = append(selectors, selector)
			}
		}
	}
	
	return selectors
}

// tokenizeText 기본 텍스트 토큰화
func tokenizeText(content string) []Token {
	var tokens []Token
	lines := strings.Split(content, "\n")
	
	for lineNum, line := range lines {
		words := strings.Fields(line)
		for colNum, word := range words {
			tokens = append(tokens, Token{
				Type:   "word",
				Value:  word,
				Line:   lineNum + 1,
				Column: colNum + 1,
			})
		}
	}
	
	return tokens
}

// getLineNumber 위치에서 라인 번호 계산
func getLineNumber(content string, pos int) int {
	return strings.Count(content[:pos], "\n") + 1
}