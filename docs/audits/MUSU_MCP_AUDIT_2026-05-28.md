# 무수 삼형제 MCP 실사용 감사 (2026-05-28)

> Claude Code 세션 재시작 직후, 13개 MCP 도구(crawl-ai 3 · marketer 2 · nurikun 8)를 직접 호출해 본 감사. 무엇이 동작하는지, 무엇이 막히는지, 어디부터 고치면 가장 빨리 풀리는지.

---

## 0. 컨텍스트

- **환경**: Windows + Claude Code, MCP 서버 user-scope 등록(`C:\Users\empty\.claude.json`).
- **사전 조건 부재**: Ollama 미구동, 프로젝트 DB/위키 없음 (Claude Code cwd = `C:\Users\empty`).
- **테스트 방법**: 각 MCP 도구를 호출하며 입출력 + 실패 모드를 관찰.

## 1. 호출 결과 한눈에 보기

| 도구 | 호출 가능? | 실제 결과 |
|---|---|---|
| `mcp__musu-nurikun__doctor` | ✅ | 정상 JSON 리포트 (mailbox/ai 미설정 정확히 surface) |
| `mcp__musu-nurikun__list_lists` | ✅ | ❌ `unable to open database file (14)` |
| `mcp__musu-nurikun__messages_by_status` | ❌ (arg 필요) | "status is required" — **clean validation** |
| `mcp__musu-nurikun__create_list` | ❌ (arg 필요) | "name is required" — **clean validation** |
| `mcp__musu-nurikun__subscribe / confirm / list_subscribers / suppress` | ❌ (arg 필요) | 호출 불가 (F1 참조) |
| `mcp__musu-marketer__list_campaigns` | ✅ | ❌ `unable to open database file (14)` |
| `mcp__musu-marketer__draft_campaign` | ❌ (arg 필요) | 호출 불가 (F1 참조) |
| `mcp__musu-crawl-ai__search` | ✅ | "Found 0 results:" — graceful but ambiguous |
| `mcp__musu-crawl-ai__fetch` | ❌ (arg 필요) | 호출 불가 (F1 참조) |
| `mcp__musu-crawl-ai__research` | ✅ (arg 없이 호출됨) | **(빈 출력) — silent no-op** |

**요약**: 13개 중 4개만 arg 없이 호출 가능. 그중 2개는 DB가 없어 실패, 1개는 0 results로 graceful, 1개(doctor)만 의미 있는 결과 반환.

---

## 2. 핵심 발견 (severity 순)

### F1 · [CRITICAL] MCP 도구 스키마가 비어 있음 — args 전달 불가
**증거**: ToolSearch가 반환한 모든 13개 도구의 schema:
```json
{ "parameters": { "properties": {}, "required": [], "type": "object" } }
```
설명문에는 "Args: name (string, required), cadence_days (int, default 4)..." 같은 안내가 있지만, JSON 스키마에는 **declared properties가 0개**. MCP 클라이언트(Claude Code 포함)는 선언되지 않은 args를 전달할 수단이 없거나 거부함.

**영향**: arg를 받는 9개 도구는 사실상 **호출 불가능**. 작동하는 것은 arg-less 4개뿐.

**원인**: 세 레포의 모든 `mcp.NewTool(...)` 호출이 `WithDescription`만 쓰고 `WithString`/`WithNumber`/`Required` 등 **파라미터 선언자를 안 씀**.

**제안 수정** (각 도구 정의에 적용):
```go
// Before (현재)
mcp.NewTool("subscribe", mcp.WithDescription("..."))

// After (제안)
mcp.NewTool("subscribe",
    mcp.WithDescription("Record a pending subscriber..."),
    mcp.WithNumber("list_id", mcp.Required(), mcp.Description("Target list ID")),
    mcp.WithString("email", mcp.Required(), mcp.Description("Subscriber email")),
    mcp.WithString("name", mcp.Description("Optional display name")),
    mcp.WithString("project", mcp.Description("Project scope (default: 'default')")),
)
```
영향 범위: crawl-ai 3 도구 + marketer 2 도구 + nurikun 8 도구 = **총 13 정의 보강**. 기계적 작업이지만 이것 없이는 ecosystem이 MCP로 거의 사용 불가.

