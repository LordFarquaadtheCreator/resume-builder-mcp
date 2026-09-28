package vectorstore

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	vectorsFile = "vectors.json"
	chunksFile  = "chunks.json"
)

// vectorEntry is the on-disk format for vectors.json.
type vectorEntry struct {
	ID        string    `json:"id"`
	Embedding []float64 `json:"embedding"`
}

// Store is an in-memory vector store with disk persistence.
type Store struct {
	dataDir string
	chunks  []Chunk
	vectors []vectorEntry
}

// NewStore creates an empty store. Call Load to read existing data.
func NewStore(dataDir string) *Store {
	return &Store{dataDir: dataDir}
}

// Load reads vectors and chunks from disk.
func (s *Store) Load() error {
	vpath := filepath.Join(s.dataDir, vectorsFile)
	cpath := filepath.Join(s.dataDir, chunksFile)

	vb, err := os.ReadFile(vpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read vectors: %w", err)
	}
	if err := json.Unmarshal(vb, &s.vectors); err != nil {
		return fmt.Errorf("unmarshal vectors: %w", err)
	}

	cb, err := os.ReadFile(cpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read chunks: %w", err)
	}
	if err := json.Unmarshal(cb, &s.chunks); err != nil {
		return fmt.Errorf("unmarshal chunks: %w", err)
	}

	// attach embeddings to chunks
	for i := range s.chunks {
		for _, v := range s.vectors {
			if v.ID == s.chunks[i].ID {
				s.chunks[i].Embedding = v.Embedding
				break
			}
		}
	}

	return nil
}

// Save writes vectors and chunks to disk.
func (s *Store) Save() error {
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// rebuild vector entries from chunks
	s.vectors = make([]vectorEntry, len(s.chunks))
	for i, c := range s.chunks {
		s.vectors[i] = vectorEntry{ID: c.ID, Embedding: c.Embedding}
	}

	vb, err := json.MarshalIndent(s.vectors, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal vectors: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, vectorsFile), vb, 0644); err != nil {
		return fmt.Errorf("write vectors: %w", err)
	}

	// chunks without embedding data
	type chunkOnDisk struct {
		ID       string    `json:"id"`
		Type     ChunkType `json:"type"`
		Text     string    `json:"text"`
		Metadata Metadata  `json:"metadata"`
	}
	onDisk := make([]chunkOnDisk, len(s.chunks))
	for i, c := range s.chunks {
		onDisk[i] = chunkOnDisk{
			ID:       c.ID,
			Type:     c.Type,
			Text:     c.Text,
			Metadata: c.Metadata,
		}
	}
	cb, err := json.MarshalIndent(onDisk, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal chunks: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, chunksFile), cb, 0644); err != nil {
		return fmt.Errorf("write chunks: %w", err)
	}

	return nil
}

// Reset clears all stored chunks and vectors.
func (s *Store) Reset() {
	s.chunks = nil
	s.vectors = nil
}

// AddChunk adds a chunk with its embedding.
func (s *Store) AddChunk(c Chunk) {
	s.chunks = append(s.chunks, c)
}

// Search returns the top-K chunks by cosine similarity to the query embedding.
func (s *Store) Search(queryEmbedding []float64, topK int) []ScoredChunk {
	if len(s.chunks) == 0 || topK <= 0 {
		return nil
	}

	scored := make([]ScoredChunk, len(s.chunks))
	for i, c := range s.chunks {
		scored[i] = ScoredChunk{
			Chunk: c,
			Score: cosineSim(queryEmbedding, c.Embedding),
		}
	}

	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	if topK > len(scored) {
		topK = len(scored)
	}
	return scored[:topK]
}

// SearchByType returns scored chunks of a specific type.
func (s *Store) SearchByType(queryEmbedding []float64, ct ChunkType, topK int) []ScoredChunk {
	if len(s.chunks) == 0 || topK <= 0 {
		return nil
	}

	var scored []ScoredChunk
	for _, c := range s.chunks {
		if c.Type != ct {
			continue
		}
		scored = append(scored, ScoredChunk{
			Chunk: c,
			Score: cosineSim(queryEmbedding, c.Embedding),
		})
	}

	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	if topK > len(scored) {
		topK = len(scored)
	}
	return scored[:topK]
}

