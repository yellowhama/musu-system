package seo

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// CitationReport is the deterministic verdict on an article body. It is produced
// without any LLM call so the result is reproducible and unit-testable.
type CitationReport struct {
	// InvalidLabels are [S#] tokens that reference a label NOT in the SourcePack
	// (i.e. the model invented a citation). These are ALWAYS fatal, even in
	// non-strict mode — a dangling citation is worse than no citation.
	InvalidLabels []string
	// UncitedClaims are sentences that make a checkable factual/legal claim but
	// carry no valid [S#]. Fatal only in strict mode.
	UncitedClaims []string
	// CitedClaims is the count of claim sentences that DID carry a valid citation.
	CitedClaims int
}

// OK reports whether the article may be published under the given strictness.
// Invalid labels always fail. Uncited claims fail only when strict.
func (r CitationReport) OK(strict bool) bool {
	if len(r.InvalidLabels) > 0 {
		return false
	}
	if strict && len(r.UncitedClaims) > 0 {
		return false
	}
	return true
}

// Err renders a human-readable blocking reason, or "" if OK.
func (r CitationReport) Err(strict bool) error {
	if r.OK(strict) {
		return nil
	}
	var parts []string
	if len(r.InvalidLabels) > 0 {
		parts = append(parts, fmt.Sprintf("존재하지 않는 출처 인용 %d건: %s",
			len(r.InvalidLabels), strings.Join(r.InvalidLabels, ", ")))
	}
	if strict && len(r.UncitedClaims) > 0 {
		preview := r.UncitedClaims
		if len(preview) > 3 {
			preview = preview[:3]
		}
		parts = append(parts, fmt.Sprintf("출처 없는 사실/법률 주장 %d건 (예: %q)",
			len(r.UncitedClaims), strings.Join(preview, " | ")))
	}
	return fmt.Errorf("citation gate 차단: %s", strings.Join(parts, "; "))
}

var citationToken = regexp.MustCompile(`\[(S\d+)\]`)

// claimSignals are the markers of a "checkable" claim — a SPECIFIC legal/factual
// proposition that must be backed by a verified source. The signals are
// deliberately precise: a specific statute reference (제10조), a quantity (1년,
// 25%), or a named authority (한국농어촌공사). This is exactly where hallucination
// lives, so this is exactly what the gate enforces.
//
// Bare topic words alone (농지법, 처분명령, 이행강제금) are NOT signals: a framing
// sentence like "처분명령을 받으면 당황스럽습니다" or "본 가이드는 처분명령을
// 설명합니다" asserts no specific fact and must not be forced to cite. The moment
// such a sentence states a specific (제11조, 6개월, 한국농어촌공사) it trips a
// precise signal and is gated again. This keeps the "no hallucinated law"
// guarantee while not blocking on topic mentions.
var claimSignals = []*regexp.Regexp{
	// Specific legal references
	regexp.MustCompile(`제\s*\d+\s*조`),
	regexp.MustCompile(`제\s*\d+\s*항`),
	regexp.MustCompile(`제\s*\d+\s*호`),
	regexp.MustCompile(`시행령`),
	regexp.MustCompile(`시행규칙`),
	regexp.MustCompile(`헌법`),
	regexp.MustCompile(`판례`),
	// Quantities / dates (factual specifics)
	regexp.MustCompile(`\d{4}\s*년`),
	regexp.MustCompile(`\d+\s*개월`),
	regexp.MustCompile(`\d+\s*제곱미터`),
	regexp.MustCompile(`\d+\s*만\s*제곱미터`),
	regexp.MustCompile(`\d+\s*%`),
	regexp.MustCompile(`\d+\s*퍼센트`),
	regexp.MustCompile(`\d+\s*분의\s*\d+`),
	regexp.MustCompile(`\d+\s*만\s*원`),
	regexp.MustCompile(`\d+\s*억`),
	// Named authorities
	regexp.MustCompile(`법제처`),
	regexp.MustCompile(`농림축산식품부`),
	regexp.MustCompile(`농식품부`),
	regexp.MustCompile(`농지은행`),
	regexp.MustCompile(`한국농어촌공사`),
}

// sentenceSplit breaks Korean/English prose into sentences. We split on sentence
// terminators (。.!?…) and newlines, which is coarse but sufficient: the gate
// only needs sentence-level granularity to decide "does this claim cite a source".
var sentenceSplit = regexp.MustCompile(`[.!?。…]+|\n+`)

// markdownStructural matches lines we never treat as prose claims: headings,
// list bullets that are pure structure, code fences, the references label.
var headingLine = regexp.MustCompile(`^\s{0,3}#{1,6}\s`)

// isClaim reports whether a sentence asserts a checkable factual/legal claim.
func isClaim(sentence string) bool {
	for _, sig := range claimSignals {
		if sig.MatchString(sentence) {
			return true
		}
	}
	return false
}

// validCitationsIn returns the valid [S#] labels found in a sentence, and any
// invalid ones (referencing labels absent from the pack).
func validCitationsIn(sentence string, pack SourcePack) (valid, invalid []string) {
	for _, m := range citationToken.FindAllStringSubmatch(sentence, -1) {
		label := m[1]
		if pack.HasLabel(label) {
			valid = append(valid, label)
		} else {
			invalid = append(invalid, label)
		}
	}
	return valid, invalid
}

// CheckCitations runs the deterministic citation audit over an article body.
//
// Algorithm:
//  1. Skip heading/code-fence lines (structure, not claims).
//  2. Split remaining prose into sentences.
//  3. Every [S#] token is checked against the pack — unknown labels are fatal.
//  4. A sentence that trips a claim signal but holds no valid [S#] is an
//     uncited claim.
func CheckCitations(body string, pack SourcePack) CitationReport {
	var rep CitationReport
	invalidSet := map[string]bool{}

	inFence := false
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence || headingLine.MatchString(line) {
			// Still scan headings/code for *invalid* labels (a fabricated cite
			// anywhere is a problem) but never count them as claims.
			_, invalid := validCitationsIn(line, pack)
			for _, l := range invalid {
				invalidSet[l] = true
			}
			continue
		}

		for _, sentence := range sentenceSplit.Split(line, -1) {
			s := strings.TrimSpace(sentence)
			if s == "" {
				continue
			}
			valid, invalid := validCitationsIn(s, pack)
			for _, l := range invalid {
				invalidSet[l] = true
			}
			if !isClaim(s) {
				continue
			}
			if len(valid) > 0 {
				rep.CitedClaims++
			} else {
				rep.UncitedClaims = append(rep.UncitedClaims, squeeze(s))
			}
		}
	}

	for l := range invalidSet {
		rep.InvalidLabels = append(rep.InvalidLabels, l)
	}
	sort.Strings(rep.InvalidLabels)
	return rep
}

var ws = regexp.MustCompile(`\s+`)

func squeeze(s string) string {
	s = ws.ReplaceAllString(s, " ")
	if len(s) > 120 {
		// Trim on rune boundary to avoid splitting a multibyte 한글 char.
		r := []rune(s)
		if len(r) > 60 {
			return string(r[:60]) + "…"
		}
	}
	return s
}
