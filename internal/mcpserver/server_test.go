package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
)

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

func testDeps(t *testing.T) deps {
	dir := t.TempDir()
	return deps{
		ResumeStore: resume.NewStore(dir),
		VectorStore: nil, // not needed for manual mode
		ConfigStore: nil, // not needed for manual mode
	}
}

func sampleResumeData() resume.ResumeData {
	return resume.ResumeData{
		Name: "E2E User",
		Contact: resume.Contact{
			Location: "Queens, NYC",
			Email:    "e2e@example.com",
			Links: map[string]string{
				"linkedin": "https://linkedin.com/in/e2e",
				"github":   "https://github.com/e2e",
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
			{Category: "Languages", Values: "Go, TypeScript, Python, Swift"},
			{Category: "Cloud", Values: "AWS, Docker, Kubernetes"},
		},
		Experiences: []resume.Experience{
			{
				Company:  "E2E Corp",
				Role:     "Senior Engineer",
				Start:    "Jan. 2024",
				End:      "Present",
				Location: "Remote",
				Link:     "https://e2ecorp.com",
				Bullets: []string{
					"Built scalable API serving 1M requests/day.",
					"Led team of 5 engineers to deliver platform redesign.",
					"Reduced infrastructure costs by 35% through optimization.",
				},
			},
			{
				Company:  "Previous Co",
				Role:     "Software Engineer",
				Start:    "Jun. 2022",
				End:      "Dec. 2023",
				Location: "NYC",
				Bullets: []string{
					"Developed React frontend with TypeScript.",
					"Implemented CI/CD pipeline reducing deploy time by 50%.",
				},
			},
		},
		Projects: []resume.Project{
			{
				Name: "E2E Tool",
				Tech: "Go",
				Date: "Mar. 2025",
				Link: "https://github.com/e2e/tool",
				Bullets: []string{
					"CLI tool for automating deployment workflows.",
				},
			},
		},
	}
}

// TestE2EGenerateManualMode tests the full handler pipeline:
// handleGenerateResume → generate.Run → PDF output
func TestE2EGenerateManualMode(t *testing.T) {
	d := testDeps(t)
	outDir := t.TempDir()
	data := sampleResumeData()

	_, out, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:      "manual",
			Data:      &data,
			Template:  "fahad",
			OutputDir: outDir,
		},
		d,
	)
	if err != nil {
		t.Fatalf("handleGenerateResume: %v", err)
	}

	// Verify filename
	expectedFname := "E2EUserResume.pdf"
	if out.Filename != expectedFname {
		t.Fatalf("filename = %q, want %q", out.Filename, expectedFname)
	}

	// Verify output path
	expectedPath := filepath.Join(outDir, expectedFname)
	if out.OutputPath != expectedPath {
		t.Fatalf("outputPath = %q, want %q", out.OutputPath, expectedPath)
	}

	// Verify file exists on disk
	if _, err := os.Stat(out.OutputPath); err != nil {
		t.Fatalf("PDF not created: %v", err)
	}

	// Verify page count = 1
	pages := pdfPageCount(t, out.OutputPath)
	if pages != 1 {
		t.Fatalf("page count = %d, want 1", pages)
	}

	// Verify one-page enforcement
	if !out.Trimmed.FitsOnePage {
		t.Fatal("FitsOnePage = false, want true")
	}
}

// TestE2EGenerateUnknownTemplate tests that requesting an unknown template fails
func TestE2EGenerateUnknownTemplate(t *testing.T) {
	d := testDeps(t)
	data := sampleResumeData()

	_, _, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:      "manual",
			Data:      &data,
			Template:  "nonexistent",
			OutputDir: t.TempDir(),
		},
		d,
	)
	if err == nil {
		t.Fatal("expected error for unknown template")
	}
	if !strings.Contains(err.Error(), "unknown template") {
		t.Fatalf("error = %q, want 'unknown template'", err.Error())
	}
}

// TestE2EGenerateMissingTemplate tests that missing template fails
func TestE2EGenerateMissingTemplate(t *testing.T) {
	d := testDeps(t)
	data := sampleResumeData()

	_, _, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:      "manual",
			Data:      &data,
			Template:  "",
			OutputDir: t.TempDir(),
		},
		d,
	)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	if !strings.Contains(err.Error(), "template is required") {
		t.Fatalf("error = %q, want 'template is required'", err.Error())
	}
}

// TestE2EGenerateInvalidMode tests that invalid mode fails
func TestE2EGenerateInvalidMode(t *testing.T) {
	d := testDeps(t)
	data := sampleResumeData()

	_, _, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:      "invalid",
			Data:      &data,
			Template:  "fahad",
			OutputDir: t.TempDir(),
		},
		d,
	)
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "mode must be") {
		t.Fatalf("error = %q, want 'mode must be'", err.Error())
	}
}

// TestE2EGenerateManualWithoutData tests that manual mode without data fails
func TestE2EGenerateManualWithoutData(t *testing.T) {
	d := testDeps(t)

	_, _, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:      "manual",
			Data:      nil,
			Template:  "fahad",
			OutputDir: t.TempDir(),
		},
		d,
	)
	if err == nil {
		t.Fatal("expected error for manual mode without data")
	}
	if !strings.Contains(err.Error(), "data is required") {
		t.Fatalf("error = %q, want 'data is required'", err.Error())
	}
}

