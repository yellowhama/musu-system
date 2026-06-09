# musu-website-co

AI 웹사이트 운영회사 — 단일 멀티테넌트 Go 데몬. musu-njd 케이던스 + paperclip 오케스트레이션을
**한 서비스로 통합**하고, work를 paperclip 이슈로 두지 않아 **reconciler/block-storm을 구조적으로 차단**한다.
claude는 stateless 실행기(stdout 반환), Go가 상태·파일·발행을 소유. 사이트는 **테넌트 config(유저 데이터)**.

> 정본 PRD: `musu-for-njd/docs/specs/musu-website-co-PRD.md` · 진행: `TASKS.md` · 메모리: [[project-musu-website-co-go-service]]

## 구조
```
main.go                 once(단발) / run(데몬) CLI
internal/
  agent/    claude -p 실행기(PATH주입·stdin·재시도)
  prompts/  작가·편집장 시스템 프롬프트(go:embed, paperclip AGENTS.md 이식)
  pipeline/ writer→validate→editor→loop (needs_human≠blocked)
  tenant/   멀티테넌트 config 로더
  publish/  frontmatter·file/command 어댑터·원장(write-ahead intent·멱등)
  queue/    토픽 큐(claim/lease 재시작안전, blocked 없음)
  supply/   pool.json 백로그 refill
  daemon/   워커풀 루프(refill·일일캡·claim→드라이브→발행|needs_human)
  health/   health+json
tenants/<site>/config.json + pool.json
```

## 사용
```
website-co once --probe                          # claude 실행기 점검
website-co once --topic "<토픽>"                 # 1토픽 섀도(발행 안 함)
website-co once --tenant nongjida --publish      # 1토픽 실발행(승인 시)
website-co run --tenants tenants --health-addr :8088          # 데몬(섀도)
website-co run --tenants tenants --publish --health-addr :8088  # 데몬(실발행)
```

## 왜 block-storm이 불가능한가
paperclip은 work를 "이슈"로 표현하고 reconciler가 in_progress+에이전트할당+활성run없음 이슈를 `blocked`로
에스컬레이션한다(버그 #3882, 재시작 시 다발). 여기선 work가 우리 **큐 항목**이고 in_progress는 **lease**로
관리(타임아웃 재claim=재시작안전). 외부 reconciler가 없고, 미수렴은 `blocked`가 아니라 `needs_human`이다.
