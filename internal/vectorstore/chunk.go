package vectorstore

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
)

// Chunk IDs encode resume item positions so CRUD operations can map items to
// chunks; deleting an item renumbers later IDs to keep that mapping.
//
//	exp_<expIndex>_bullet_<bulletIndex>
//	proj_<projIndex>_bullet_<bulletIndex>
//	skill_<index>
//	edu_<index>
func ExperienceChunkID(expIndex, bulletIndex int) string {
	return fmt.Sprintf("exp_%d_bullet_%d", expIndex, bulletIndex)
}

func ProjectChunkID(projIndex, bulletIndex int) string {
	return fmt.Sprintf("proj_%d_bullet_%d", projIndex, bulletIndex)
}

func SkillChunkID(index int) string {
	return fmt.Sprintf("skill_%d", index)
}

func EducationChunkID(index int) string {
	return fmt.Sprintf("edu_%d", index)
}

// ExperienceBulletChunk builds an experience bullet chunk; text is separate
// because add/update targets are not in exp.Bullets yet.
func ExperienceBulletChunk(exp resume.Experience, expIndex, bulletIndex int, text string) Chunk {
	return Chunk{
		ID:   ExperienceChunkID(expIndex, bulletIndex),
		Type: ChunkExperienceBullet,
		Text: text,
		Metadata: Metadata{
			Company:     exp.Company,
			Role:        exp.Role,
			Start:       exp.Start,
			End:         exp.End,
			Location:    exp.Location,
			Link:        exp.Link,
			BulletIndex: bulletIndex,
		},
	}
}

func ProjectBulletChunk(proj resume.Project, projIndex, bulletIndex int, text string) Chunk {
	return Chunk{
		ID:   ProjectChunkID(projIndex, bulletIndex),
		Type: ChunkProjectBullet,
		Text: text,
		Metadata: Metadata{
			ProjectName: proj.Name,
			Tech:        proj.Tech,
			Date:        proj.Date,
			Link:        proj.Link,
			BulletIndex: bulletIndex,
		},
	}
}

func SkillChunk(skill resume.SkillGroup, index int) Chunk {
	return Chunk{
		ID:   SkillChunkID(index),
		Type: ChunkSkillGroup,
		Text: SkillText(skill),
		Metadata: Metadata{
			Category: skill.Category,
		},
	}
}

func SkillText(skill resume.SkillGroup) string {
	return fmt.Sprintf("%s: %s", skill.Category, skill.Values)
}

func EducationChunk(edu resume.Education, index int) Chunk {
	return Chunk{
		ID:   EducationChunkID(index),
		Type: ChunkEducation,
		Text: EducationText(edu),
		Metadata: Metadata{
			Institution: edu.Institution,
			Degree:      edu.Degree,
			Start:       edu.Start,
			End:         edu.End,
			Location:    edu.Location,
			Link:        edu.Link,
		},
	}
}

func EducationText(edu resume.Education) string {
	return fmt.Sprintf("%s - %s", edu.Institution, edu.Degree)
}

// chunkCodec maps a chunk ID format to and from (parent, child) indices.
type chunkCodec struct {
	parse   func(id string) (parent, child int, ok bool)
	rebuild func(parent, child int) string
}

var (
	experienceCodec = chunkCodec{
		parse:   func(id string) (int, int, bool) { return parseTwoLevelID(id, "exp_", "_bullet_") },
		rebuild: ExperienceChunkID,
	}
	projectCodec = chunkCodec{
		parse:   func(id string) (int, int, bool) { return parseTwoLevelID(id, "proj_", "_bullet_") },
		rebuild: ProjectChunkID,
	}
	skillCodec = chunkCodec{
		parse:   func(id string) (int, int, bool) { return parseOneLevelID(id, "skill_") },
		rebuild: func(parent, _ int) string { return SkillChunkID(parent) },
	}
	educationCodec = chunkCodec{
		parse:   func(id string) (int, int, bool) { return parseOneLevelID(id, "edu_") },
		rebuild: func(parent, _ int) string { return EducationChunkID(parent) },
	}
)

func parseTwoLevelID(id, prefix, sep string) (int, int, bool) {
	rest, ok := strings.CutPrefix(id, prefix)
	if !ok {
		return 0, 0, false
	}
	parts := strings.Split(rest, sep)
	if len(parts) != 2 {
		return 0, 0, false
	}
	parent, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	child, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return parent, child, true
}

func parseOneLevelID(id, prefix string) (int, int, bool) {
	rest, ok := strings.CutPrefix(id, prefix)
	if !ok {
		return 0, 0, false
	}
	index, err := strconv.Atoi(rest)
	if err != nil {
		return 0, 0, false
	}
	return index, 0, true
}
