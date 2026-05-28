package utils

import (
	"regexp"
	"sort"
	"strings"
)

var StopWords = map[string]bool{
	"the": true, "and": true, "a": true, "to": true, "of": true, "in": true, "i": true, "is": true, "that": true, "it": true,
	"on": true, "you": true, "this": true, "for": true, "with": true, "are": true, "be": true, "as": true, "at": true, "from": true,
	"하": true, "는": true, "가": true, "이": true, "을": true, "를": true, "에": true, "은": true, "도": true, "한": true,
}

// ExtractKeywords returns top N keywords from the text, filtering out common stop words.
func ExtractKeywords(text string, count int) []string {
	// Simple tokenization: lowercase and remove non-alphanumeric (roughly)
	text = strings.ToLower(text)
	words := regexp.MustCompile(`[a-z0-9\p{Hangul}]{2,}`).FindAllString(text, -1)

	freq := make(map[string]int)
	for _, w := range words {
		if StopWords[w] {
			continue
		}
		freq[w]++
	}

	type kv struct {
		Word  string
		Count int
	}
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	var result []string
	for i := 0; i < len(sorted) && i < count; i++ {
		result = append(result, sorted[i].Word)
	}

	return result
}
