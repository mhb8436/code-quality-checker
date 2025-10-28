@echo off
REM Code Quality Checker - Windows Build Script
REM 코드 품질 검사기 윈도우 빌드 스크립트

setlocal enabledelayedexpansion

set VERSION=1.0.0
set APP_NAME=cqc
set BUILD_DIR=build

echo 🔨 Code Quality Checker 빌드 시작...
echo Version: %VERSION%

REM 빌드 디렉토리 생성
if not exist %BUILD_DIR% mkdir %BUILD_DIR%

REM Go 모듈 정리
echo 📦 Go 모듈 정리 중...
go mod tidy
if errorlevel 1 (
    echo ❌ Go 모듈 정리 실패
    exit /b 1
)

REM 테스트 실행
echo 🧪 테스트 실행 중...
go test -v ./...
if errorlevel 1 (
    echo ⚠️ 테스트 실패 - 계속 진행
)

echo 🏗️ 윈도우 플랫폼 빌드 시작...

REM Windows 64비트
echo 🔧 빌드 중: windows/amd64
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w -X main.version=%VERSION%" -o %BUILD_DIR%\%APP_NAME%-windows-amd64.exe .\cmd\cqc
if errorlevel 1 (
    echo ❌ Windows 64비트 빌드 실패
    exit /b 1
)

REM Windows 32비트
echo 🔧 빌드 중: windows/386
set GOOS=windows
set GOARCH=386
go build -ldflags="-s -w -X main.version=%VERSION%" -o %BUILD_DIR%\%APP_NAME%-windows-386.exe .\cmd\cqc
if errorlevel 1 (
    echo ❌ Windows 32비트 빌드 실패
    exit /b 1
)

REM Linux 64비트 (크로스 컴파일)
echo 🔧 빌드 중: linux/amd64
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w -X main.version=%VERSION%" -o %BUILD_DIR%\%APP_NAME%-linux-amd64 .\cmd\cqc
if errorlevel 1 (
    echo ❌ Linux 64비트 빌드 실패
    exit /b 1
)

REM macOS 64비트 (크로스 컴파일)
echo 🔧 빌드 중: darwin/amd64
set GOOS=darwin
set GOARCH=amd64
go build -ldflags="-s -w -X main.version=%VERSION%" -o %BUILD_DIR%\%APP_NAME%-darwin-amd64 .\cmd\cqc
if errorlevel 1 (
    echo ❌ macOS 64비트 빌드 실패
    exit /b 1
)

REM 환경 변수 초기화
set GOOS=
set GOARCH=

echo.
echo ✅ 빌드 완료!
echo 📁 빌드 파일 위치: %BUILD_DIR%\
echo.

echo 📋 빌드된 바이너리:
dir /b %BUILD_DIR%\%APP_NAME%-*

REM 설정 파일 복사
echo.
echo 📄 설정 파일 복사 중...
if not exist %BUILD_DIR%\configs mkdir %BUILD_DIR%\configs
copy configs\*.yaml %BUILD_DIR%\configs\ > nul
echo ✅ 설정 파일 복사 완료

REM 사용법 안내
echo.
echo 🚀 사용법:
echo 1. 적절한 플랫폼의 바이너리를 선택하세요
echo 2. configs\ 폴더와 함께 배포하세요
echo 3. 실행: cqc.exe scan C:\path\to\source
echo.

REM ZIP 패키지 생성 (PowerShell 사용)
echo 📦 배포 패키지 생성 중...
powershell -Command "& {
    $buildDir = '%BUILD_DIR%'
    $files = Get-ChildItem $buildDir -Filter 'cqc-*.exe'
    foreach ($file in $files) {
        $zipName = $file.BaseName + '.zip'
        $zipPath = Join-Path $buildDir $zipName
        Compress-Archive -Path $file.FullName, (Join-Path $buildDir 'configs') -DestinationPath $zipPath -Force
        Write-Host \"생성됨: $zipName\"
    }
}"

echo ✅ 배포 패키지 생성 완료
echo 🎉 모든 작업이 완료되었습니다!

pause