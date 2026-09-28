package resume

import (
	"fmt"
	"log"
	"strings"
)

// Quotas defines hard caps per section to prevent absurd input.
type Quotas struct {
	MaxExperiences int
	MaxBulletsExp  int
	MaxProjects    int
	MaxBulletsProj int
	MaxSkillGroups int
	MaxEducation   int
}

// DefaultQuotas are the guard rail limits for the fahad template.
var DefaultQuotas = Quotas{
	MaxExperiences: 6,
	MaxBulletsExp:  5,
	MaxProjects:    4,
	MaxBulletsProj: 2,
	MaxSkillGroups: 5,
	MaxEducation:   3,
}

// Validate checks resume data against quotas. Returns list of violations.
func Validate(data ResumeData, q Quotas) error {
	var violations []string

	if len(data.Experiences) > q.MaxExperiences {
		v := fmt.Sprintf("experiences: %d exceeds max %d", len(data.Experiences), q.MaxExperiences)
		log.Printf("validate: %s", v)
		violations = append(violations, v)
	}
	for i, e := range data.Experiences {
		if len(e.Bullets) > q.MaxBulletsExp {
			v := fmt.Sprintf("experience[%d] %s: %d bullets exceeds max %d", i, e.Company, len(e.Bullets), q.MaxBulletsExp)
			log.Printf("validate: %s", v)
			violations = append(violations, v)
		}
	}

	if len(data.Projects) > q.MaxProjects {
		v := fmt.Sprintf("projects: %d exceeds max %d", len(data.Projects), q.MaxProjects)
		log.Printf("validate: %s", v)
		violations = append(violations, v)
	}
	for i, p := range data.Projects {
		if len(p.Bullets) > q.MaxBulletsProj {
			v := fmt.Sprintf("project[%d] %s: %d bullets exceeds max %d", i, p.Name, len(p.Bullets), q.MaxBulletsProj)
			log.Printf("validate: %s", v)
			violations = append(violations, v)
		}
	}

	if len(data.Skills) > q.MaxSkillGroups {
		v := fmt.Sprintf("skill groups: %d exceeds max %d", len(data.Skills), q.MaxSkillGroups)
		log.Printf("validate: %s", v)
		violations = append(violations, v)
	}

	if len(data.Education) > q.MaxEducation {
		v := fmt.Sprintf("education: %d exceeds max %d", len(data.Education), q.MaxEducation)
		log.Printf("validate: %s", v)
		violations = append(violations, v)
	}

	if len(violations) > 0 {
		return fmt.Errorf("quota violations: [%s]", strings.Join(violations, "; "))
	}
	return nil
}
