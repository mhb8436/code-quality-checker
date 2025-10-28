# Code Quality Checker Makefile
# 코드 품질 검사기 빌드 자동화

VERSION ?= 1.0.0
APP_NAME = cqc
BUILD_DIR = build
CMD_DIR = cmd/cqc

# Build flags
LDFLAGS = -s -w -X main.version=$(VERSION)
GCFLAGS = -trimpath

.PHONY: all build clean test install help

# Default target
all: clean test build

# Help
help:
	@echo "Code Quality Checker Build Commands:"
	@echo "  make build     - 모든 플랫폼용 빌드"
	@echo "  make test      - 테스트 실행"
	@echo "  make clean     - 빌드 파일 정리"
	@echo "  make install   - 로컬 설치 (현재 플랫폼)"
	@echo "  make dev       - 개발용 빌드 (현재 플랫폼)"
	@echo "  make package   - 배포 패키지 생성"

# Test
test:
	@echo "🧪 테스트 실행 중..."
	go test -v ./...

# Clean
clean:
	@echo "🧹 빌드 파일 정리 중..."
	rm -rf $(BUILD_DIR)
	go clean

# Development build (current platform only)
dev:
	@echo "🔧 개발용 빌드 중..."
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)

# Install to local GOPATH/bin
install:
	@echo "📦 로컬 설치 중..."
	go install -ldflags="$(LDFLAGS)" ./$(CMD_DIR)

# Cross-platform build
build: clean
	@echo "🏗️ 크로스 플랫폼 빌드 시작..."
	mkdir -p $(BUILD_DIR)
	
	# Windows
	@echo "🔧 Windows 빌드 중..."
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./$(CMD_DIR)
	GOOS=windows GOARCH=386 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-386.exe ./$(CMD_DIR)
	
	# Linux
	@echo "🔧 Linux 빌드 중..."
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=386 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-386 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./$(CMD_DIR)
	
	# macOS
	@echo "🔧 macOS 빌드 중..."
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./$(CMD_DIR)
	
	# Copy configs
	@echo "📄 설정 파일 복사 중..."
	cp -r configs $(BUILD_DIR)/
	
	@echo "✅ 빌드 완료!"
	@ls -la $(BUILD_DIR)/

# Package for distribution
package: build
	@echo "📦 배포 패키지 생성 중..."
	cd $(BUILD_DIR) && \
	for file in $(APP_NAME)-*; do \
		if echo $$file | grep -q windows; then \
			zip -q "$${file%.*}.zip" "$$file" -r configs/; \
		else \
			tar -czf "$${file}.tar.gz" "$$file" configs/; \
		fi; \
	done
	@echo "✅ 패키지 생성 완료"
	@ls -la $(BUILD_DIR)/*.{zip,tar.gz} 2>/dev/null || true

# Quick build for current platform
quick:
	@echo "⚡ 빠른 빌드 (현재 플랫폼)..."
	go build -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)

# Dependencies
deps:
	@echo "📥 의존성 설치 중..."
	go mod download
	go mod tidy

# Format code
fmt:
	@echo "🎨 코드 포맷팅 중..."
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@echo "🔍 코드 린트 검사 중..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint가 설치되지 않았습니다."; \
		echo "설치: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run with sample data
run-sample:
	@echo "🚀 샘플 데이터로 실행 중..."
	go run ./$(CMD_DIR) scan --config configs/rules.yaml .

# Show build info
info:
	@echo "빌드 정보:"
	@echo "  Version: $(VERSION)"
	@echo "  App Name: $(APP_NAME)"
	@echo "  Build Dir: $(BUILD_DIR)"
	@echo "  Go Version: $$(go version)"
	@echo "  Git Commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'N/A')"