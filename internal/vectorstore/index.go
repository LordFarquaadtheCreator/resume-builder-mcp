package vectorstore

import (
	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
)

// BuildChunks builds one chunk per embeddable resume item, leaving embeddings
// nil for the caller to fill.
func BuildChunks(data resume.ResumeData) []Chunk {
	var chunks []Chunk

	for i, exp := range data.Experiences {
		for j, bullet := range exp.Bullets {
			chunks = append(chunks, ExperienceBulletChunk(exp, i, j, bullet))
		}
	}

	for i, skill := range data.Skills {
		chunks = append(chunks, SkillChunk(skill, i))
	}

	for i, proj := range data.Projects {
		for j, bullet := range proj.Bullets {
			chunks = append(chunks, ProjectBulletChunk(proj, i, j, bullet))
		}
	}

	for i, edu := range data.Education {
		chunks = append(chunks, EducationChunk(edu, i))
	}

	return chunks
}
