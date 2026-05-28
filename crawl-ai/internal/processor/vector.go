package processor

import (
	"encoding/json"
	"math"
	"os"
	"sort"
)

type VectorStore struct {
	Embeddings map[string][]float64 // Map of DocID -> Vector
}

func NewVectorStore() *VectorStore {
	return &VectorStore{
		Embeddings: make(map[string][]float64),
	}
}

func (s *VectorStore) Save(path string) error {
	data, err := json.Marshal(s.Embeddings)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *VectorStore) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &s.Embeddings)
}

func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

type SearchMatch struct {
	ID    string
	Score float64
}

func (s *VectorStore) Search(queryVector []float64, topK int) []SearchMatch {
	var matches []SearchMatch
	for id, vec := range s.Embeddings {
		score := CosineSimilarity(queryVector, vec)
		matches = append(matches, SearchMatch{ID: id, Score: score})
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if len(matches) > topK {
		return matches[:topK]
	}
	return matches
}
