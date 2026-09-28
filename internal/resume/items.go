package resume

import "fmt"

// Item types addressable through the resume_items tool.
const (
	ItemExperience = "experience"
	ItemProject    = "project"
	ItemEducation  = "education"
	ItemSkill      = "skill"
	ItemBullet     = "bullet"
)

// Parent types that own a bullet list.
const (
	ParentExperience = "experience"
	ParentProject    = "project"
)

func ValidItemType(t string) bool {
	switch t {
	case ItemExperience, ItemProject, ItemEducation, ItemSkill, ItemBullet:
		return true
	}
	return false
}

func ValidParentType(t string) bool {
	return t == ParentExperience || t == ParentProject
}

func (d ResumeData) ExperienceAt(index int) (Experience, error) {
	if err := checkIndex("experience", index, len(d.Experiences)); err != nil {
		return Experience{}, err
	}
	return d.Experiences[index], nil
}

func (d ResumeData) SkillAt(index int) (SkillGroup, error) {
	if err := checkIndex("skill", index, len(d.Skills)); err != nil {
		return SkillGroup{}, err
	}
	return d.Skills[index], nil
}

func (d ResumeData) ProjectAt(index int) (Project, error) {
	if err := checkIndex("project", index, len(d.Projects)); err != nil {
		return Project{}, err
	}
	return d.Projects[index], nil
}

func (d ResumeData) EducationAt(index int) (Education, error) {
	if err := checkIndex("education", index, len(d.Education)); err != nil {
		return Education{}, err
	}
	return d.Education[index], nil
}

func (d ResumeData) Bullets(parentType string, parentIndex int) ([]string, error) {
	switch parentType {
	case ParentExperience:
		if err := checkIndex("experience", parentIndex, len(d.Experiences)); err != nil {
			return nil, err
		}
		return d.Experiences[parentIndex].Bullets, nil
	case ParentProject:
		if err := checkIndex("project", parentIndex, len(d.Projects)); err != nil {
			return nil, err
		}
		return d.Projects[parentIndex].Bullets, nil
	default:
		return nil, invalidParentTypeError(parentType)
	}
}

// Merge* apply non-empty patch fields onto base; a non-nil patch bullet list
// replaces the base bullets.
func MergeExperience(base, patch Experience) Experience {
	if patch.Company != "" {
		base.Company = patch.Company
	}
	if patch.Role != "" {
		base.Role = patch.Role
	}
	if patch.Start != "" {
		base.Start = patch.Start
	}
	if patch.End != "" {
		base.End = patch.End
	}
	if patch.Location != "" {
		base.Location = patch.Location
	}
	if patch.Link != "" {
		base.Link = patch.Link
	}
	if patch.Bullets != nil {
		base.Bullets = patch.Bullets
	}
	return base
}

func MergeProject(base, patch Project) Project {
	if patch.Name != "" {
		base.Name = patch.Name
	}
	if patch.Tech != "" {
		base.Tech = patch.Tech
	}
	if patch.Date != "" {
		base.Date = patch.Date
	}
	if patch.Link != "" {
		base.Link = patch.Link
	}
	if patch.Bullets != nil {
		base.Bullets = patch.Bullets
	}
	return base
}

func MergeEducation(base, patch Education) Education {
	if patch.Institution != "" {
		base.Institution = patch.Institution
	}
	if patch.Degree != "" {
		base.Degree = patch.Degree
	}
	if patch.Start != "" {
		base.Start = patch.Start
	}
	if patch.End != "" {
		base.End = patch.End
	}
	if patch.Location != "" {
		base.Location = patch.Location
	}
	if patch.Link != "" {
		base.Link = patch.Link
	}
	return base
}

func MergeSkill(base, patch SkillGroup) SkillGroup {
	if patch.Category != "" {
		base.Category = patch.Category
	}
	if patch.Values != "" {
		base.Values = patch.Values
	}
	return base
}

func checkIndex(kind string, index, count int) error {
	if index < 0 || index >= count {
		return fmt.Errorf("%s index %d out of range (have %d)", kind, index, count)
	}
	return nil
}

func invalidParentTypeError(t string) error {
	return fmt.Errorf("invalid parentType %q (must be %q or %q)", t, ParentExperience, ParentProject)
}
