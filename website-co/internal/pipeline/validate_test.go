package pipeline

import "testing"

const goodDraft = `---
title: 제목
slug: x
category: 실전가이드
tags: [a, b, c, d]
published: false
---

## 무엇을 알아야 하나
출처: law.go.kr 참고.

## 왜 중요한가
내용.

## 내 농지에 미치는 영향
내 농지에 따라 다르다.

## 오늘 확인할 것
- 확인할 것: 서류.

## 출처
- https://law.go.kr
`

func TestValidate_Good(t *testing.T) {
	if f := Validate(goodDraft); len(f) != 0 {
		t.Fatalf("정상 초안인데 위반: %v", f)
	}
}

func TestValidate_MissingSection(t *testing.T) {
	bad := "## 무엇을 알아야 하나\n내용. law.go.kr tags: [a,b,c,d] 내 농지 확인할 것"
	f := Validate(bad)
	if len(f) == 0 {
		t.Fatal("섹션 누락인데 통과")
	}
}

func TestValidate_AssertionWord(t *testing.T) {
	d := goodDraft + "\n이건 반드시 그렇다."
	found := false
	for _, f := range Validate(d) {
		if f == "단정 표현 사용: 반드시" {
			found = true
		}
	}
	if !found {
		t.Fatal("단정 표현 '반드시' 미검출")
	}
}
