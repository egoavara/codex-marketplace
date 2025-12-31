#!/bin/bash
set -e

BINARY_NAME="codex-market"
CONFIG_DIR="${HOME}/.config/codex-market"
CACHE_DIR="${CONFIG_DIR}/cache"
MARKETPLACES_DIR="${CONFIG_DIR}/marketplaces"

# 색상 정의
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

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

# 바이너리 찾기
find_binary() {
    local paths=(
        "${HOME}/.local/bin/${BINARY_NAME}"
        "/usr/local/bin/${BINARY_NAME}"
        "/opt/homebrew/bin/${BINARY_NAME}"
    )

    for path in "${paths[@]}"; do
        if [ -x "$path" ]; then
            echo "$path"
            return 0
        fi
    done

    # which로 찾기
    if command -v "$BINARY_NAME" &> /dev/null; then
        which "$BINARY_NAME"
        return 0
    fi

    return 1
}

# 폴더 크기 계산
get_folder_size() {
    if [ -d "$1" ]; then
        du -sh "$1" 2>/dev/null | cut -f1
    else
        echo "0"
    fi
}

# 삭제 확인
confirm() {
    local prompt="$1"
    local default="${2:-n}"

    if [ "$FORCE" = "true" ]; then
        return 0
    fi

    local yn
    if [ "$default" = "y" ]; then
        read -p "$prompt [Y/n] " yn
        yn=${yn:-y}
    else
        read -p "$prompt [y/N] " yn
        yn=${yn:-n}
    fi

    case $yn in
        [Yy]*) return 0 ;;
        *) return 1 ;;
    esac
}

# 언인스톨 실행
uninstall() {
    echo ""
    echo "╔═══════════════════════════════════════╗"
    echo "║   codex-market 언인스톨 스크립트      ║"
    echo "╚═══════════════════════════════════════╝"
    echo ""

    # 현재 상태 표시
    echo -e "${CYAN}현재 설치 상태:${NC}"
    echo "────────────────────────────────────────"

    # 바이너리 확인
    local binary_path
    if binary_path=$(find_binary); then
        echo -e "  바이너리: ${GREEN}${binary_path}${NC}"
    else
        echo -e "  바이너리: ${YELLOW}찾을 수 없음${NC}"
    fi

    # 설정 폴더 확인
    if [ -d "$CONFIG_DIR" ]; then
        local config_size=$(get_folder_size "$CONFIG_DIR")
        echo -e "  설정 폴더: ${GREEN}${CONFIG_DIR}${NC} (${config_size})"

        if [ -d "$CACHE_DIR" ]; then
            local cache_size=$(get_folder_size "$CACHE_DIR")
            echo -e "    - 캐시: ${cache_size}"
        fi
        if [ -d "$MARKETPLACES_DIR" ]; then
            local mp_size=$(get_folder_size "$MARKETPLACES_DIR")
            echo -e "    - 마켓플레이스: ${mp_size}"
        fi
        if [ -f "${CONFIG_DIR}/config.json" ]; then
            echo -e "    - config.json: 있음"
        fi
        if [ -f "${CONFIG_DIR}/installed.json" ]; then
            echo -e "    - installed.json: 있음"
        fi
    else
        echo -e "  설정 폴더: ${YELLOW}없음${NC}"
    fi

    echo ""
    echo "────────────────────────────────────────"
    echo ""

    # 삭제 진행
    local removed_something=false

    # 1. 바이너리 삭제
    if [ -n "$binary_path" ] && [ -x "$binary_path" ]; then
        if confirm "바이너리를 삭제하시겠습니까? ($binary_path)"; then
            rm -f "$binary_path"
            info "바이너리 삭제됨: $binary_path"
            removed_something=true
        fi
    fi

    # 2. 캐시 삭제 (항상 삭제 권장)
    if [ -d "$CACHE_DIR" ]; then
        local cache_size=$(get_folder_size "$CACHE_DIR")
        if confirm "캐시 폴더를 삭제하시겠습니까? ($cache_size)" "y"; then
            rm -rf "$CACHE_DIR"
            info "캐시 삭제됨: $CACHE_DIR"
            removed_something=true
        fi
    fi

    # 3. 마켓플레이스 삭제 (권장)
    if [ -d "$MARKETPLACES_DIR" ]; then
        local mp_size=$(get_folder_size "$MARKETPLACES_DIR")
        if confirm "마켓플레이스 폴더를 삭제하시겠습니까? ($mp_size)" "y"; then
            rm -rf "$MARKETPLACES_DIR"
            info "마켓플레이스 삭제됨: $MARKETPLACES_DIR"
            removed_something=true
        fi
    fi

    # 4. 설정 파일 삭제 (선택)
    if [ -f "${CONFIG_DIR}/config.json" ] || [ -f "${CONFIG_DIR}/installed.json" ]; then
        echo ""
        warn "설정 파일을 삭제하면 마켓플레이스 등록 정보와 설치 기록이 사라집니다."
        if confirm "설정 파일을 삭제하시겠습니까? (config.json, installed.json)"; then
            rm -f "${CONFIG_DIR}/config.json" "${CONFIG_DIR}/installed.json"
            info "설정 파일 삭제됨"
            removed_something=true
        fi
    fi

    # 5. 빈 폴더 정리
    if [ -d "$CONFIG_DIR" ]; then
        # 폴더가 비어있으면 삭제
        if [ -z "$(ls -A "$CONFIG_DIR" 2>/dev/null)" ]; then
            rmdir "$CONFIG_DIR" 2>/dev/null || true
            info "빈 설정 폴더 삭제됨"
        fi
    fi

    echo ""
    if [ "$removed_something" = true ]; then
        echo -e "${GREEN}언인스톨이 완료되었습니다.${NC}"
    else
        echo -e "${YELLOW}삭제된 항목이 없습니다.${NC}"
    fi

    # Homebrew로 설치된 경우 안내
    if command -v brew &> /dev/null; then
        if brew list codex-market &> /dev/null 2>&1; then
            echo ""
            warn "Homebrew로 설치된 것 같습니다. 다음 명령으로 완전히 제거하세요:"
            echo ""
            echo "  brew uninstall codex-market"
            echo ""
        fi
    fi
}

