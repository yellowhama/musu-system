# musu-marketer v2 — Commercial-grade Marketing Agent (SPEC)

> 상태: **DRAFT (Sprint 1 진입)** · 작성 2026-05-29
> 상위 목표: nongjida.kr **유저 10,000명**을 *건강한 방법*으로 유치.
> 이 문서는 v2의 계약(contract). v1 `SPEC.md`(draft CLI)는 그대로 하위호환 유지.

## 0. "건강한 방법"의 운영적 정의 (운영자 2026-05-29 재정의)

음식 비유: **패스트푸드까지는 OK. 길바닥 비위생 노점은 NO.**
판정선 = `합법 AND not-cringe`.

| 허용 (패스트푸드급) | 금지 (비위생 노점) |
|---|---|
| SEO 콘텐츠(출처 명시), paid ads(네이버/카카오/Meta), 리타게팅, 리드마그넷, 적극적·정직한 CTA, 옵트인 nurturing, 제휴/affiliate, 정통매체 기획기사 | 봇/자동카페/시드러닝/link farm, 가짜리뷰·testimonial, 페르소나 위조, **보이스피싱식 압박·공포 카피**, 다크패턴, 동의 없는 콜드 발송, **출처 없는 법조문(할루시네이션)** |

→ 코드로 강제되는 두 가드: **(a) Compliance Gate** (정보통신망법 §50, nurikun이 이미 보유), **(b) Citation Gate** (이 v2가 신규 — 법률·사실 주장은 반드시 검증된 출처 인용).

## 1. v1 → v2 gap

v1 (v2.0.5, ~2180 LOC): wiki-grounded 단발 draft CLI (Strategist→Copywriter→Critic), local/webhook publish. 캠페인/시퀀스/analytics/distribution 0.

v2가 더해야 할 layer:
```
┌─ Strategy Agent ───────── 채널 mix · 페르소나 세그먼트 · 캠페인 캘린더
├─ Content Agents ───────── SEO blog · 알림톡 · email seq · ad copy · PR pitch
│     (각자 Citation Gate + Compliance Gate 통과 필수)
├─ Distribution Adapters ── 네이버블로그 · 카카오채널 · YouTube desc · (paid: opt-in)
├─ Knowledge Asset RAG ──── crawl-ai wiki + 법제처 chunk (grounding/citation 원천)
├─ Compliance Gate ──────── 정보통신망법 §50 (nurikun 재사용)
├─ Citation Gate ────────── 사실/법률 주장 ↔ 출처 강제 (strict: 미인용 시 차단)
├─ Analytics + A/B ──────── 자기 데이터 기반 (외부 추적 없음)
└─ Funnel Orchestrator ──── subscribe→confirm→digest→conversion (nurikun 통합)
```

## 2. 설계 원칙 (불변)

1. **0 외부 LLM API default** — Ollama 우선. (NJD가 Gemini/Groq 별도 호출은 무관.)
2. **출처 없는 사실 주장 = 발송 불가** — Citation Gate strict 기본 ON.
3. **outbound는 항상 human-in-the-loop** — auto-send는 옵트인 + 운영자 결재 후만.
4. **delivery ops는 MCP로 노출 금지** — nurikun watch/campaign/serve는 CLI/내부HTTP only.
5. **paid 채널은 opt-in 모듈** — zero budget에서 organic-only로 완전 작동.

## 3. Sprint 우선순위 (10k leverage 순)

| Sprint | 산출물 | LOC | 의존 |
|---|---|---|---|
| **S1** ✅ | SEO 블로그 generator + **Citation Gate** | ~600 | wiki bridge (보유) |
| S2 | 카카오 알림톡 channel adapter + funnel | ~400 | nurikun internal HTTP (v0.4.0) |
| **S3** ✅ | persona variant generator (4 페르소나) — `--persona`/`--all-personas`, S1 출처+게이트 재사용 | ~300 | S1 |
| **S4** ✅ | 농민신문 PR pitch generator — `pr <keyword> --outlet/--all-outlets`, draft-only·human-in-loop, S1 게이트 재사용 | ~200 | S1 |
| **S5** ✅ | 변호사 leads referral matcher — `referral --leads --lawyers`, 결정론 매칭(전문분야 필수·지역 부스트·긴급도·capacity), draft-only. **exported 파일만**(live subscriber X) | ~300 | exported leads |
| **S6** ✅ | acquisition funnel planner — `plan <keyword...>`, 목표→투명한 퍼널 수치 + 캠페인 캘린더(seo/persona/pr/opt-in/nurture 오케스트레이션). 결정론·LLM 불요 | ~300 | S1·S3·S4 |

## 4. Sprint 1 계약 (SEO blog generator)

### 4.1 명령
```
musu-marketer seo <keyword> [--strict=true] [--min-words=900] [--persona=default] [--out=blog]
```

### 4.2 파이프라인
1. `WikiBridge.FindByTopic(keyword)` → grounding 소스 N개
2. 소스를 안정 라벨 `[S1]..[Sn]`으로 SourcePack 구성 (라벨↔소스ID/Title/Source 매핑)
3. **SEO Strategist** (LLM, JSON): `title`, `meta_description`(≤160자), `slug`, `target_keyword`, `secondary_keywords`, `outline`(H2/H3)
4. **SEO Writer** (LLM): outline 따라 longform 본문 생성. 규칙 — *모든 법률·수치·날짜·기관 주장 문장은 끝에 `[S#]` 인용*
5. **Citation Gate** (결정론적, LLM 아님):
   - 모든 `[S#]` 토큰이 SourcePack에 존재하는지 검증 (없는 라벨 = 항상 error)
   - "검증 가능한 주장" 문장 탐지: 법령 신호(`농지법`, `제\d+조`, `처분명령`, `이행강제금`, `시행령`…) / 수치(`\d{4}년`, `\d+개월`, `\d+%`) / 기관(`법제처`, `농림축산식품부`, `농지은행`) 포함
   - 주장 문장에 유효한 `[S#]`가 없으면 **uncited claim**으로 수집
   - `strict && len(uncited)>0` → **block (publish 차단, exit ≠0)**. 비strict면 경고만
6. 렌더: YAML frontmatter + schema.org `Article` JSON-LD + 본문 + `## 출처` 섹션(라벨→소스 역참조)
7. 저장: `projects/<project>/blog/<slug>.md`

### 4.3 검증 가능성
- Citation Gate · 렌더 · SourcePack은 **순수 함수**(LLM 무관) → 단위 테스트 100%.
- generator end-to-end는 Ollama 있을 때만 도는 통합 테스트로 분리.

### 4.4 성공 기준
- 검증된 출처 0개 → 명확한 error (draft.go의 "no verified knowledge" 패턴 재사용)
- 할루시네이션 법조문이 strict 모드에서 파일로 나가는 경우 0
- 산출 `.md`는 네이버/구글에 바로 붙일 수 있는 SEO frontmatter + JSON-LD 보유

## 5. 비목표 (Sprint 1)

- paid ads 집행(예산 의존) — S2 이후 opt-in 어댑터
- 자동 발송 — 본 sprint는 파일 산출까지만. 발송은 nurikun + 운영자 결재.
- 실시간 SERP 키워드 볼륨 조회(외부 API) — 후속. 지금은 운영자 제공 keyword.
