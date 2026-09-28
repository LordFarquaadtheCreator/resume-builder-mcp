package generate

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
)

// pdfPageCount parses raw PDF bytes and counts page objects.
// fpdf writes /Type /Page for each page and /Type /Pages for the page tree.
// We count all /Type /Page occurrences and subtract /Type /Pages matches.
func pdfPageCount(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pdf: %v", err)
	}
	allPage := regexp.MustCompile(`/Type\s*/Page`).FindAll(data, -1)
	pagesOnly := regexp.MustCompile(`/Type\s*/Pages`).FindAll(data, -1)
	return len(allPage) - len(pagesOnly)
}

// pdfContainsText checks if the PDF file contains the given string in its raw bytes.
// fpdf embeds text with font encoding, so we check for the raw string presence.
func pdfContainsText(path string, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

func sampleResume() resume.ResumeData {
	return resume.ResumeData{
		Name: "Test User",
		Contact: resume.Contact{
			Location: "NYC",
			Email:    "test@example.com",
			Links: map[string]string{
				"linkedin": "https://linkedin.com/in/test",
				"github":   "https://github.com/test",
			},
		},
		Education: []resume.Education{
			{
				Institution: "Test University",
				Degree:      "B.S. Computer Science",
				End:         "Class of 2025",
				Location:    "New York, NY",
			},
		},
		Skills: []resume.SkillGroup{
			{Category: "Languages", Values: "Go, TypeScript, Python"},
			{Category: "Tools", Values: "Docker, Git, Kubernetes"},
		},
		Experiences: []resume.Experience{
			{
				Company:  "Tech Corp",
				Role:     "Software Engineer",
				Start:    "Jan. 2024",
				End:      "Present",
				Location: "Remote",
				Link:     "https://techcorp.com",
				Bullets: []string{
					"Built a scalable API serving 1M requests/day.",
					"Led migration to microservices architecture.",
					"Reduced latency by 40% through caching optimization.",
				},
			},
		},
		Projects: []resume.Project{
			{
				Name: "OpenTool",
				Tech: "Go",
				Date: "Mar. 2025",
				Link: "https://github.com/test/opentool",
				Bullets: []string{
					"CLI tool for automating dev workflows.",
				},
			},
		},
	}
}

// overflowResume has enough content to trigger one-page trimming.
func overflowResume() resume.ResumeData {
	r := sampleResume()
	r.Name = "Overflow User"
	r.Experiences = []resume.Experience{
		{
			Company: "Company One", Role: "Senior Engineer", Start: "Jan. 2025", End: "Present",
			Location: "Remote", Link: "https://one.com",
			Bullets: []string{
				"Built a scalable API serving 1M requests per day with Go and PostgreSQL.",
				"Led migration to microservices, reducing deploy time by 60%.",
				"Reduced latency by 40% through multi-tier caching with Redis.",
				"Designed event-driven architecture with Kafka processing 10M events/day.",
				"Mentored 3 junior engineers and established code review standards.",
			},
		},
		{
			Company: "Company Two", Role: "Software Engineer", Start: "Jun. 2023", End: "Dec. 2024",
			Location: "NYC", Link: "https://two.com",
			Bullets: []string{
				"Developed React frontend with TypeScript serving 500K monthly users.",
				"Implemented CI/CD pipeline with GitHub Actions, cutting deploy time by 50%.",
				"Built GraphQL gateway unifying 12 backend services into single API.",
				"Optimized PostgreSQL queries, reducing p99 latency from 800ms to 120ms.",
				"Led adoption of integration testing, increasing coverage from 20% to 85%.",
			},
		},
		{
			Company: "Company Three", Role: "Junior Developer", Start: "Jan. 2022", End: "May 2023",
			Location: "Boston", Link: "https://three.com",
			Bullets: []string{
				"Built REST API in Node.js for e-commerce platform handling 10K orders/day.",
				"Implemented real-time notifications with WebSocket and Redis pub/sub.",
				"Created admin dashboard with Vue.js for managing product catalog.",
				"Automated data pipeline with Python, saving 20 hours/week of manual work.",
				"Set up monitoring with Prometheus and Grafana for 99.9% uptime tracking.",
			},
		},
		{
			Company: "Company Four", Role: "Intern", Start: "Jun. 2021", End: "Dec. 2021",
			Location: "Remote", Link: "https://four.com",
			Bullets: []string{
				"Developed Python scripts for data analysis and reporting automation.",
				"Built internal tool with Flask for tracking project milestones.",
				"Contributed to open-source documentation for internal libraries.",
				"Assisted in database migration from MySQL to PostgreSQL.",
				"Participated in code reviews and agile development ceremonies.",
			},
		},
	}
	r.Projects = []resume.Project{
		{
			Name: "Project Alpha", Tech: "Go, Docker", Date: "Feb. 2025",
			Link: "https://github.com/test/alpha",
			Bullets: []string{
				"Distributed task queue with Go and Redis, processing 100K jobs/day.",
				"Deployed on Kubernetes with auto-scaling and health checks.",
			},
		},
		{
			Name: "Project Beta", Tech: "TypeScript", Date: "Nov. 2024",
			Link: "https://github.com/test/beta",
			Bullets: []string{
				"Real-time collaboration editor with WebSocket and CRDT.",
				"Built plugin system supporting 20+ third-party extensions.",
			},
		},
		{
			Name: "Project Gamma", Tech: "Python", Date: "Aug. 2024",
			Link: "https://github.com/test/gamma",
			Bullets: []string{
				"ML-powered recommendation engine with scikit-learn.",
				"REST API with FastAPI serving 50K requests/hour.",
			},
		},
		{
			Name: "Project Delta", Tech: "Rust", Date: "May. 2024",
			Link: "https://github.com/test/delta",
			Bullets: []string{
				"High-performance file parser processing 1GB/s.",
				"CLI tool with zero-copy parsing and memory-mapped I/O.",
			},
		},
	}
	return r
}

func TestRunFahadTemplateProducesOnePagePDF(t *testing.T) {
	data := sampleResume()
	outDir := t.TempDir()

	out, err := Run(data, "fahad", outDir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify output path
	expectedFname := "TestUserResume.pdf"
	if out.Filename != expectedFname {
		t.Fatalf("filename = %q, want %q", out.Filename, expectedFname)
	}
	expectedPath := filepath.Join(outDir, expectedFname)
	if out.OutputPath != expectedPath {
		t.Fatalf("outputPath = %q, want %q", out.OutputPath, expectedPath)
	}

	// Verify file exists
	if _, err := os.Stat(out.OutputPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	// Verify page count = 1
	pages := pdfPageCount(t, out.OutputPath)
	if pages != 1 {
		t.Fatalf("page count = %d, want 1", pages)
	}

	// Verify one-page enforcement reported success
	if !out.Trimmed.FitsOnePage {
		t.Fatal("FitsOnePage = false, want true")
	}
}

func TestRunUnknownTemplateFails(t *testing.T) {
	data := sampleResume()
	_, err := Run(data, "nonexistent", t.TempDir())
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
	if !strings.Contains(err.Error(), "unknown template") {
		t.Fatalf("error = %q, want 'unknown template'", err.Error())
	}
}

func TestRunEmptyTemplateFails(t *testing.T) {
	data := sampleResume()
	_, err := Run(data, "", t.TempDir())
	if err == nil {
		t.Fatal("expected error for empty template")
	}
}

func TestRunFilenameNoSpaces(t *testing.T) {
	data := sampleResume()
	data.Name = "John Doe Smith"
	out, err := Run(data, "fahad", t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Filename != "JohnDoeSmithResume.pdf" {
		t.Fatalf("filename = %q, want %q", out.Filename, "JohnDoeSmithResume.pdf")
	}
}

func TestRunDefaultOutputDir(t *testing.T) {
	data := sampleResume()
	data.Name = "DefaultDirTest"
	out, err := Run(data, "fahad", "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.HasPrefix(out.OutputPath, "/tmp/") {
		t.Fatalf("outputPath = %q, want /tmp/ prefix", out.OutputPath)
	}
	// cleanup
	defer os.Remove(out.OutputPath)
}

func TestRunOverflowTrimsToOnePage(t *testing.T) {
	data := overflowResume()
	out, err := Run(data, "fahad", t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// PDF must still be one page
	pages := pdfPageCount(t, out.OutputPath)
	if pages != 1 {
		t.Fatalf("page count = %d, want 1 (overflow should be trimmed)", pages)
	}

	// One-page enforcement should report success
	if !out.Trimmed.FitsOnePage {
		t.Fatal("FitsOnePage = false after trimming, want true")
	}

	// Should have dropped something — bullets, experiences, or projects
	droppedCount := len(out.Trimmed.DroppedBullets) + len(out.Trimmed.DroppedExperiences) + len(out.Trimmed.DroppedProjects)
	if droppedCount == 0 && out.Trimmed.FontScale >= 1.0 {
		t.Fatal("expected some trimming activity (dropped items or font scaling)")
	}
}

func TestRunOverflowDropsBulletsBeforeExperiences(t *testing.T) {
	data := overflowResume()
	out, err := Run(data, "fahad", t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// If experiences were dropped, bullets must have been dropped first
	// (trim order: bullets → experiences → projects → font)
	if len(out.Trimmed.DroppedExperiences) > 0 && len(out.Trimmed.DroppedBullets) == 0 {
		t.Fatal("experiences dropped but no bullets dropped — trim order violated")
	}
}

func TestRunNullFieldsOmitted(t *testing.T) {
	data := resume.ResumeData{
		Name: "Minimal User",
		Contact: resume.Contact{
			Email: "min@example.com",
			Links: map[string]string{
				"github": "https://github.com/min",
			},
		},
		Experiences: []resume.Experience{
			{
				Company: "Solo",
				Role:    "Founder",
				End:     "Present",
				Bullets: []string{"Did everything."},
			},
		},
	}
	out, err := Run(data, "fahad", t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Should still be one page
	pages := pdfPageCount(t, out.OutputPath)
	if pages != 1 {
		t.Fatalf("page count = %d, want 1", pages)
	}

	// Should fit without trimming
	if !out.Trimmed.FitsOnePage {
		t.Fatal("FitsOnePage = false for minimal resume")
	}
}

func TestRunQuotaValidationFails(t *testing.T) {
	data := sampleResume()
	// Add 7 experiences (max is 6)
	for i := 0; i < 6; i++ {
		data.Experiences = append(data.Experiences, resume.Experience{
			Company: "Extra Corp", Role: "Engineer", End: "Present",
			Bullets: []string{"Did stuff."},
		})
	}
	_, err := Run(data, "fahad", t.TempDir())
	if err == nil {
		t.Fatal("expected error for exceeding quota")
	}
	if !strings.Contains(err.Error(), "quota") && !strings.Contains(err.Error(), "validation") {
		t.Fatalf("error = %q, want 'quota' or 'validation'", err.Error())
	}
}

// TestRunFahadTemplateProducesNonEmptyPDF verifies the PDF has meaningful content
func TestRunFahadTemplateProducesNonEmptyPDF(t *testing.T) {
	data := sampleResume()
	out, err := Run(data, "fahad", t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	info, err := os.Stat(out.OutputPath)
	if err != nil {
		t.Fatalf("stat pdf: %v", err)
	}
	if info.Size() < 1000 {
		t.Fatalf("PDF size = %d bytes, expected > 1000 for content-rich resume", info.Size())
	}
}
