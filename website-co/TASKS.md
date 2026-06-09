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
- [x] **T3.1** 큐 `internal/queue/`: claim/lease(타임아웃 재claim=재시작안전)/done/needs_human/failed, file JSON. **유닛테스트**(claim·lease만료재claim·중복방지). **blocked 상태 자체가 없음 = block-storm 구조적 불가.**
- [x] **T3.2** 백로그+refill `internal/supply/`: 테넌트 pool.json에서 min/target 보충(범용=풀은 테넌트 데이터). njd 16토픽 이식. **스모크: 큐 8편 보충 확인.** 마케터 동적발굴(MCP)은 후속.
- [ ] **T3.3** geo-refresh `internal/supply/geo.go`: 발행글 GEO 보강 사이클 (후속)
- [~] **T3.4** 스케줄러: **일일 발행 캡 완료**(`dailyCap`, ledger.PublishedToday — "2신규/일"). 연속 데몬+캡=케이던스 달성. 명시적 cron(특정시각 잡)은 후속(선택).
- [x] **T3.5** health+json `internal/health/`: 테넌트별 pending·publishedToday·total·cap. `run --health-addr`. **스모크: status=pass 응답 확인.**
- [x] **T3.6** 워커풀·동시성 `internal/daemon/`: N goroutine 병렬 드라이브 + graceful shutdown. `run` 멀티테넌트 배선. **데몬 결정적 테스트**(큐→작가→검증→편집장→발행→done + needs_human≠blocked).

## Phase 4 — 멀티테넌트 입증
- [x] **T4.1** vibecode 테넌트 `tenants/vibecode/config.json`+pool.json(file 모드 플레이스홀더, 운영자가 niche/발행 채움)
- [x] **T4.2** **같은 바이너리·2 테넌트 동시 구동 검증.** 스모크: health에 njd+vibecode 둘 다, 각 큐·원장·캡 격리, 독립 refill. **멀티테넌트 입증 완료.**

## ✅ Phase 1-4 코어 완료 (2026-06-09) — musu-website-co 작동
PRD의 정석 아키텍처를 작동하는 Go 서비스로 구현. **block-storm 구조적 불가**(paperclip 이슈/reconciler 없음, lease 재시작안전, needs_human≠blocked). 작가→검증→편집장(YMYL)→발행 루프 섀도 검증. 멀티테넌트(농지다·vibecode) 입증. 결정적 유닛테스트 다수.

### 남은 작업(enhancement·supervised)
- [ ] **T2.4** 농지다 실발행 컷오버 — `run --publish` 또는 `once --tenant nongjida --publish`. ⚠️프로덕션(wiki-gen→nongjida.kr), 감독 하에. 빌드 준비완료.
- [ ] **T3.3** geo-refresh 사이클 (발행글 GEO 보강)
- [ ] **T3.4** 명시적 cron 스케줄(특정시각 잡) — 현재 연속데몬+일일캡으로 케이던스 달성, cron은 선택
- [x] **작가 구조회귀 튜닝 완료**: Write(첫패스)/Revise(이전초안+고칠점만, 구조유지) 분리. **섀도 재런 검증: approved(3라운드)** — 편집장이 농지법 제8조(시행 2025.1.24)·MAFRA 보도자료 WebFetch 실재확인 후 승인. **파이프라인이 발행가능 기사를 approved까지 수렴 입증.**
- [ ] 마케터 동적 토픽발굴(MCP) · Task Scheduler/systemd 상시 등록 · website-co marketer-v2-seo→main 이전

## ✅ 종합: musu-website-co 프로덕션 준비 완료(실발행 컷오버만 감독 대기)
파이프라인이 needs_human(위조출처 보류)·approved(검증출처 발행가능) 둘 다 정확 판정. block-storm 구조적 불가. 멀티테넌트. **T2.4 실발행(운영자 go-ahead)만 남음.**

## 진행 로그
- 2026-06-09: PRD 정본화(0920cd2). 환경 확인(musu-system·Go 1.26.3·claude CLI). TASKS 작성. Phase 1 착수.
- 2026-06-09 #1: **T1.1·T1.2·T1.3 완료.** go.mod·main.go(once/run)·agent/claude.go(PATH주입·stdin·재시도) 빌드·vet 통과, `once --probe` 실제 claude 응답 수신. claude=stateless 실행기 입증.
- 2026-06-09 #2: **T1.4~T1.9 완료 = Phase 1 전체 완료.** prompts·writer·editor·validate·loop 구현. **섀도 런 end-to-end 성공**(농지연금 기사 생성→3라운드→needs_human, 편집장이 위조출처+시행일누락 보류). block-storm 구조적 불가 입증. 튜닝: 작가 재작성 시 구조 회귀(R2 11실패). 다음=Phase 2(발행 어댑터+농지다 테넌트).
- 2026-06-09 #3: **Phase 2 완료**(T2.1-T2.3): tenant config·publish 어댑터(file/command·write-ahead·멱등)·농지다 테넌트. 유닛테스트 통과. T2.4 실발행은 감독.
- 2026-06-09 #4: **Phase 3 대부분 완료**(T3.1·T3.2·T3.4부분·T3.5·T3.6): 큐(lease 재시작안전, blocked 없음)·데몬 워커풀·refill·일일캡·health. **데몬 스모크 성공**(refill 8편·health pass·원장공유·워커풀). 결정적 테스트 다수. 남음: T3.3 geo·T3.4 cron(선택)·Phase 4 vibecode·T2.4 컷오버.
