package kakao

import (
	"strings"
	"testing"
)

func TestDefaultTemplatesAreCompliant(t *testing.T) {
	for _, tmpl := range DefaultTemplates {
		if issues := Validate(tmpl); len(issues) != 0 {
			t.Errorf("default template %q has issues: %+v", tmpl.Code, issues)
		}
	}
}

func TestValidateCatchesPromoMarker(t *testing.T) {
	tmpl := Template{
		Code: "bad", Category: CategoryInformational, Trigger: "구독",
		Body: "#{name}님, 지금 할인 이벤트 중입니다!", Variables: []string{"name"},
	}
	issues := Validate(tmpl)
	if len(issues) == 0 {
		t.Fatal("expected promo markers to be flagged")
	}
	joined := ""
	for _, is := range issues {
		joined += is.Problem
	}
	if !strings.Contains(joined, "할인") || !strings.Contains(joined, "이벤트") {
		t.Errorf("expected 할인/이벤트 flagged, got: %s", joined)
	}
}

func TestValidateRequiresTriggerAndCategory(t *testing.T) {
	tmpl := Template{Code: "x", Category: "광고성", Body: "안내", Trigger: ""}
	issues := Validate(tmpl)
	var cat, trig bool
	for _, is := range issues {
		if strings.Contains(is.Problem, "정보성만 허용") {
			cat = true
		}
		if strings.Contains(is.Problem, "트리거") {
			trig = true
		}
	}
	if !cat || !trig {
		t.Errorf("expected category+trigger issues, got %+v", issues)
	}
}

func TestValidateVariableDeclarationMismatch(t *testing.T) {
	// Uses #{token} (undeclared) and declares "extra" (unused).
	tmpl := Template{
		Code: "v", Category: CategoryInformational, Trigger: "구독",
		Body: "#{name}님 확인 #{token}", Variables: []string{"name", "extra"},
	}
	issues := Validate(tmpl)
	var undeclared, unused bool
	for _, is := range issues {
		if strings.Contains(is.Problem, "token") {
			undeclared = true
		}
		if strings.Contains(is.Problem, "extra") {
			unused = true
		}
	}
	if !undeclared || !unused {
		t.Errorf("expected undeclared(token)+unused(extra), got %+v", issues)
	}
}

func TestValidateButtonURLVariableCounts(t *testing.T) {
	// #{token} used only in the button URL must count as used (declared -> ok).
	tmpl := Template{
		Code: "b", Category: CategoryInformational, Trigger: "구독",
		Body: "#{name}님 확정해주세요", Variables: []string{"name", "token"},
		Buttons: []Button{{Name: "확정", Type: "WL", URL: "https://x/confirm?t=#{token}"}},
	}
	if issues := Validate(tmpl); len(issues) != 0 {
		t.Errorf("button-URL variable should count as used, got %+v", issues)
	}
}

func TestRenderMarksScopeAndCompliance(t *testing.T) {
	out := Render(DefaultTemplates)
	for _, want := range []string{
		"kind: kakao-alimtalk-templates",
		"사전 등록·심사 승인",
		"광고성(친구톡)",
		"nurikun",
		"njd_optin_confirm",
		"✅ 컴플라이언스 통과",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}
