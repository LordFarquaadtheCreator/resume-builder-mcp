package generate

import (
	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/vectorstore"
)

// AutoBuild constructs a ResumeData from search results.
// Experiences are reverse chronological (already sorted by SearchGrouped).
// Bullets within each experience are ranked by relevance score.
// Projects ranked by max bullet score.
// Skills and education included as-is from search results.
func AutoBuild(stored resume.ResumeData, result vectorstore.SearchResult) resume.ResumeData {
	data := resume.ResumeData{
		Name:    stored.Name,
		Contact: stored.Contact,
	}

	// Education: include all from search results, fallback to stored
	if len(result.Education) > 0 {
		for _, sc := range result.Education {
			data.Education = append(data.Education, resume.Education{
				Institution: sc.Chunk.Metadata.Institution,
				Degree:      sc.Chunk.Metadata.Degree,
				Start:       sc.Chunk.Metadata.Start,
				End:         sc.Chunk.Metadata.End,
				Location:    sc.Chunk.Metadata.Location,
				Link:        sc.Chunk.Metadata.Link,
			})
		}
	} else {
		data.Education = stored.Education
	}

	// Skills: include all from search results, fallback to stored
	if len(result.Skills) > 0 {
		for _, sc := range result.Skills {
			data.Skills = append(data.Skills, resume.SkillGroup{
				Category: sc.Chunk.Metadata.Category,
				Values:   sc.Chunk.Text,
			})
		}
	} else {
		data.Skills = stored.Skills
	}

	// Experiences: from search results (already reverse chronological)
	for _, exp := range result.Experiences {
		e := resume.Experience{
			Company:  exp.Company,
			Role:     exp.Role,
			Start:    exp.Start,
			End:      exp.End,
			Location: exp.Location,
			Link:     exp.Link,
		}
		for _, b := range exp.Bullets {
			e.Bullets = append(e.Bullets, b.Text)
		}
		data.Experiences = append(data.Experiences, e)
	}

	// Projects: from search results (already ranked by score)
	for _, proj := range result.Projects {
		p := resume.Project{
			Name: proj.Name,
			Tech: proj.Tech,
			Date: proj.Date,
			Link: proj.Link,
		}
		for _, b := range proj.Bullets {
			p.Bullets = append(p.Bullets, b.Text)
		}
		data.Projects = append(data.Projects, p)
	}

	return data
}
