package mcpserver

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

func experienceChunks(e resume.Experience, index int) []vectorstore.Chunk {
	chunks := make([]vectorstore.Chunk, len(e.Bullets))
	for j, b := range e.Bullets {
		chunks[j] = vectorstore.ExperienceBulletChunk(e, index, j, b)
	}
	return chunks
}

func projectChunks(p resume.Project, index int) []vectorstore.Chunk {
	chunks := make([]vectorstore.Chunk, len(p.Bullets))
	for j, b := range p.Bullets {
		chunks[j] = vectorstore.ProjectBulletChunk(p, index, j, b)
	}
	return chunks
}

// bulletChunk builds the chunk for one bullet whose text may not be stored in
// the parent yet (add/update).
func bulletChunk(data resume.ResumeData, parentType string, parentIndex, bulletIndex int, text string) (vectorstore.Chunk, error) {
	switch parentType {
	case resume.ParentExperience:
		exp, err := data.ExperienceAt(parentIndex)
		if err != nil {
			return vectorstore.Chunk{}, err
		}
		return vectorstore.ExperienceBulletChunk(exp, parentIndex, bulletIndex, text), nil
	case resume.ParentProject:
		proj, err := data.ProjectAt(parentIndex)
		if err != nil {
			return vectorstore.Chunk{}, err
		}
		return vectorstore.ProjectBulletChunk(proj, parentIndex, bulletIndex, text), nil
	default:
		return vectorstore.Chunk{}, fmt.Errorf("invalid parentType %q (must be %q or %q)", parentType, resume.ParentExperience, resume.ParentProject)
	}
}

// staleBulletIDs lists positions that existed before an update but no longer do.
func staleBulletIDs(idFn func(parent, bullet int) string, parentIndex, newCount, oldCount int) []string {
	var ids []string
	for j := newCount; j < oldCount; j++ {
		ids = append(ids, idFn(parentIndex, j))
	}
	return ids
}

func txResult(op string, entry ItemEntry, tx *itemTx) (*mcp.CallToolResult, ResumeItemsOutput, error) {
	stats := tx.stats()
	chunks := tx.chunkCount()
	return jsonResult(ResumeItemsOutput{
		Operation:    op,
		Message:      mutationMessage(op, entry),
		Item:         &entry,
		Stats:        &stats,
		VectorChunks: &chunks,
	})
}

func mutationMessage(op string, entry ItemEntry) string {
	verb := map[string]string{"add": "Added", "update": "Updated", "delete": "Deleted"}[op]
	if entry.Type == resume.ItemBullet {
		return fmt.Sprintf("%s bullet %d in %s %d", verb, entry.ID, entry.ParentType, *entry.ParentID)
	}
	return fmt.Sprintf("%s %s at index %d", verb, entry.Type, entry.ID)
}
