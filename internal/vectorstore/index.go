package vectorstore

import (
	"fmt"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
)

// IndexResume builds chunks from resume data and embeds them all.
// Returns the chunks with embeddings ready to add to the store.
func IndexResume(data resume.ResumeData, client *EmbedClient) ([]Chunk, error) {
	var chunks []Chunk
	var texts []string

	// Experience bullets
	for i, exp := range data.Experiences {
		for j, bullet := range exp.Bullets {
			text := bullet
			chunk := Chunk{
				ID:   fmt.Sprintf("exp_%d_bullet_%d", i, j),
				Type: ChunkExperienceBullet,
				Text: text,
				Metadata: Metadata{
					Company:     exp.Company,
					Role:        exp.Role,
					Start:       exp.Start,
					End:         exp.End,
					Location:    exp.Location,
					Link:        exp.Link,
					BulletIndex: j,
				},
			}
			chunks = append(chunks, chunk)
			texts = append(texts, text)
		}
	}

	// Skill groups
	for i, skill := range data.Skills {
		text := fmt.Sprintf("%s: %s", skill.Category, skill.Values)
		chunk := Chunk{
			ID:   fmt.Sprintf("skill_%d", i),
			Type: ChunkSkillGroup,
			Text: text,
			Metadata: Metadata{
				Category: skill.Category,
			},
		}
		chunks = append(chunks, chunk)
		texts = append(texts, text)
	}

	// Project bullets
	for i, proj := range data.Projects {
		for j, bullet := range proj.Bullets {
			text := bullet
			chunk := Chunk{
				ID:   fmt.Sprintf("proj_%d_bullet_%d", i, j),
				Type: ChunkProjectBullet,
				Text: text,
				Metadata: Metadata{
					ProjectName: proj.Name,
					Tech:        proj.Tech,
					Date:        proj.Date,
					Link:        proj.Link,
					BulletIndex: j,
				},
			}
			chunks = append(chunks, chunk)
			texts = append(texts, text)
		}
	}

	// Education
	for i, edu := range data.Education {
		text := fmt.Sprintf("%s - %s", edu.Institution, edu.Degree)
		chunk := Chunk{
			ID:   fmt.Sprintf("edu_%d", i),
			Type: ChunkEducation,
			Text: text,
			Metadata: Metadata{
				Institution: edu.Institution,
				Degree:      edu.Degree,
				Start:       edu.Start,
				End:         edu.End,
				Location:    edu.Location,
				Link:        edu.Link,
			},
		}
		chunks = append(chunks, chunk)
		texts = append(texts, text)
	}

	// Embed all texts
	embeddings, err := client.EmbedBatch(texts)
	if err != nil {
		return nil, fmt.Errorf("embed resume chunks: %w", err)
	}

	for i := range chunks {
		chunks[i].Embedding = embeddings[i]
	}

	return chunks, nil
}
