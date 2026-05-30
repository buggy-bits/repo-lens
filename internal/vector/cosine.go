package vector

import (
	"math"
	"sort"

	"github.com/buggy-bits/repo-lens/internal/store"
)

type RankedChunk struct {
	Content  string
	FilePath string
	Score    float64
}

// CosineSimilarity measures the angle between two vectors
func CosineSimilarity(a, b []float64) float64 {
	var dot, magA, magB float64
	for i := 0; i < len(a) && i < len(b); i++ {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// FindTopMatches returns the N highest-scoring chunks
func FindTopMatches(queryVec []float64, chunks []store.VectorChunk, topN int) []RankedChunk {
	ranked := make([]RankedChunk, len(chunks))
	for i, c := range chunks {
		ranked[i] = RankedChunk{
			Content:  c.Content,
			FilePath: c.FilePath,
			Score:    CosineSimilarity(queryVec, c.Vector),
		}
	}

	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score > ranked[j].Score
	})

	if topN > len(ranked) {
		topN = len(ranked)
	}
	return ranked[:topN]
}