// TestE2EGenerateAutoWithoutConfig tests that auto mode fails without embedding config
func TestE2EGenerateAutoWithoutConfig(t *testing.T) {
	d := testDeps(t)

	_, _, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:      "auto",
			Query:     "software engineer job",
			Template:  "fahad",
			OutputDir: t.TempDir(),
		},
		d,
	)
	if err == nil {
		t.Fatal("expected error for auto mode without config")
	}
}

// TestE2EGenerateAutoWithoutQuery tests that auto mode without query fails
func TestE2EGenerateAutoWithoutQuery(t *testing.T) {
	d := testDeps(t)

	_, _, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:      "auto",
			Query:     "",
			Template:  "fahad",
			OutputDir: t.TempDir(),
		},
		d,
	)
	if err == nil {
		t.Fatal("expected error for auto mode without query")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("error = %q, want 'query is required'", err.Error())
	}
}

// TestE2EGenerateOverflowTrimsToOnePage tests that overflow content is trimmed
// and the output PDF is still one page
func TestE2EGenerateOverflowTrimsToOnePage(t *testing.T) {
	d := testDeps(t)
	outDir := t.TempDir()

	data := sampleResumeData()
	data.Name = "Overflow E2E"
	// Add experiences with max bullets to trigger overflow (stay within quota of 6)
	for i := 0; i < 4; i++ {
		data.Experiences = append(data.Experiences, resume.Experience{
			Company:  "Overflow Corp",
			Role:     "Engineer",
			Start:    "Jan. 2020",
			End:      "Dec. 2021",
			Location: "Remote",
			Bullets: []string{
				"Built scalable microservice handling 500K requests per second with Go.",
				"Implemented distributed tracing with Jaeger across 20 services.",
				"Optimized database queries reducing p99 latency from 500ms to 50ms.",
				"Led incident response and post-mortem reviews for critical systems.",
				"Designed fault-tolerant architecture with circuit breakers and retries.",
			},
		})
	}
	// Add projects to increase content
	for i := 0; i < 3; i++ {
		data.Projects = append(data.Projects, resume.Project{
			Name: "OverflowProject",
			Tech: "Go",
			Date: "Jan. 2024",
			Bullets: []string{
				"High-throughput data pipeline processing 10M records per hour.",
				"Deployed with Kubernetes and Helm across 3 regions.",
			},
		})
	}

	_, out, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:      "manual",
			Data:      &data,
			Template:  "fahad",
			OutputDir: outDir,
		},
		d,
	)
	if err != nil {
		t.Fatalf("handleGenerateResume: %v", err)
	}

	// PDF must be one page even with overflow content
	pages := pdfPageCount(t, out.OutputPath)
	if pages != 1 {
		t.Fatalf("page count = %d, want 1 (overflow should be trimmed)", pages)
	}

	// One-page enforcement should report success
	if !out.Trimmed.FitsOnePage {
		t.Fatal("FitsOnePage = false after trimming, want true")
	}

	// Should have dropped something
	droppedCount := len(out.Trimmed.DroppedBullets) + len(out.Trimmed.DroppedExperiences) + len(out.Trimmed.DroppedProjects)
	if droppedCount == 0 && out.Trimmed.FontScale >= 1.0 {
		t.Fatal("expected trimming activity for overflow content")
	}
}

// TestE2EGenerateDefaultOutputDir tests that output defaults to /tmp
func TestE2EGenerateDefaultOutputDir(t *testing.T) {
	d := testDeps(t)
	data := sampleResumeData()
	data.Name = "DefaultTest"

	_, out, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:     "manual",
			Data:     &data,
			Template: "fahad",
		},
		d,
	)
	if err != nil {
		t.Fatalf("handleGenerateResume: %v", err)
	}

	if !strings.HasPrefix(out.OutputPath, "/tmp/") {
		t.Fatalf("outputPath = %q, want /tmp/ prefix", out.OutputPath)
	}
	defer os.Remove(out.OutputPath)
}

// TestE2EGenerateResultJSON tests that the MCP result contains valid JSON
func TestE2EGenerateResultJSON(t *testing.T) {
	d := testDeps(t)
	data := sampleResumeData()

	result, _, err := handleGenerateResume(
		context.Background(),
		&mcp.CallToolRequest{},
		GenerateResumeInput{
			Mode:      "manual",
			Data:      &data,
			Template:  "fahad",
			OutputDir: t.TempDir(),
		},
		d,
	)
	if err != nil {
		t.Fatalf("handleGenerateResume: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Content) == 0 {
		t.Fatal("result.Content is empty")
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("result.Content[0] is not TextContent")
	}
	if !strings.Contains(text.Text, "outputPath") {
		t.Fatalf("result JSON does not contain 'outputPath': %s", text.Text)
	}
	if !strings.Contains(text.Text, "fitsOnePage") {
		t.Fatalf("result JSON does not contain 'fitsOnePage': %s", text.Text)
	}
}
