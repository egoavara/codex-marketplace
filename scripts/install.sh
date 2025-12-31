#!/bin/bash
set -e

REPO="egoavara/codex-marketplace"
BINARY_NAME="codex-market"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# OS 및 아키텍처 감지
detect_platform() {
    local os arch

    case "$(uname -s)" in
        Darwin) os="darwin" ;;
        Linux) os="linux" ;;
        MINGW*|MSYS*|CYGWIN*) os="windows" ;;
        *) error "지원하지 않는 OS: $(uname -s)" ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) error "지원하지 않는 아키텍처: $(uname -m)" ;;
    esac

    echo "${os}-${arch}"
}

# 최신 버전 가져오기
get_latest_version() {
    curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
}

# 다운로드 및 설치
install() {
    local platform version download_url binary_path

    platform=$(detect_platform)
    info "감지된 플랫폼: ${platform}"

    # 버전 결정
    if [ -n "$VERSION" ]; then
        version="$VERSION"
    else
        info "최신 버전 확인 중..."
        version=$(get_latest_version)
    fi

    if [ -z "$version" ]; then
        error "버전을 가져올 수 없습니다"
    fi

    info "설치할 버전: ${version}"

    # 다운로드 URL 생성
    local suffix="${platform}"
    if [ "$platform" = "windows-amd64" ]; then
        suffix="${platform}.exe"
    fi
    download_url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}-${version}-${suffix}"

    # 설치 디렉토리 생성
    mkdir -p "$INSTALL_DIR"

    # 다운로드
    info "다운로드 중: ${download_url}"
    binary_path="${INSTALL_DIR}/${BINARY_NAME}"

    if command -v curl &> /dev/null; then
        curl -sL "$download_url" -o "$binary_path"
    elif command -v wget &> /dev/null; then
        wget -q "$download_url" -O "$binary_path"
    else
        error "curl 또는 wget이 필요합니다"
    fi

    # 실행 권한 부여
    chmod +x "$binary_path"

    # 설치 확인
    if [ -x "$binary_path" ]; then
        info "설치 완료: ${binary_path}"
        info "버전 확인:"
        "$binary_path" version
    else
        error "설치 실패"
    fi

    # PATH 확인
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warn "PATH에 ${INSTALL_DIR}가 포함되어 있지 않습니다"
        echo ""
        echo "다음 명령을 실행하여 PATH에 추가하세요:"
        echo ""
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
        echo "영구적으로 추가하려면 ~/.bashrc 또는 ~/.zshrc에 추가하세요"
    fi
}

# 도움말
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -v, --version VERSION  설치할 버전 지정 (기본: 최신)"
    echo "  -d, --dir DIR          설치 디렉토리 (기본: ~/.local/bin)"
    echo "  -h, --help             도움말 표시"
    echo ""
    echo "Examples:"
    echo "  $0                     # 최신 버전 설치"
    echo "  $0 -v v0.1.0           # 특정 버전 설치"
    echo "  $0 -d /usr/local/bin   # 다른 디렉토리에 설치"
}

# 인자 파싱
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -d|--dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            error "알 수 없는 옵션: $1"
            ;;
    esac
done

# 메인 실행
echo ""
echo "╔═══════════════════════════════════════╗"
echo "║     codex-market 설치 스크립트        ║"
echo "╚═══════════════════════════════════════╝"
echo ""

install
