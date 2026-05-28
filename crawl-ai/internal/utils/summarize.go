package utils

import (
	"regexp"
	"sort"
	"strings"
)

// Summarize extracts the top numSentences most important sentences from the text based on word frequency.
func Summarize(text string, numSentences int) string {
	if text == "" {
		return ""
	}

	// 1. Split into sentences (simple rule: . ! ? followed by space or end)
	re := regexp.MustCompile(`[^.!?]+[.!?]+`)
	rawSentences := re.FindAllString(text, -1)
	if len(rawSentences) <= numSentences {
		return strings.Join(rawSentences, " ")
	}

	// 2. Score words (using keyword logic)
	// We'll reuse the keywords logic for word scoring
	cleanText := strings.ToLower(text)
	words := regexp.MustCompile(`[a-z0-9\p{Hangul}]{2,}`).FindAllString(cleanText, -1)

	wordFreq := make(map[string]int)
	for _, w := range words {
		if !StopWords[w] {
			wordFreq[w]++
		}
	}

	// 3. Score sentences
	type sentenceScore struct {
		index int
		text  string
		score float64
	}
	scores := make([]sentenceScore, len(rawSentences))
	for i, s := range rawSentences {
		sClean := strings.ToLower(s)
		sWords := regexp.MustCompile(`[a-z0-9\p{Hangul}]{2,}`).FindAllString(sClean, -1)

		var totalScore float64
		for _, w := range sWords {
			if freq, ok := wordFreq[w]; ok {
				totalScore += float64(freq)
			}
		}

		// Normalize by sentence length to avoid picking only long sentences
		if len(sWords) > 0 {
			scores[i] = sentenceScore{
				index: i,
				text:  strings.TrimSpace(s),
				score: totalScore / float64(len(sWords)),
			}
		}
	}

	// 4. Sort by score to find top sentences
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 5. Select top sentences and sort them back to original order
	topSentences := scores
	if len(scores) > numSentences {
		topSentences = scores[:numSentences]
	}

	sort.Slice(topSentences, func(i, j int) bool {
		return topSentences[i].index < topSentences[j].index
	})

	var result []string
	for _, ts := range topSentences {
		result = append(result, ts.text)
	}

	return strings.Join(result, " ")
}
