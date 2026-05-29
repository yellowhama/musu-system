package seo

import (
	"strings"
	"testing"
)

func TestPersonaBySlug(t *testing.T) {
	p, ok := PersonaBySlug("heir")
	if !ok {
		t.Fatalf("expected heir persona to exist")
	}
	if p.Name != "농지 상속인" {
		t.Errorf("heir Name = %q", p.Name)
	}
	if _, ok := PersonaBySlug("nope"); ok {
		t.Errorf("unknown slug must return ok=false")
	}
}

func TestPersonaSlugsCoverDefaults(t *testing.T) {
	if got := len(PersonaSlugs()); got != len(DefaultPersonas) {
		t.Fatalf("PersonaSlugs len = %d, want %d", got, len(DefaultPersonas))
	}
	// Every default persona must have the fields the prompt relies on.
	for _, p := range DefaultPersonas {
		if p.Slug == "" || p.Name == "" || p.Voice == "" || p.CTAAngle == "" {
			t.Errorf("persona %+v has empty required field", p)
		}
		if p.Slug != Slugify(p.Slug) {
			t.Errorf("persona slug %q is not URL-safe", p.Slug)
		}
	}
}

func TestPersonaPromptBlock(t *testing.T) {
	// Zero persona injects nothing (generic path).
	if (Persona{}).promptBlock() != "" {
		t.Errorf("zero persona must produce empty prompt block")
	}
	p, _ := PersonaBySlug("elderly-owner")
	blk := p.promptBlock()
	for _, want := range []string{p.Name, p.Pain, p.Voice, p.CTAAngle, "사실은 절대 바꾸지 말 것"} {
		if !strings.Contains(blk, want) {
			t.Errorf("prompt block missing %q:\n%s", want, blk)
		}
	}
}
