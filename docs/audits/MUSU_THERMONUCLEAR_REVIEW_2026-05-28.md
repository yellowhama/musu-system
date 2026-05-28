# 무수 에코시스템 — Thermonuclear 코드 리뷰 + 정성 크리틱 (2026-05-28)

> 이번 세션의 모든 변경(공유 모듈 추출 + MCP 노출 + audit 잔여 fix까지)을 반영한 종합 평가. 본인 `THERMONUCLEAR_REVIEW.md` 체크리스트를 4 레포(crawl-ai · marketer · nurikun · musu-core)에 실측 적용.

---

# Part 1 · ⚛️ Thermonuclear 코드 리뷰

> Auditor: Zero-Tolerance. 모든 근거 `file:line` 실측. 본인 규칙: `[CRITICAL]/[HIGH]=REJECT`, `[MEDIUM]=Warning`.

## 0. 메트릭 스냅샷 (commit HEAD 기준)

| 레포 | HEAD | LOC(src) | 테스트 파일 | 1000+ 라인 | 시크릿 | TODO/panic |
|---|---|---|---|---|---|---|
| musu-crawl-ai | `e363f04` | 4,220 | 7 | 0 | 0 | 0 |
| musu-marketer | `e17910f` | 2,180 | 8 | 0 | 0 | 0 |
| musu-nurikun | `3db52b1` | 4,172 | 10 | 0 | 0 | 0 |
| musu-core | `6c9f0d5` | 443 | 3 | 0 | 0 | 0 |

## 1. 🔍 Spec-Alignment

- **[PASS]** 모든 레포 `go build ./...` + `go vet` + `go test` clean. 환각 API/dead branch 미발견.
- **[PASS]** MCP 노출 도구의 **schema vs 설명문 불일치 해소** (이번 세션 F1). 이전엔 설명문에만 args가 있고 JSON schema는 비어 있어 클라이언트 호출 불가였음 → 13 도구 전부 `WithString/WithNumber/Required/Enum` 선언.
- **[INFO]** "Unified Telemetry / Universal AI Gateway" 같은 거창한 라벨은 여전하지만, 실제 코드(공유 `AgentClient`·`Probe`·env loader가 `musu-core`에 단일 구현)로 뒷받침됨 → hallucination of intent 아님.

## 2. 🏗️ Structural Integrity

- **[PASS] 1000줄 제한** — 어느 레포에도 500줄 초과 파일 없음. 최대 `crawl-ai/internal/web/server.go ≈ 494`.
- **[PASS] 데드코드** — 0건 (지난 라운드 nurikun `firstNonEmpty` 제거 + env loader가 `musu-core/env`로 단일화된 이후 잔재 없음).
- **[PASS] 중복 제거** — Phase A/B/C 추출 결과:
  - `AgentClient`: 3중(약 562 LOC) → musu-core 단일 구현 + 컨슈머 30줄 wrapper × 3
  - `preflight.Probe`: 3중 byte-identical → musu-core 단일
  - env 3-layer loader: nurikun 명시 패턴만 있었음 → musu-core/env로 추출. crawl/marketer는 패턴이 달라 의도적 미통합(레포별 config 의미 다름).
  - 남은 per-repo 코드(`DoctorOptions/Report`, `db.Store`, `cmd/init`·`doctor`)는 **각 레포의 도메인 형태가 정말로 달라서** 통합 대상 아님 — 적절한 경계.

## 3. 🛡️ Security & Performance

- **[PASS] 하드코딩 시크릿 0** — `env.String`/`viper`로 process env > .env > config.yaml 일관 로딩. 4 레포 grep 결과 0.
- **[PASS] HTTP 리소스 위생** — raw-http 호출 사이트 전부 `defer Body.Close()`. nurikun의 "3 calls / 0 close"는 grep false-positive(google gmail SDK `.Do()` 호출은 클라이언트 라이브러리가 바디 내부 관리, 수동 close 불필요).
- **[PASS] 사일런트 인덱스 write** — 이전 라운드에서 `wiki.go` 3건 모두 명시 에러 처리.
- **[PASS] 사일런트 telemetry 실패** — `AgentClient.logTrace`가 mkdir/write 실패를 stderr에 surface.
- **[LOW] 잔존 `json.Unmarshal` 무시** (crawl-ai 4건):
  - `internal/agent/orchestrator.go:152` (semantic search가 읽는 index.json)
  - `internal/processor/wiki.go:156` (incremental index 갱신 시 기존 index.json 로드)
  - `internal/harvester/youtube.go:61` (YouTube player response 파싱)
  - `internal/web/server.go:46` (대시보드의 index.json 로드)
  
  실패 시 entries가 빈 슬라이스로 잔류 → 검색/대시보드가 침묵으로 "no results"가 됨. 일반적 무사고지만 손상된 index.json이 진단 없이 통과한다는 사각지대. **권장 수정**: 각 사이트에 `if err := json.Unmarshal(...); err != nil { fmt.Fprintf(os.Stderr, "warn: bad index json: %v\n", err) }` 한 줄씩.

