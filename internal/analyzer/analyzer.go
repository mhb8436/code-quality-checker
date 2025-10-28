package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code-quality-checker/internal/config"
	"code-quality-checker/internal/parser"
	"code-quality-checker/internal/rules"
	"code-quality-checker/internal/types"
)

// Type aliases for backward compatibility
type Issue = types.Issue
type AnalysisResult = types.AnalysisResult
type Summary = types.Summary

// Analyzer 코드 분석기
type Analyzer struct {
	config     *config.Config
	ruleEngine *rules.Engine
}

// New 새로운 분석기 생성
func New(cfg *config.Config) *Analyzer {
	return &Analyzer{
		config:     cfg,
		ruleEngine: rules.NewEngine(cfg),
	}
}

// Analyze 코드 분석 실행
func (a *Analyzer) Analyze(targetPath string) (*AnalysisResult, error) {
	startTime := time.Now()
	
	result := &AnalysisResult{
		StartTime: startTime,
		Summary: Summary{
			SeverityCount: make(map[config.Severity]int),
			CategoryCount: make(map[string]int),
			LanguageCount: make(map[string]int),
		},
	}

	// 대상 파일 수집
	files, err := a.collectFiles(targetPath)
	if err != nil {
		return nil, fmt.Errorf("파일 수집 실패: %w", err)
	}

	result.Summary.TotalFiles = len(files)

	// 각 파일 분석
	for _, file := range files {
		issues, err := a.analyzeFile(file)
		if err != nil {
			fmt.Printf("경고: %s 파일 분석 중 오류 발생: %v\n", file, err)
			continue
		}

		result.Issues = append(result.Issues, issues...)
		
		// 언어별 카운트 업데이트
		language := a.detectLanguage(file)
		result.Summary.LanguageCount[language]++
	}

	// 요약 정보 계산
	result.Summary.TotalIssues = len(result.Issues)
	for _, issue := range result.Issues {
		result.Summary.SeverityCount[issue.Severity]++
		result.Summary.CategoryCount[issue.Category]++
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// collectFiles 분석할 파일 수집
func (a *Analyzer) collectFiles(targetPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// 제외할 디렉토리 스킵
			dirName := filepath.Base(path)
			if a.shouldSkipDirectory(dirName) {
				return filepath.SkipDir
			}
			return nil
		}

		// 지원하는 파일 확장자인지 확인
		if a.isSupportedFile(path) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// shouldSkipDirectory 스킵할 디렉토리인지 확인
func (a *Analyzer) shouldSkipDirectory(dirName string) bool {
	skipDirs := []string{
		".git", ".svn", ".hg",
		"node_modules", "vendor", "target",
		"build", "dist", ".gradle",
		"__pycache__", ".pytest_cache",
		".idea", ".vscode",
	}

	for _, skip := range skipDirs {
		if dirName == skip {
			return true
		}
	}
	return false
}

// isSupportedFile 지원하는 파일인지 확인
func (a *Analyzer) isSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	supportedExts := []string{".java", ".js", ".jsx", ".ts", ".tsx", ".html", ".htm", ".css", ".scss", ".less"}
	
	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

// detectLanguage 파일 확장자로 언어 감지
func (a *Analyzer) detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	switch ext {
	case ".java":
		return "java"
	case ".js", ".jsx":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".html", ".htm":
		return "html"
	case ".css", ".scss", ".less":
		return "css"
	default:
		return "unknown"
	}
}

// analyzeFile 개별 파일 분석
func (a *Analyzer) analyzeFile(filePath string) ([]Issue, error) {
	language := a.detectLanguage(filePath)
	
	// 파일 파싱
	parseResult, err := parser.ParseFile(filePath, language)
	if err != nil {
		return nil, fmt.Errorf("파일 파싱 실패: %w", err)
	}

	// 규칙 엔진으로 검사
	issues := a.ruleEngine.CheckFile(parseResult, language)

	// 파일 경로를 상대 경로로 변환
	for i := range issues {
		issues[i].File = filePath
	}

	return issues, nil
}