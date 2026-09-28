package resume

import (
	"fmt"
	"strings"
)

// In-memory mutation helpers shared by the Store CRUD methods and the
// resume_items tool. ApplyAdd* return the new item's index; ApplyDelete*
// return the removed item.

func ApplyAddExperience(rd *ResumeData, data Experience) int {
	rd.Experiences = append(rd.Experiences, data)
	return len(rd.Experiences) - 1
}

func ApplyUpdateExperience(rd *ResumeData, index int, data Experience) error {
	if err := checkIndex("experience", index, len(rd.Experiences)); err != nil {
		return err
	}
	rd.Experiences[index] = data
	return nil
}

func ApplyDeleteExperience(rd *ResumeData, index int) (Experience, error) {
	if err := checkIndex("experience", index, len(rd.Experiences)); err != nil {
		return Experience{}, err
	}
	deleted := rd.Experiences[index]
	rd.Experiences = append(rd.Experiences[:index], rd.Experiences[index+1:]...)
	return deleted, nil
}

func ApplyAddProject(rd *ResumeData, data Project) int {
	rd.Projects = append(rd.Projects, data)
	return len(rd.Projects) - 1
}

func ApplyUpdateProject(rd *ResumeData, index int, data Project) error {
	if err := checkIndex("project", index, len(rd.Projects)); err != nil {
		return err
	}
	rd.Projects[index] = data
	return nil
}

func ApplyDeleteProject(rd *ResumeData, index int) (Project, error) {
	if err := checkIndex("project", index, len(rd.Projects)); err != nil {
		return Project{}, err
	}
	deleted := rd.Projects[index]
	rd.Projects = append(rd.Projects[:index], rd.Projects[index+1:]...)
	return deleted, nil
}

func ApplyAddEducation(rd *ResumeData, data Education) int {
	rd.Education = append(rd.Education, data)
	return len(rd.Education) - 1
}

func ApplyUpdateEducation(rd *ResumeData, index int, data Education) error {
	if err := checkIndex("education", index, len(rd.Education)); err != nil {
		return err
	}
	rd.Education[index] = data
	return nil
}

func ApplyDeleteEducation(rd *ResumeData, index int) (Education, error) {
	if err := checkIndex("education", index, len(rd.Education)); err != nil {
		return Education{}, err
	}
	deleted := rd.Education[index]
	rd.Education = append(rd.Education[:index], rd.Education[index+1:]...)
	return deleted, nil
}

func ApplyAddSkill(rd *ResumeData, data SkillGroup) int {
	rd.Skills = append(rd.Skills, data)
	return len(rd.Skills) - 1
}

func ApplyUpdateSkill(rd *ResumeData, index int, data SkillGroup) error {
	if err := checkIndex("skill", index, len(rd.Skills)); err != nil {
		return err
	}
	rd.Skills[index] = data
	return nil
}

func ApplyDeleteSkill(rd *ResumeData, index int) (SkillGroup, error) {
	if err := checkIndex("skill", index, len(rd.Skills)); err != nil {
		return SkillGroup{}, err
	}
	deleted := rd.Skills[index]
	rd.Skills = append(rd.Skills[:index], rd.Skills[index+1:]...)
	return deleted, nil
}

func ApplyAddBullet(rd *ResumeData, parentType string, parentIndex int, text string) (int, error) {
	if strings.TrimSpace(text) == "" {
		return 0, fmt.Errorf("bullet text is required")
	}
	bullets, err := bulletsPtr(rd, parentType, parentIndex)
	if err != nil {
		return 0, err
	}
	*bullets = append(*bullets, text)
	return len(*bullets) - 1, nil
}

func ApplyUpdateBullet(rd *ResumeData, parentType string, parentIndex, bulletIndex int, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("bullet text is required")
	}
	bullets, err := bulletsPtr(rd, parentType, parentIndex)
	if err != nil {
		return err
	}
	if err := checkIndex("bullet", bulletIndex, len(*bullets)); err != nil {
		return err
	}
	(*bullets)[bulletIndex] = text
	return nil
}

func ApplyDeleteBullet(rd *ResumeData, parentType string, parentIndex, bulletIndex int) (string, error) {
	bullets, err := bulletsPtr(rd, parentType, parentIndex)
	if err != nil {
		return "", err
	}
	if err := checkIndex("bullet", bulletIndex, len(*bullets)); err != nil {
		return "", err
	}
	deleted := (*bullets)[bulletIndex]
	*bullets = append((*bullets)[:bulletIndex], (*bullets)[bulletIndex+1:]...)
	return deleted, nil
}

func bulletsPtr(d *ResumeData, parentType string, parentIndex int) (*[]string, error) {
	switch parentType {
	case ParentExperience:
		if err := checkIndex("experience", parentIndex, len(d.Experiences)); err != nil {
			return nil, err
		}
		return &d.Experiences[parentIndex].Bullets, nil
	case ParentProject:
		if err := checkIndex("project", parentIndex, len(d.Projects)); err != nil {
			return nil, err
		}
		return &d.Projects[parentIndex].Bullets, nil
	default:
		return nil, invalidParentTypeError(parentType)
	}
}
