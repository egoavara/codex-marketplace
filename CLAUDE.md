# codex-marketplace

Codex를 위한 claude-plugin 호환 설치기입니다.

## 프로젝트 목적

- Codex에 claude-plugin 형식의 플러그인을 설치하고 관리
- claude-plugin marketplace 저장소 등록 및 관리
- skills, commands 플러그인 설치 지원

## 주요 명령어

```bash
# marketplace 관리
codex-market marketplace add <git-url>    # marketplace 등록
codex-market marketplace list             # 등록된 marketplace 목록
codex-market marketplace update           # marketplace 업데이트

# plugin 관리
codex-market plugin install <plugin>@<marketplace>  # 플러그인 설치
codex-market plugin list                            # 설치된 플러그인 목록
codex-market plugin uninstall <plugin>              # 플러그인 제거
codex-market plugin search <query>                  # 플러그인 검색
```

## 지원하는 plugin source 타입

- `path`: 로컬 경로 (예: `"./plugins/my-plugin"`)
- `url`: Git URL (예: `{"source": "url", "url": "https://..."}`)
- `github`: GitHub 저장소 (예: `{"source": "github", "repo": "owner/repo"}`)
