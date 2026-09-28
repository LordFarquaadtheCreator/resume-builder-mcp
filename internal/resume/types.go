package resume

import "time"

// ResumeData is the full structured resume. All fields optional except Name.
type ResumeData struct {
	Name        string       `json:"name"`
	Contact     Contact      `json:"contact,omitempty"`
	Education   []Education  `json:"education,omitempty"`
	Skills      []SkillGroup `json:"skills,omitempty"`
	Experiences []Experience `json:"experiences,omitempty"`
	Projects    []Project    `json:"projects,omitempty"`
}

type Contact struct {
	Location string            `json:"location,omitempty"`
	Email    string            `json:"email,omitempty"`
	Links    map[string]string `json:"links,omitempty"`
}

type Education struct {
	Institution string `json:"institution,omitempty"`
	Degree      string `json:"degree,omitempty"`
	Start       string `json:"start,omitempty"`
	End         string `json:"end,omitempty"`
	Location    string `json:"location,omitempty"`
	Link        string `json:"link,omitempty"`
}

type SkillGroup struct {
	Category string `json:"category,omitempty"`
	Values   string `json:"values,omitempty"`
}

type Experience struct {
	Company  string   `json:"company,omitempty"`
	Role     string   `json:"role,omitempty"`
	Start    string   `json:"start,omitempty"`
	End      string   `json:"end,omitempty"`
	Location string   `json:"location,omitempty"`
	Link     string   `json:"link,omitempty"`
	Bullets  []string `json:"bullets,omitempty"`
}

type Project struct {
	Name    string   `json:"name,omitempty"`
	Tech    string   `json:"tech,omitempty"`
	Date    string   `json:"date,omitempty"`
	Link    string   `json:"link,omitempty"`
	Bullets []string `json:"bullets,omitempty"`
}

// StoredResume wraps ResumeData with metadata.
type StoredResume struct {
	Data          ResumeData `json:"data"`
	InitializedAt time.Time  `json:"initializedAt"`
}

// Stats summarizes stored resume content.
type Stats struct {
	Experiences int `json:"experiences"`
	Bullets     int `json:"bullets"`
	Skills      int `json:"skills"`
	Projects    int `json:"projects"`
	Education   int `json:"education"`
}

func ComputeStats(data ResumeData) Stats {
	s := Stats{
		Experiences: len(data.Experiences),
		Skills:      len(data.Skills),
		Projects:    len(data.Projects),
		Education:   len(data.Education),
	}
	for _, e := range data.Experiences {
		s.Bullets += len(e.Bullets)
	}
	for _, p := range data.Projects {
		s.Bullets += len(p.Bullets)
	}
	return s
}