### F2 · [HIGH] cwd 의존성 — DB/wiki 경로가 깨짐
**증거**: nurikun/marketer `list_*` 호출이 `unable to open database file (14)` (SQLite SQLITE_CANTOPEN). MCP 서버 프로세스는 Claude Code의 cwd(`C:\Users\empty`)에서 `projects/<name>/data/*.db` 상대경로를 찾음 → 그 경로 미존재.

**영향**: 사용자가 다른 위치에서 만들어 둔 프로젝트 데이터에 도구가 접근 불가. "default" 프로젝트조차 cwd에 따라 못 찾음.

**제안 수정 (3 갈래 모두 도움됨)**:
1. **DB layer**: `db.NewStore`가 SQLite open 전에 `os.MkdirAll(filepath.Dir(path), 0755)` 호출 → 적어도 새 프로젝트는 cwd에 자동 생성. (~3줄)
2. **MCP 등록 시점**: `claude mcp add` 시 `--env MUSU_PROJECT_BASE=C:\path\to\projects` 형태로 절대 base 경로 주입. config loader가 이 env를 우선 사용. (작은 config 수정)
3. **각 도구가 `project_base` arg 받기**: F1 fix와 함께 — args에 절대 경로 옵션 추가, fallback이 cwd-relative.

### F3 · [HIGH] `research`가 빈 입력에 silent하게 빈 결과 반환
**증거**: `mcp__musu-crawl-ai__research`를 인자 없이 호출 → exit 0, "completed with no output". 에러도 안내도 없음.

**원인**: handler가 `args["question"].(string)` 추출 후 빈 문자열 검증 없이 orchestrator.ResearchAction 호출. 빈 question → planner가 빈 plan → 0 sources → 루프 break → empty finalReport.

**영향**: LLM이 잘못 호출했을 때 디버깅 불가능. nurikun MCP 도구들은 동일 상황에서 "X is required" 명시(좋음). crawl-ai/marketer 도구는 미검증.

**제안 수정**:
```go
func handleResearch(...) (*mcp.CallToolResult, error) {
    question, _ := args["question"].(string)
    if strings.TrimSpace(question) == "" {
        return mcp.NewToolResultError("question is required"), nil
    }
    // ...
}
```
crawl-ai의 fetch/search/research 3개 + marketer의 draft_campaign에 동일 패턴 적용 필요.

### F4 · [MEDIUM] crawl-ai `search`가 "no index"와 "0 matches"를 구분 못 함
**증거**: `mcp__musu-crawl-ai__search` (arg 없이) → "Found 0 results:". 인덱스 파일이 존재하지 않아도 빈 결과처럼 보임 → 사용자가 "데이터 없네"로 오해 가능.

**제안 수정**: 인덱스 파일 존재 여부를 먼저 확인하고, 없으면 `"index not found at <path> — run 'fetch' first"` 형태의 명시적 메시지.

### F5 · [MEDIUM] DoctorResult JSON 응답이 CamelCase envelope
**증거**: `doctor` 응답 = `{"Report": {...}, "Blocking": true, "ActionableFix": "..."}`. 안쪽 `Report`는 snake_case(json tag 있음), 바깥 envelope는 Go struct field name 그대로 노출.

**제안 수정**: `DoctorResult` struct에 json tag 추가 (`Report → "report"`, `Blocking → "blocking"`, `ActionableFix → "actionable_fix"`). ecosystem 전체에서 snake_case JSON envelope 일관성.

### F6 · [LOW] MCP 서버에 secret/config 전달 경로 부재
**현 상태**: MCP 서버가 Claude Code env를 상속 → 사용자의 `NURIKUN_GMAIL_TOKEN`, `NURIKUN_UNSUB_SECRET` 등 환경변수가 자동 전달되지 않음(Claude Code가 명시 setEnv 안 하면). 결과: doctor가 mailbox unconfigured로 reportt.