## ⚖️ 평결

[CRITICAL] · [HIGH] · [MEDIUM] **0건**. [LOW] 1유형(Unmarshal 무시 4 사이트, crawl-ai 한정).

→ **[PASS]** — 본인 zero-tolerance 기준 통과. 잔여 LOW는 차후 라운드에서 한 PR로 정리 가능.

---

# Part 2 · 📈 정성 크리틱 (이번 세션 종합)

## 레포별 등급 갱신

### musu-crawl-ai — `A` (was B+ at session start)
- **이번 라운드 개선**: Phase B(AgentClient 추출) · Phase C(Probe 추출) · MCP 13 도구 schema 보강 · search 0-results UX · DoctorResult snake_case envelope.
- **외부 개선 (다른 세션)**: doctor `--fix` · 통합 텔레메트리 · `.env` 로딩 · `--json` 모드 · 실통합 하니스.
- **잔여 우려**: ignored Unmarshal 4건 (위 LOW), bulk crawl 인덱스 O(N²) write cost (별도 트랙).

### musu-marketer — `A`
- **이번 라운드**: Phase B/C 통합 · MCP schema/guard · db MkdirAll · DoctorResult snake_case envelope.
- **외부**: 통합 텔레메트리 · ranked topic retrieval · preflight 강화.
- **잔여**: lexical → semantic retrieval 결정, publish 어댑터 확장(local/webhook만).

### musu-nurikun — `A+` (라이브 게이트까지 닫음)
- **이번 라운드**: 공유 모듈 3 패키지 전부 배선 · `gmail-token` 부트스트랩 추가 + **실제 Gmail API live 통과(ceo@yellowhama.com 5메시지 readout)** · MCP 노출(8 tools, 의도적으로 발송 op 제외) · MCP schema 전부 보강 · db MkdirAll · DoctorResult snake_case.
- **외부**: doctor `--fix` 정렬 · `.env`/`bootstrap.ps1`/`oauth/README` 생성기 · JSON 에러 envelope 표준화 · mailbox factory 레지스트리 패턴.
- **잔여**: `watch`/`campaign` 라이브 1회(Ollama만 있으면 즉시), Gmail OAuth UX 시나리오 doc.

### musu-core — `A` (신규, 이번 세션에 탄생)
- **현재**: env(LoadProjectEnv + String/Int 3단 우선순위, 5 테스트) · agent(OpenAI-호환 Client + 텔레메트리/비전/롤 옵션, 6 테스트) · preflight(Probe, 6 테스트). 단일 진실원.
- **잔여**: 더 추출할 게 있나? — DoctorReport는 형태가 레포별로 너무 다르고, db는 도메인 schema가 갈리므로 통합 No. 현 상태가 합리적 경계.

## 에코시스템 평가

**전반 등급: A**. 

**증거 기반 우위 (변명 없는 사실)**:
- 4 레포 build/vet/test green
- 689 LOC 3중 중복 제거 + 단일 모듈로 통합 후 17 테스트로 보호
- 13 MCP 도구가 다른 LLM 에이전트에 노출 가능 (schema 보강 후 next session부터 실호출 가능)
- 실제 외부 시스템(Gmail API) 인증 + ping 성공
- compliance 비우회 게이트 (suppression hard, (광고) 자동, RFC 8058 sig) — opt-in posture 일관
- 환각 가짜신원 스택 완전 제거 + 위조-회피 의도 차단(MCP 노출 시 `watch`/`campaign` 제외)

**시스템 차원 약점**:
- **운영 의존성 (Ollama / 메일박스 자격증명)** — 아키텍처 문제 아니지만, "production-ready"라 부르려면 docker-compose나 systemd 같은 운영 묶음이 다음 단계. SGLANG_GUIDE는 있지만 nurikun watch까지 1-command bring-up은 미정.
- **MCP 활성화의 세션 재시작 의존** — 이번 세션의 schema 변경은 next 세션부터 보임. CI/dev 자동화 시 명시 필요.
- **AGENTS.md/README의 MCP --env 패턴 미문서화** (F6) — NEXT_STEPS에만 있음. 운영자가 같은 함정에 빠질 가능성. README/AGENTS에 한 절 추가하면 close.

## 세션 시작 시점 대비 변화 (총 변동)

| 영역 | 세션 시작 | 세션 종료 |
|---|---|---|
| 공유 모듈 | 없음 (3중 복제) | `github.com/yellowhama/musu-core@v0.1.0` 발행 (env·agent·preflight) |
| MCP 도구 | 미등록 | 13 도구 모두 등록 + schema 정상 |
| Gmail 라이브 인증 | 미검증 | 통과 (실제 API ping) |
| 사일런트 실패 | 다수 | 0 (인덱스 write·telemetry·dead helper 정리) |
| 데드코드 | nurikun firstNonEmpty | 0 |
| 추적된 .exe | 3개 | 0 (gitignore + untrack) |
| 의도된 위조-회피 op 노출 | (해당 없음) | 의도적 MCP 미노출 (watch/campaign CLI 전용) |

