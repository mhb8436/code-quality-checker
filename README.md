# Code Quality Checker (CQC)

통합 소스코드 품질 검사 도구

## 📋 개요

Code Quality Checker는 Java, JavaScript, HTML, CSS 소스코드의 품질을 검사하는 통합 도구입니다. 기존의 여러 도구(SonarQube, ESLint, PMD 등)를 사용하지 않고도 종합적인 코드 품질 검사를 수행할 수 있습니다.

## ✨ 주요 기능

- **다중 언어 지원**: Java, JavaScript, HTML, CSS
- **크로스 플랫폼**: Windows, Linux, macOS 지원
- **오프라인 실행**: 인터넷 연결 없이 동작
- **확장 가능**: YAML 설정을 통한 규칙 커스터마이징
- **다양한 출력 형식**: Console, JSON, HTML 리포트
- **한국어 지원**: 한국어 메시지 및 문서

## 🔍 검사 기준

### Java
- @Transactional 어노테이션 누락
- System.out.println 사용
- 레이어 아키텍처 위반
- 매직 넘버 사용
- 메소드 길이 초과
- 예외 처리 누락
- 입력값 검증 누락
- 순환 복잡도 초과
- 중복 코드
- 코딩 컨벤션 위반

### JavaScript
- innerHTML XSS 취약점
- 메모리 누수 위험
- 함수 길이 초과
- console.log 사용
- var 키워드 사용
- Strict Mode 미사용
- 전역 변수 사용
- 콜백 지옥
- 사용하지 않는 변수
- 동등 연산자 사용

### HTML
- img 태그 alt 속성 누락
- 웹 접근성 위반
- SEO 최적화 누락
- 시맨틱 마크업 미사용
- HTML 유효성 검사
- 폐기된 태그 사용
- 인라인 스타일 사용
- 폼 레이블 누락

### CSS
- CSS 셀렉터 효율성
- 반응형 디자인 미적용
- 벤더 프리픽스 누락
- 사용하지 않는 CSS
- !important 남용
- 폰트 폴백 누락
- 색상 대비 부족

## 🚀 설치 및 사용

### 1. 바이너리 다운로드

`build/` 디렉토리에서 해당 플랫폼의 바이너리를 다운로드:

- Windows: `cqc-windows-amd64.exe`
- Linux: `cqc-linux-amd64`
- macOS: `cqc-darwin-amd64`

### 2. 기본 사용법

```bash
# 기본 스캔
./cqc scan /path/to/source

# 설정 파일 지정
./cqc scan --config configs/rules.yaml /path/to/source

# 출력 형식 지정
./cqc scan --format json --output report.json /path/to/source

# HTML 리포트 생성
./cqc scan --format html --output report.html /path/to/source
```

### 3. Windows에서 사용

```cmd
REM 기본 스캔
cqc.exe scan C:\path\to\source

REM JSON 리포트 생성
cqc.exe scan --format json --output report.json C:\path\to\source
```

## ⚙️ 설정

### 설정 파일 구조

`configs/rules.yaml` 파일을 통해 검사 규칙을 커스터마이징할 수 있습니다:

```yaml
languages:
  java:
    enabled: true
    rules:
      - id: "java-transactional-missing"
        severity: "high"
        enabled: true
```

### 심각도 수준

- **Critical**: 즉시 수정 필요한 심각한 문제
- **High**: 릴리즈 전 수정 권장
- **Medium**: 점진적 개선 필요
- **Low**: 시간이 될 때 개선

## 🛠️ 개발자 가이드

### 빌드 요구사항

- Go 1.19 이상
- Make (선택사항)

### 빌드 방법

```bash
# 모든 플랫폼 빌드
make build

# 현재 플랫폼만 빌드
make dev

# 테스트 실행
make test

# 로컬 설치
make install
```

또는 빌드 스크립트 사용:

```bash
# Linux/macOS
./build.sh

# Windows
build.bat
```

### 프로젝트 구조

```
code-quality-checker/
├── cmd/cqc/           # CLI 엔트리 포인트
├── internal/
│   ├── analyzer/      # 분석 엔진
│   ├── config/        # 설정 관리
│   ├── parser/        # 언어별 파서
│   ├── rules/         # 규칙 엔진
│   └── reporter/      # 리포트 생성
├── configs/           # 설정 파일
├── build/            # 빌드 결과물
└── docs/             # 문서
```

### 새로운 규칙 추가

1. `internal/rules/` 에서 해당 언어의 규칙 파일 수정
2. `configs/rules.yaml` 에서 규칙 설정 추가
3. 테스트 케이스 작성
4. 빌드 및 테스트

## 📊 출력 예시

### Console 출력

```
🔍 Code Quality Checker 분석 결과
==================================================

📊 분석 요약
--------------------
검사 파일 수: 25개
발견된 이슈: 12개
분석 시간: 1.23초

⚠️ 심각도별 통계
--------------------
🚨 CRITICAL: 2개
⚠️ HIGH: 5개
📝 MEDIUM: 3개
💡 LOW: 2개
```

### JSON 출력

```json
{
  "summary": {
    "total_files": 25,
    "total_issues": 12,
    "severity_count": {
      "critical": 2,
      "high": 5,
      "medium": 3,
      "low": 2
    }
  },
  "issues": [
    {
      "rule_id": "java-transactional-missing",
      "file": "src/main/java/Service.java",
      "line": 15,
      "severity": "high",
      "message": "@Transactional 어노테이션이 누락되었습니다"
    }
  ]
}
```

## 🤝 기여하기

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📝 라이선스

MIT License

## 🔗 관련 링크

- [코드 품질 기준 문서](CODE_QUALITY_STANDARDS.md)
- [설정 가이드](configs/rules.yaml)
- [개발자 문서](docs/)

## 🚧 로드맵

- [ ] 추가 언어 지원 (Python, TypeScript)
- [ ] IDE 플러그인 개발
- [ ] CI/CD 통합
- [ ] 웹 대시보드
- [ ] 실시간 코드 분석

## 💬 지원

문제가 있으시면 GitHub Issues를 통해 신고해 주세요.

---

**Code Quality Checker** - 통합 소스코드 품질 검사 도구