# 전체 삭제 (강제)
purge() {
    echo ""
    echo -e "${RED}╔═══════════════════════════════════════╗${NC}"
    echo -e "${RED}║   codex-market 완전 삭제 (PURGE)      ║${NC}"
    echo -e "${RED}╚═══════════════════════════════════════╝${NC}"
    echo ""

    warn "이 작업은 모든 데이터를 삭제합니다!"
    echo ""

    if ! confirm "정말로 모든 데이터를 삭제하시겠습니까?"; then
        echo "취소되었습니다."
        exit 0
    fi

    # 바이너리 삭제
    local binary_path
    if binary_path=$(find_binary); then
        rm -f "$binary_path"
        info "바이너리 삭제됨: $binary_path"
    fi

    # 전체 설정 폴더 삭제
    if [ -d "$CONFIG_DIR" ]; then
        rm -rf "$CONFIG_DIR"
        info "설정 폴더 삭제됨: $CONFIG_DIR"
    fi

    echo ""
    echo -e "${GREEN}완전 삭제가 완료되었습니다.${NC}"
}

# 도움말
usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --purge     모든 데이터 완전 삭제"
    echo "  --force     확인 없이 삭제"
    echo "  -h, --help  도움말 표시"
    echo ""
    echo "Examples:"
    echo "  $0                # 대화형 언인스톨"
    echo "  $0 --purge        # 모든 데이터 삭제"
    echo "  $0 --force        # 확인 없이 삭제"
}

# 인자 파싱
PURGE=false
FORCE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --purge)
            PURGE=true
            shift
            ;;
        --force|-f)
            FORCE=true
            shift
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
if [ "$PURGE" = true ]; then
    purge
else
    uninstall
fi