**제안 수정**: `claude mcp add` 시 `--env KEY=VAL` 옵션 사용 안내. 또는 README/AGENTS.md에 표준 등록 명령 예시 제공:
```powershell
claude mcp add -s user musu-nurikun "C:\path\musu-nurikun.exe" mcp `
  --env NURIKUN_GMAIL_CREDENTIALS=E:\nurikun\credentials.json `
  --env NURIKUN_GMAIL_TOKEN=E:\nurikun\token.json `
  --env NURIKUN_UNSUB_SECRET=...
```

### F7 · [LOW] AI 의존 도구 미검증 — Ollama 미구동
research / draft_campaign / fetch (LLM summary 사용 시) / nurikun triage·responder는 실제 호출 결과 검증 못 함. 이건 아키텍처 문제가 아니라 운영 의존성. SGLANG_GUIDE 또는 Ollama 띄우면 즉시 검증 가능.

---

## 3. 잘 작동한 것 (인정)

- **nurikun args 검증**: `create_list` / `messages_by_status` 같은 도구는 빠진 arg를 정확한 메시지로 reject ("name is required", "status is required (received|drafted|sent|escalated)"). 이 패턴이 crawl-ai/marketer로 확산되면 좋음.
- **doctor 도구**: 호출 한 번으로 의미 있는 상태 리포트 + actionable fix 반환. 진짜 유용한 MCP 도구의 모범 사례.
- **MCP 서버 시작 신뢰성**: 세 서버 모두 `claude mcp list` health-check 통과, stdio JSON-RPC 정상 응답(parse error 케이스도 정확히 처리).
- **`search` graceful**: 인덱스가 비었어도 크래시 X (F4와 trade-off지만 안정성은 좋음).

---

## 4. 권장 우선순위

1. **🔥 F1 (스키마 보강)** — 가장 영향 큼. 13 도구 정의에 `WithString`/`WithNumber`/`Required` 추가. 약 30분 작업, MCP를 진짜 쓸 수 있게 만드는 잠금 해제.
2. **F3 (빈 입력 가드)** — F1과 함께 같은 PR로. 4 핸들러에 한두 줄씩.
3. **F2.1 (DB MkdirAll)** — 3줄 패치, nurikun/marketer DB 자동 생성. 가장 cheap한 UX 개선.
4. **F2.2 또는 F6 (env/path 주입)** — README에 표준 `claude mcp add --env ...` 예시. 코드 변경 없음, doc만.
5. **F5 (snake_case envelope)** — DoctorResult struct tag. 한 줄씩.
6. **F4 (search "no index" 구분)** — 2-3줄.

전체 합쳐도 **반나절 이내** 작업. 이후 무수 ecosystem은 다른 LLM 에이전트가 호출하는 1급 도구로 완성됨.

---

## 5. 다음 단계 제안

이 감사 결과를 각 레포 `NEXT_STEPS.md`에 1줄씩 반영(P1으로 F1, P2로 F3, P3로 F2.1):

- `musu-crawl-ai/NEXT_STEPS.md` ← MCP tool params + research 빈입력 가드
- `musu-marketer/NEXT_STEPS.md` ← MCP tool params + draft_campaign 빈입력 가드
- `musu-nurikun/NEXT_STEPS.md` ← MCP tool params (validation은 이미 있음, schema만 보강) + db.NewStore MkdirAll

위 PR 한 번이면 실사용 MCP 점수가 "거의 못 씀"에서 "다른 에이전트가 도구로 잘 활용"으로 점프.

---

**작성**: 2026-05-28, Claude Opus 4.7 (1M context) MCP 직접 호출 + 결과 캡처 기반.
**검증 방법**: 본 문서의 결론은 모두 실제 MCP 호출 + Bash 검증으로 재현 가능.
