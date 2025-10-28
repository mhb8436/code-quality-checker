package main

import (
	"fmt"
	"os"

	"code-quality-checker/internal/analyzer"
	"code-quality-checker/internal/config"
	"code-quality-checker/internal/reporter"

	"github.com/spf13/cobra"
)

var (
	configFile   string
	outputFormat string
	outputFile   string
	minSeverity  string
	rulesFilter  string
	verbose      bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "cqc [path]",
		Short: "Code Quality Checker - 소스코드 품질 검사 도구",
		Long: `Code Quality Checker (CQC)
		
CODE_QUALITY_STANDARDS.md에 정의된 기준에 따라 Java, JavaScript, HTML, CSS 소스코드의 품질을 검사합니다.

사용 예시:
  cqc ./src                           # 기본 검사
  cqc ./src --output=html             # HTML 리포트 생성
  cqc ./src --min-severity=high       # 높은 심각도만 표시
  cqc ./src --rules=security,performance  # 특정 카테고리만 검사`,
		Args: cobra.ExactArgs(1),
		Run:  runAnalysis,
	}

	// 플래그 설정
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "configs/rules.yaml", "설정 파일 경로")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "console", "출력 형식 (console/json/html)")
	rootCmd.Flags().StringVar(&outputFile, "output-file", "", "출력 파일 경로 (기본값: stdout)")
	rootCmd.Flags().StringVarP(&minSeverity, "min-severity", "s", "low", "최소 심각도 (low/medium/high/critical)")
	rootCmd.Flags().StringVar(&rulesFilter, "rules", "", "검사할 규칙 카테고리 (쉼표로 구분)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "상세 출력")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "오류 발생: %v\n", err)
		os.Exit(1)
	}
}

func runAnalysis(cmd *cobra.Command, args []string) {
	targetPath := args[0]

	if verbose {
		fmt.Printf("Code Quality Checker 시작\n")
		fmt.Printf("대상 경로: %s\n", targetPath)
		fmt.Printf("설정 파일: %s\n", configFile)
		fmt.Printf("출력 형식: %s\n", outputFormat)
	}

	// 1. 설정 로드
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "설정 파일 로드 실패: %v\n", err)
		os.Exit(1)
	}

	// 2. 설정 필터링
	if rulesFilter != "" {
		cfg.FilterByCategories(rulesFilter)
	}
	cfg.FilterBySeverity(config.ParseSeverity(minSeverity))

	// 3. 분석 실행
	analyzer := analyzer.New(cfg)
	result, err := analyzer.Analyze(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "분석 실패: %v\n", err)
		os.Exit(1)
	}

	// 4. 결과 리포팅
	rep, err := reporter.New(outputFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "리포터 생성 실패: %v\n", err)
		os.Exit(1)
	}

	err = rep.Generate(result, outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "리포트 생성 실패: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("\n분석 완료! 총 %d개 이슈 발견\n", len(result.Issues))
	}

	// 5. 심각한 이슈가 있으면 종료 코드 1 반환
	if result.HasCriticalIssues() {
		os.Exit(1)
	}
}