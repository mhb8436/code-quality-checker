#!/bin/bash

# Code Quality Checker - Build Script
# 코드 품질 검사기 빌드 스크립트

set -e

VERSION="1.0.0"
APP_NAME="cqc"
BUILD_DIR="build"

echo "🔨 Code Quality Checker 빌드 시작..."
echo "Version: $VERSION"

# 빌드 디렉토리 생성
mkdir -p $BUILD_DIR

# Go 모듈 정리
echo "📦 Go 모듈 정리 중..."
go mod tidy

# 테스트 실행
echo "🧪 테스트 실행 중..."
go test -v ./...

# 플랫폼별 빌드
platforms=(
    "windows/amd64"
    "windows/386"
    "linux/amd64"
    "linux/386"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

echo "🏗️ 크로스 플랫폼 빌드 시작..."

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    output_name=$APP_NAME
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi
    
    output_path="$BUILD_DIR/${APP_NAME}-${GOOS}-${GOARCH}"
    if [ $GOOS = "windows" ]; then
        output_path+='.exe'
    fi
    
    echo "🔧 빌드 중: $GOOS/$GOARCH"
    
    env GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags="-s -w -X main.version=$VERSION" \
        -o $output_path \
        ./cmd/cqc
    
    if [ $? -ne 0 ]; then
        echo "❌ 빌드 실패: $GOOS/$GOARCH"
        exit 1
    fi
done

# 빌드 완료 메시지
echo ""
echo "✅ 빌드 완료!"
echo "📁 빌드 파일 위치: $BUILD_DIR/"
echo ""
echo "📋 빌드된 바이너리:"
ls -la $BUILD_DIR/

# 설정 파일 복사
echo ""
echo "📄 설정 파일 복사 중..."
cp -r configs $BUILD_DIR/
echo "✅ 설정 파일 복사 완료"

# 사용법 안내
echo ""
echo "🚀 사용법:"
echo "1. 적절한 플랫폼의 바이너리를 선택하세요"
echo "2. configs/ 폴더와 함께 배포하세요"
echo "3. 실행: ./cqc scan /path/to/source"
echo ""

# 패키지 생성 (선택사항)
if command -v zip &> /dev/null; then
    echo "📦 배포 패키지 생성 중..."
    cd $BUILD_DIR
    for file in cqc-*; do
        if [[ $file == *"windows"* ]]; then
            zip -q "${file%-*}-${file##*-}.zip" "$file" -r configs/
        else
            tar -czf "${file%-*}-${file##*-}.tar.gz" "$file" configs/
        fi
    done
    cd ..
    echo "✅ 배포 패키지 생성 완료"
fi

echo "🎉 모든 작업이 완료되었습니다!"