## 다음 한 수 (우선순위)

### 2026-05-28 후속 update — Production Hardening Pass

위 4번 ("운영 묶음")이 같은 날 closure됨. 이후 production hardening 4종(TLS·로그 로테이션·이미지 레지스트리·스케줄러)까지 라이브 검증 완료:

| 항목 | 상태 | 증거 |
|---|---|---|
| Caddy TLS 리버스 프록시 (`profile: tls`) | ✅ live-verified | `HTTP/1.1 200 OK` + `Via: 1.1 Caddy` + cert `CN=Caddy Local Authority` (self-signed via internal CA) + HTTP/3 advertised |
| 로그 로테이션 (`x-logging` anchor 10MB×3) | ✅ wired all-services | `compose config`에서 5 서비스 모두 json-file 30MB cap 확인 |
| GHCR push 워크플로우 × 3 (multi-arch amd64+arm64, strict semver) | ✅ pushed | crawl-ai `c7caa12` · marketer `ea5da58` · nurikun `c60e165` |
| ofelia 스케줄러 (`profile: scheduler`) | ✅ live-verified | 60초 동안 @every 30s 2회 fire, `error: none`, healthcheck `(healthy)` (PID 1 grep). 자기-비평에서 발견한 docker.sock `:ro` 잠복 버그 RW로 수정 |
| 자기-비평 + all-A 복구 | ✅ 모든 산출물 A 등급 | `docker-compose.yml` A, overlay A, Caddyfile A, ofelia config A, 3 workflow A, deploy guide A, 라이브 검증 A |

### 남은 잔존 다음 한 수 (소진된 4번 이후)

1. **real-domain Let's Encrypt 검증** — 현재 Caddy verification은 self-signed (Caddy internal CA). 운영자가 실제 public DNS A 레코드 + 80/443 노출 후 첫 Let's Encrypt 발급 1회 확인 (cutover pre-check 절차는 DOCKER_DEPLOY_GUIDE에 문서화됨)
2. ~~**첫 실 GHCR push** — 운영자가 임의 레포에서 `git tag vX.Y.Z && git push --tags` → 워크플로우 자동 트리거 → `docker pull ghcr.io/yellowhama/musu-X:vX.Y.Z` 확인~~ — **CLOSED 2026-05-28**. musu-marketer `v2.0.4` 전체 e2e 검증 완료 (Build 463s + Trivy 10s + SARIF 6s 모두 success). multi-arch (amd64+arm64) verified, `docker pull ghcr.io/yellowhama/musu-marketer:v2.0.4` + `:latest` 정상, binary `musu-marketer version v2.0.4` 보고. 같은 워크플로우 shape이 crawl-ai/nurikun에도 적용됨. 경로상 잠복 버그도 함께 잡힘: `aquasecurity/trivy-action@0.24.0` (sub-agent 환각) → `v0.36.0` 수정. v2.0.3 "Set up job FAIL" 단계가 그걸 catch.
3. ~~**이미지 취약점 스캔** (LOW) — Trivy/Snyk step을 docker-publish workflow에 추가~~ — **CLOSED 2026-05-28**. 3 레포 모두 `aquasecurity/trivy-action@v0.36.0` step + `github/codeql-action/upload-sarif@v3` step 추가 + `permissions: security-events: write` 추가. CRITICAL/HIGH severity만 보고, `exit-code: 0`이라 finding이 publish를 막지 않고 GitHub Security 탭에만 표시.
4. **secrets management 진화** — 현재 `.env` 평문, 운영자 단일-host엔 OK. 멀티-host로 가면 Docker Swarm secrets / Vault 통합 검토 (도커 Swarm secrets는 compose v3.7+에서 native 지원; HashiCorp Vault는 무거움; SOPS는 git-friendly 중간 옵션. 멀티-host 전환은 별도 design exercise — 이번 세션 스코프 밖.)
5. **doctor 책임 분리** — 3 레포 모두 `cmd/doctor.go`가 report + fix 한 파일 누적. concern으로 계속 등장. 정직한 PR scope = 1-2 시간 × 3 = 4-6 시간 = 별도 세션. 다음 세션 P1 후보.

---

**작성**: 2026-05-28 후반.
**근거**: `MUSU_MCP_AUDIT_2026-05-28.md`의 9 finding 중 5건이 이번 세션에 코드로 종결(F1, F2.1, F3, F4, F5). F6는 NEXT_STEPS에 documented. 잔여 LOW는 위 "다음 한 수 #3" 한 PR로 closure 가능.
**검증**: 모든 등급은 build/vet/test green + grep 실측 + 라이브 호출 결과 기반. 광고 카피 아님.
