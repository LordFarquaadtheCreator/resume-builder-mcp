package vectorstore

// EmbeddingConfig holds the OpenAI-compatible embedding endpoint config.
type EmbeddingConfig struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	Model   string `json:"model"`
}

// ChunkType identifies what kind of resume item a chunk represents.
type ChunkType string

const (
	ChunkExperienceBullet ChunkType = "experience_bullet"
	ChunkSkillGroup       ChunkType = "skill_group"
	ChunkProjectBullet    ChunkType = "project_bullet"
	ChunkEducation        ChunkType = "education"
)

// Chunk is a single embedded unit in the vector store.
type Chunk struct {
	ID       string    `json:"id"`
	Type     ChunkType `json:"type"`
	Text     string    `json:"text"`
	Embedding []float64 `json:"-"` // not persisted in chunks.json
	Metadata Metadata  `json:"metadata"`
}

// Metadata carries context about the chunk's source item.
type Metadata struct {
	Company     string `json:"company,omitempty"`
	Role        string `json:"role,omitempty"`
	Start       string `json:"start,omitempty"`
	End         string `json:"end,omitempty"`
	Location    string `json:"location,omitempty"`
	Link        string `json:"link,omitempty"`
	BulletIndex int    `json:"bulletIndex,omitempty"`

	ProjectName string `json:"projectName,omitempty"`
	Tech        string `json:"tech,omitempty"`
	Date        string `json:"date,omitempty"`

	Category string `json:"category,omitempty"`

	Institution string `json:"institution,omitempty"`
	Degree      string `json:"degree,omitempty"`
}

// ScoredChunk is a chunk with its similarity score.
type ScoredChunk struct {
	Chunk Chunk    `json:"chunk"`
	Score float64  `json:"score"`
}

// SearchResult groups scored chunks by type.
type SearchResult struct {
	Experiences []ScoredExperience `json:"experiences"`
	Skills      []ScoredChunk      `json:"skills"`
	Projects    []ScoredProject    `json:"projects"`
	Education   []ScoredChunk      `json:"education"`
}

// ScoredExperience groups bullets under their parent experience.
type ScoredExperience struct {
	Company  string         `json:"company"`
	Role     string         `json:"role"`
	Start    string         `json:"start"`
	End      string         `json:"end"`
	Location string         `json:"location"`
	Link     string         `json:"link"`
	Bullets  []ScoredBullet `json:"bullets"`
}

// ScoredBullet is a single bullet with its relevance score.
type ScoredBullet struct {
	Text  string  `json:"text"`
	Score float64 `json:"score"`
}

// ScoredProject groups bullets under their parent project.
type ScoredProject struct {
	Name    string         `json:"name"`
	Tech    string         `json:"tech"`
	Date    string         `json:"date"`
	Link    string         `json:"link"`
	Bullets []ScoredBullet `json:"bullets"`
}
