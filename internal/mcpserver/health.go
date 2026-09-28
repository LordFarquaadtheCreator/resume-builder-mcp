package mcpserver

import (
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
)

type HealthInput struct{}

type HealthOutput struct {
	HasResume          bool         `json:"hasResume"`
	HasEmbeddingConfig bool         `json:"hasEmbeddingConfig"`
	VectorChunkCount   int          `json:"vectorChunkCount"`
	ResumeStats        resume.Stats `json:"resumeStats"`
	InitializedAt      string       `json:"initializedAt,omitempty"`
}

func handleHealth(d deps) (*mcp.CallToolResult, HealthOutput, error) {
	out := HealthOutput{
		HasEmbeddingConfig: d.ConfigStore.Exists(),
		VectorChunkCount:   d.VectorStore.ChunkCount(),
	}
	if d.ResumeStore.Exists() {
		stored, err := d.ResumeStore.Load()
		if err != nil {
			log.Printf("health: ERROR loading resume: %v", err)
			return nil, HealthOutput{}, err
		}
		out.HasResume = true
		out.ResumeStats = resume.ComputeStats(stored.Data)
		if !stored.InitializedAt.IsZero() {
			out.InitializedAt = stored.InitializedAt.Format("2006-01-02T15:04:05Z")
		}
	}
	log.Printf("health: hasResume=%v hasEmbeddingConfig=%v chunks=%d", out.HasResume, out.HasEmbeddingConfig, out.VectorChunkCount)
	return jsonResult(out)
}
