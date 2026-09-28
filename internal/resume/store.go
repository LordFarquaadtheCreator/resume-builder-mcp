package resume

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const resumeFile = "resume.json"

const noResumeHint = `no resume found — call resume_items with operation "init" first`

// Store handles disk persistence of resume data.
type Store struct {
	dataDir string
}

func NewStore(dataDir string) *Store {
	return &Store{dataDir: dataDir}
}

func (s *Store) Exists() bool {
	_, err := os.Stat(filepath.Join(s.dataDir, resumeFile))
	return err == nil
}

// Save writes resume data to disk, overwriting any existing data. It resets
// InitializedAt, unlike the staged writes that use SaveStored.
func (s *Store) Save(data ResumeData) error {
	if data.Name == "" {
		return fmt.Errorf("name is required")
	}
	return s.SaveStored(StoredResume{Data: data, InitializedAt: time.Now()})
}

func (s *Store) SaveStored(stored StoredResume) error {
	b, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal resume: %w", err)
	}
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(s.dataDir, resumeFile), b, 0644); err != nil {
		return fmt.Errorf("write resume: %w", err)
	}
	return nil
}

// Load reads stored resume data from disk.
func (s *Store) Load() (*StoredResume, error) {
	path := filepath.Join(s.dataDir, resumeFile)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New(noResumeHint)
		}
		return nil, fmt.Errorf("read resume: %w", err)
	}
	var stored StoredResume
	if err := json.Unmarshal(b, &stored); err != nil {
		return nil, fmt.Errorf("unmarshal resume: %w", err)
	}
	return &stored, nil
}

// Delete removes stored resume data.
func (s *Store) Delete() error {
	path := filepath.Join(s.dataDir, resumeFile)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete resume: %w", err)
	}
	return nil
}

// --- CRUD ---
//
// Add* return the new item's index, Update* replace the whole item (the
// Merge* helpers build partial updates), and Delete* return the removed item.

// mutate loads the resume, applies fn, and saves it back, preserving InitializedAt.
func (s *Store) mutate(fn func(*ResumeData) error) error {
	stored, err := s.Load()
	if err != nil {
		return err
	}
	if err := fn(&stored.Data); err != nil {
		return err
	}
	return s.SaveStored(*stored)
}

func (s *Store) AddExperience(data Experience) (int, error) {
	index := 0
	err := s.mutate(func(rd *ResumeData) error {
		index = ApplyAddExperience(rd, data)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return index, nil
}

func (s *Store) GetExperience(index int) (Experience, error) {
	stored, err := s.Load()
	if err != nil {
		return Experience{}, err
	}
	return stored.Data.ExperienceAt(index)
}

func (s *Store) UpdateExperience(index int, data Experience) error {
	return s.mutate(func(rd *ResumeData) error {
		return ApplyUpdateExperience(rd, index, data)
	})
}

func (s *Store) DeleteExperience(index int) (Experience, error) {
	var deleted Experience
	err := s.mutate(func(rd *ResumeData) error {
		var err error
		deleted, err = ApplyDeleteExperience(rd, index)
		return err
	})
	if err != nil {
		return Experience{}, err
	}
	return deleted, nil
}

func (s *Store) AddProject(data Project) (int, error) {
	index := 0
	err := s.mutate(func(rd *ResumeData) error {
		index = ApplyAddProject(rd, data)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return index, nil
}

func (s *Store) GetProject(index int) (Project, error) {
	stored, err := s.Load()
	if err != nil {
		return Project{}, err
	}
	return stored.Data.ProjectAt(index)
}

func (s *Store) UpdateProject(index int, data Project) error {
	return s.mutate(func(rd *ResumeData) error {
		return ApplyUpdateProject(rd, index, data)
	})
}

func (s *Store) DeleteProject(index int) (Project, error) {
	var deleted Project
	err := s.mutate(func(rd *ResumeData) error {
		var err error
		deleted, err = ApplyDeleteProject(rd, index)
		return err
	})
	if err != nil {
		return Project{}, err
	}
	return deleted, nil
}

func (s *Store) AddEducation(data Education) (int, error) {
	index := 0
	err := s.mutate(func(rd *ResumeData) error {
		index = ApplyAddEducation(rd, data)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return index, nil
}

func (s *Store) GetEducation(index int) (Education, error) {
	stored, err := s.Load()
	if err != nil {
		return Education{}, err
	}
	return stored.Data.EducationAt(index)
}

func (s *Store) UpdateEducation(index int, data Education) error {
	return s.mutate(func(rd *ResumeData) error {
		return ApplyUpdateEducation(rd, index, data)
	})
}

func (s *Store) DeleteEducation(index int) (Education, error) {
	var deleted Education
	err := s.mutate(func(rd *ResumeData) error {
		var err error
		deleted, err = ApplyDeleteEducation(rd, index)
		return err
	})
	if err != nil {
		return Education{}, err
	}
	return deleted, nil
}

func (s *Store) AddSkill(data SkillGroup) (int, error) {
	index := 0
	err := s.mutate(func(rd *ResumeData) error {
		index = ApplyAddSkill(rd, data)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return index, nil
}

func (s *Store) GetSkill(index int) (SkillGroup, error) {
	stored, err := s.Load()
	if err != nil {
		return SkillGroup{}, err
	}
	return stored.Data.SkillAt(index)
}

func (s *Store) UpdateSkill(index int, data SkillGroup) error {
	return s.mutate(func(rd *ResumeData) error {
		return ApplyUpdateSkill(rd, index, data)
	})
}

func (s *Store) DeleteSkill(index int) (SkillGroup, error) {
	var deleted SkillGroup
	err := s.mutate(func(rd *ResumeData) error {
		var err error
		deleted, err = ApplyDeleteSkill(rd, index)
		return err
	})
	if err != nil {
		return SkillGroup{}, err
	}
	return deleted, nil
}

func (s *Store) AddBullet(parentType string, parentIndex int, text string) (int, error) {
	index := 0
	err := s.mutate(func(rd *ResumeData) error {
		var err error
		index, err = ApplyAddBullet(rd, parentType, parentIndex, text)
		return err
	})
	if err != nil {
		return 0, err
	}
	return index, nil
}

func (s *Store) GetBullet(parentType string, parentIndex, bulletIndex int) (string, error) {
	stored, err := s.Load()
	if err != nil {
		return "", err
	}
	bullets, err := stored.Data.Bullets(parentType, parentIndex)
	if err != nil {
		return "", err
	}
	if err := checkIndex("bullet", bulletIndex, len(bullets)); err != nil {
		return "", err
	}
	return bullets[bulletIndex], nil
}

func (s *Store) UpdateBullet(parentType string, parentIndex, bulletIndex int, text string) error {
	return s.mutate(func(rd *ResumeData) error {
		return ApplyUpdateBullet(rd, parentType, parentIndex, bulletIndex, text)
	})
}

func (s *Store) DeleteBullet(parentType string, parentIndex, bulletIndex int) (string, error) {
	var deleted string
	err := s.mutate(func(rd *ResumeData) error {
		var err error
		deleted, err = ApplyDeleteBullet(rd, parentType, parentIndex, bulletIndex)
		return err
	})
	if err != nil {
		return "", err
	}
	return deleted, nil
}
