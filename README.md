# codex-market

Claude의 마켓플레이스 플러그인을 Codex에서 사용할 수 있게 해주는 CLI 도구입니다.

## 설치

### Homebrew (macOS/Linux)

```bash
brew tap egoavara/codex-market
brew install codex-market
```

특정 버전 설치:

```bash
brew install egoavara/codex-market/codex-market@0.1
```

### 설치 스크립트

```bash
curl -fsSL https://raw.githubusercontent.com/egoavara/codex-marketplace/main/scripts/install.sh | bash
```

특정 버전 설치:
```bash
curl -fsSL https://raw.githubusercontent.com/egoavara/codex-marketplace/main/scripts/install.sh | bash -s -- -v v0.1.0
```

설치 디렉토리 지정:
```bash
curl -fsSL https://raw.githubusercontent.com/egoavara/codex-marketplace/main/scripts/install.sh | bash -s -- -d /usr/local/bin
```

### 수동 설치

[Releases](https://github.com/egoavara/codex-marketplace/releases)에서 플랫폼에 맞는 바이너리를 다운로드하여 PATH에 추가하세요.

## 사용법

### 마켓플레이스 추가

```bash
codex-market add <git-url>
```

예시:
```bash
codex-market add git@github.com:example/skills.git
```

### 플러그인 검색

```bash
codex-market search <keyword>
```

### 플러그인 설치

```bash
codex-market install <plugin>@<marketplace>
```

설치된 스킬은 `~/.codex/skills/`에 저장됩니다.

### 설치된 플러그인 목록

```bash
codex-market list
```

### 플러그인 삭제

```bash
codex-market remove <plugin>@<marketplace>
```

### 마켓플레이스 업데이트

```bash
codex-market update
```

### 설정 관리

```bash
# 현재 설정 보기
codex-market config show

# 설정 변경
codex-market config set locale ko-KR
codex-market config set claude.registry.share sync
```

## 설정

설정 파일 위치: `~/.config/codex-market/config.json`

### 로케일 설정

```bash
codex-market config set locale auto    # 시스템 로케일 자동 감지
codex-market config set locale ko-KR   # 한국어
codex-market config set locale en-US   # 영어
```

### Claude 레지스트리 공유 모드

```bash
codex-market config set claude.registry.share sync    # Claude settings.json과 동기화
codex-market config set claude.registry.share merge   # 병합 (양쪽 유지)
codex-market config set claude.registry.share ignore  # 독립적으로 관리
```

## 삭제

### Homebrew

```bash
brew uninstall codex-market
```

> Homebrew로 삭제 시 캐시와 마켓플레이스 폴더는 자동 삭제됩니다.
> 설정 파일까지 완전히 삭제하려면: `rm -rf ~/.config/codex-market`

### 언인스톨 스크립트

대화형 삭제:
```bash
curl -fsSL https://raw.githubusercontent.com/egoavara/codex-marketplace/main/scripts/uninstall.sh | bash
```

완전 삭제 (모든 데이터):
```bash
curl -fsSL https://raw.githubusercontent.com/egoavara/codex-marketplace/main/scripts/uninstall.sh | bash -s -- --purge
```

## 라이선스

MIT