// SearchGrouped performs a search and groups results into a SearchResult.
// Experiences are reverse chronological; bullets ranked by score within each.
func (s *Store) SearchGrouped(queryEmbedding []float64, topKPerType int) SearchResult {
	result := SearchResult{}

	// Experiences: group bullets by company, reverse chronological
	expBullets := s.SearchByType(queryEmbedding, ChunkExperienceBullet, topKPerType*5)
	expMap := make(map[string]*ScoredExperience)
	var expOrder []string

	for _, sc := range expBullets {
		key := sc.Chunk.Metadata.Company + "|" + sc.Chunk.Metadata.Role
		if _, ok := expMap[key]; !ok {
			expMap[key] = &ScoredExperience{
				Company:  sc.Chunk.Metadata.Company,
				Role:     sc.Chunk.Metadata.Role,
				Start:    sc.Chunk.Metadata.Start,
				End:      sc.Chunk.Metadata.End,
				Location: sc.Chunk.Metadata.Location,
				Link:     sc.Chunk.Metadata.Link,
			}
			expOrder = append(expOrder, key)
		}
		expMap[key].Bullets = append(expMap[key].Bullets, ScoredBullet{
			Text:  sc.Chunk.Text,
			Score: sc.Score,
		})
	}

	// sort experiences reverse chronological by end date, then start date as tiebreaker
	sort.Slice(expOrder, func(i, j int) bool {
		ei := expMap[expOrder[i]]
		ej := expMap[expOrder[j]]
		endI := parseChron(ei.End)
		endJ := parseChron(ej.End)
		if endI != endJ {
			return endI > endJ
		}
		return parseChron(ei.Start) > parseChron(ej.Start)
	})

	for _, key := range expOrder {
		// sort bullets by score desc within experience
		sort.Slice(expMap[key].Bullets, func(i, j int) bool { return expMap[key].Bullets[i].Score > expMap[key].Bullets[j].Score })
		result.Experiences = append(result.Experiences, *expMap[key])
	}

	// Skills
	result.Skills = s.SearchByType(queryEmbedding, ChunkSkillGroup, topKPerType)

	// Projects: group bullets by project
	projBullets := s.SearchByType(queryEmbedding, ChunkProjectBullet, topKPerType*3)
	projMap := make(map[string]*ScoredProject)
	var projOrder []string

	for _, sc := range projBullets {
		key := sc.Chunk.Metadata.ProjectName
		if _, ok := projMap[key]; !ok {
			projMap[key] = &ScoredProject{
				Name: sc.Chunk.Metadata.ProjectName,
				Tech: sc.Chunk.Metadata.Tech,
				Date: sc.Chunk.Metadata.Date,
				Link: sc.Chunk.Metadata.Link,
			}
			projOrder = append(projOrder, key)
		}
		projMap[key].Bullets = append(projMap[key].Bullets, ScoredBullet{
			Text:  sc.Chunk.Text,
			Score: sc.Score,
		})
	}

	// sort projects by max bullet score
	sort.Slice(projOrder, func(i, j int) bool {
		return maxBulletScore(projMap[projOrder[i]]) > maxBulletScore(projMap[projOrder[j]])
	})

	for _, key := range projOrder {
		sort.Slice(projMap[key].Bullets, func(i, j int) bool { return projMap[key].Bullets[i].Score > projMap[key].Bullets[j].Score })
		result.Projects = append(result.Projects, *projMap[key])
	}

	// Education
	result.Education = s.SearchByType(queryEmbedding, ChunkEducation, topKPerType)

	return result
}

// HasData returns true if the store has any chunks.
func (s *Store) HasData() bool {
	return len(s.chunks) > 0
}

// ChunkCount returns the number of stored chunks.
func (s *Store) ChunkCount() int {
	return len(s.chunks)
}

func cosineSim(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, magA, magB float64
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// chronAfter returns true if a is more recent than b.
// Parses month-year formats like "Dec. 2025", "Jan 2024", "2024".
// "Present" or empty sorts as most recent.
func chronAfter(a, b string) bool {
	aa := parseChron(a)
	bb := parseChron(b)
	return aa > bb
}

// parseChron converts a date string to a comparable integer (year*12 + month).
// Returns math.MaxInt32 for "Present" or unparseable strings (sorts most recent).
func parseChron(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "present") {
		return math.MaxInt32
	}

	s = strings.TrimSuffix(s, ".")
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return math.MaxInt32
	}

	var month, year int

	switch len(fields) {
	case 1:
		// just a year
		year = atoiSafe(fields[0])
		month = 12
	case 2:
		// month + year
		month = monthNum(fields[0])
		year = atoiSafe(fields[1])
	default:
		return math.MaxInt32
	}

	if year == 0 {
		return math.MaxInt32
	}
	return year*12 + month
}

var monthMap = map[string]int{
	"jan": 1, "january": 1,
	"feb": 2, "february": 2,
	"mar": 3, "march": 3,
	"apr": 4, "april": 4,
	"may": 5,
	"jun": 6, "june": 6,
	"jul": 7, "july": 7,
	"aug": 8, "august": 8,
	"sep": 9, "sept": 9, "september": 9,
	"oct": 10, "october": 10,
	"nov": 11, "november": 11,
	"dec": 12, "december": 12,
}

func monthNum(s string) int {
	return monthMap[strings.ToLower(strings.TrimSuffix(s, "."))]
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func maxBulletScore(p *ScoredProject) float64 {
	max := 0.0
	for _, b := range p.Bullets {
		if b.Score > max {
			max = b.Score
		}
	}
	return max
}
