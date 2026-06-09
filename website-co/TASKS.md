# TASKS — musu-website-co (Go) 구현 체크리스트

> PRD: `musu-for-njd/docs/specs/musu-website-co-PRD.md` (커밋 0920cd2). 30분 단위 원자작업. 진행하며 체크.
> 범례: [ ] 미착수 · [~] 진행중 · [x] 완료 · 각 작업 끝 = 컴파일/테스트 통과.

## Phase 1 — PoC: claude=stateless 실행기 + 콘텐츠 루프 (섀도, 발행 안 함)
- [x] **T1.1** 스켈레톤: `go.mod` + go.work 등록 + `main.go`(flag CLI: `once`/`run`). `go build`·`go vet` 통과. (cobra/core는 후속, 1차 stdlib)
- [x] **T1.2** agent exec 래퍼 `internal/agent/claude.go`: `Run(ctx, system, user)` — `claude -p`, **PATH 주입**(%APPDATA%\npm), 프롬프트 stdin(긴 입력 안전), 타임아웃, 지수백오프 재시도
- [x] **T1.3** 래퍼 실측: `once --probe` → 실제 claude 응답 수신 OK ("나는 Claude Code…"). **claude=stateless 실행기 작동 입증.**
- [x] **T1.4** 프롬프트 이식: `internal/prompts/`(writer.md·editor.md go:embed, {{BRAND}}/{{SLUG}} 치환). stateless 적응(작가=md만, 편집장=JSON만), YMYL·평이한국어·5섹션 게이트 보존
- [x] **T1.5** writer 패스 `pipeline/writer.go`: `Write(topic, feedback)` → 마크다운(코드펜스 방어)
- [x] **T1.6** editor 패스 `pipeline/editor.go`: `Review(draft)` → `Verdict{decision,changes,notes}`, JSON 추출 파싱
- [x] **T1.7** validate 게이트 `pipeline/validate.go`: 5섹션·tags4·.go.kr출처·"내농지"·"확인할것"·단정금지어 — 핵심 규칙 Go 포팅
- [x] **T1.8** loop `pipeline/loop.go`: writer→validate→editor→수정루프(캡)→approved|**needs_human(blocked 아님!)**. 상태 Result로 호출자 소유
- [x] **T1.9** 섀도 실행 `once --topic`: **end-to-end 검증 성공.** 실제 기사 생성→작가/검증/편집장 3라운드→needs_human(편집장이 위조의심 출처+시행일누락 정확히 보류). **block-storm·reconciler·오실레이션 전무 입증.** YMYL 게이트 작동. ⚠️튜닝 여지: R2 검증실패 11건(작가가 재작성 시 구조 일부 누락=회귀) — 피드백 주입 시 "기존 구조 유지" 강조 필요.

## ✅ Phase 1 완료 (2026-06-09) — 정석 아키텍처 PoC 입증
claude=stateless 실행기로 작가→검증→편집장→수정루프 완주, paperclip 이슈/reconciler 없음 → **block-storm 구조적 불가**. 편집장 YMYL 게이트 보존(위조출처 차단). 미수렴=needs_human(≠blocked).

## Phase 2 — 발행 + 농지다 테넌트
- [x] **T2.1** 테넌트 config 로더 `tenant/config.go`: Load/LoadAll, Publish(file/command+cwd), 검증. 멀티테넌트.
- [x] **T2.2** 발행 어댑터 `publish/`: frontmatter 파서 + file/command 모드 + 원장(published-approved.json 호환) + **write-ahead intent**. **유닛테스트 통과**(파싱·file발행·멱등, 프로덕션 무관)
- [x] **T2.3** 농지다 테넌트 `tenants/nongjida/config.json`: brand=농지다, publish=command(wiki-gen api/ cwd), 기존 원장 공유. once `--tenant`·`--publish` 배선
- [ ] **T2.4** 농지다 컷오버: `once --tenant nongjida --publish` 실발행 1편. ⚠️**프로덕션(wiki-gen→nongjida.kr)이라 감독 발행** — 빌드·경로 준비 완료, 실행만 남음(승인 시)

## Phase 3 — 토픽공급 + 스케줄 + 헬스 (musu-njd 흡수)
- [ ] **T3.1** 큐 `internal/queue/`: claim/lease(타임아웃 재claim)/done/needs_human, 1차 파일(JSON) — block-storm 불가 입증
- [ ] **T3.2** 백로그+refill `internal/supply/`: 큐레이트 풀 + 마케터(MCP/HTTP) min/target
- [ ] **T3.3** geo-refresh `internal/supply/geo.go`: 발행글 GEO 보강 사이클
- [ ] **T3.4** 내장 스케줄러 `internal/schedule/`: cron(테넌트별 잡) — 수요·refill·발행·헬스
- [ ] **T3.5** health+json + 데드맨 `internal/health/`
- [ ] **T3.6** 워커풀·동시성: N goroutine 병렬 토픽 드라이브 + graceful shutdown

## Phase 4 — 멀티테넌트 입증
- [ ] **T4.1** vibecode 테넌트 `tenants/vibecode/config.json`
- [ ] **T4.2** 같은 바이너리·두 테넌트 동시 구동 검증

## 진행 로그
- 2026-06-09: PRD 정본화(0920cd2). 환경 확인(musu-system·Go 1.26.3·claude CLI). TASKS 작성. Phase 1 착수.
- 2026-06-09 #1: **T1.1·T1.2·T1.3 완료.** go.mod·main.go(once/run)·agent/claude.go(PATH주입·stdin·재시도) 빌드·vet 통과, `once --probe` 실제 claude 응답 수신. claude=stateless 실행기 입증.
- 2026-06-09 #2: **T1.4~T1.9 완료 = Phase 1 전체 완료.** prompts·writer·editor·validate·loop 구현. **섀도 런 end-to-end 성공**(농지연금 기사 생성→3라운드→needs_human, 편집장이 위조출처+시행일누락 보류). block-storm 구조적 불가 입증. 튜닝: 작가 재작성 시 구조 회귀(R2 11실패). 다음=Phase 2(발행 어댑터+농지다 테넌트).
