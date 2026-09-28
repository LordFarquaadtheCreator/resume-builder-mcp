package resume

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const resumeFile = "resume.json"

// Store handles disk persistence of resume data.
type Store struct {
	dataDir string
}

func NewStore(dataDir string) *Store {
	return &Store{dataDir: dataDir}
}

// Save writes resume data to disk, overwriting any existing data.
func (s *Store) Save(data ResumeData) error {
	if data.Name == "" {
		return fmt.Errorf("name is required")
	}
	stored := StoredResume{
		Data:        data,
		InitializedAt: time.Now(),
	}
	b, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal resume: %w", err)
	}
	path := filepath.Join(s.dataDir, resumeFile)
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
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
			return nil, fmt.Errorf("no resume found — call init_resume first")
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
