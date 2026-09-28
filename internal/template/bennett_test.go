package template

import (
	"testing"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/go-pdf/fpdf"
)

func bennettSample() resume.ResumeData {
	return resume.ResumeData{
		Name:     "Sebastian Bennett",
		Headline: "Professional Accountant",
		Summary:  "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
		Contact: resume.Contact{
			Location: "123 Anywhere St., Any City",
			Email:    "hello@reallygreatsite.com",
			Links:    map[string]string{"phone": "+123-456-7890"},
		},
		Education: []resume.Education{
			{Institution: "Borcelle University", Degree: "Senior Accountant", Start: "2026", End: "2030"},
		},
		Skills: []resume.SkillGroup{
			{Category: "Auditing"},
			{Category: "Financial Accounting"},
			{Category: "Financial Reporting"},
		},
		Experiences: []resume.Experience{
			{
				Company: "Salford & Co.", Role: "Senior Accountant", Start: "2033", End: "2035",
				Bullets: []string{
					"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
					"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.",
				},
			},
		},
	}
}

func renderBennett(t *testing.T, data resume.ResumeData, fontScale float64) float64 {
	t.Helper()
	pdf := fpdf.New("P", "mm", "Letter", "")
	return (&BennettTemplate{}).Render(pdf, data, fontScale)
}

func TestBennettRegistered(t *testing.T) {
	if tmpl := Get("bennett"); tmpl == nil {
		t.Fatal("Get(\"bennett\") = nil, want renderer")
	}

	found := false
	for _, name := range AvailableTemplates() {
		if name == "bennett" {
			found = true
		}
	}
	if !found {
		t.Fatalf("AvailableTemplates() = %v, want it to contain \"bennett\"", AvailableTemplates())
	}
}

func TestBennettRenderFitsPage(t *testing.T) {
	tmpl := &BennettTemplate{}
	endY := renderBennett(t, bennettSample(), 1.0)

	maxY := tmpl.PageHeight() - tmpl.BottomMargin()
	if endY > maxY {
		t.Fatalf("endY = %.1fmm exceeds usable page height %.1fmm", endY, maxY)
	}
	if endY < bennettMargin {
		t.Fatalf("endY = %.1fmm, want content below the top margin", endY)
	}
}

// TestBennettHeadlineAndSummaryAddContent verifies the header/summary fields
// reach the page: dropping them shortens the render.
func TestBennettHeadlineAndSummaryAddContent(t *testing.T) {
	full := renderBennett(t, bennettSample(), 1.0)

	trimmed := bennettSample()
	trimmed.Headline = ""
	trimmed.Summary = ""
	without := renderBennett(t, trimmed, 1.0)

	if without >= full {
		t.Fatalf("render without headline/summary = %.1fmm, want less than %.1fmm", without, full)
	}
}

// TestBennettScalesDownWithScale verifies fontScale shrinks the render.
func TestBennettFontScaleShrinks(t *testing.T) {
	data := bennettSample()
	normal := renderBennett(t, data, 1.0)
	small := renderBennett(t, data, 0.8)

	if small >= normal {
		t.Fatalf("render at 0.8 scale = %.1fmm, want less than 1.0 scale = %.1fmm", small, normal)
	}
}

func TestBennettEmptyDataRenders(t *testing.T) {
	endY := renderBennett(t, resume.ResumeData{}, 1.0)

	tmpl := &BennettTemplate{}
	if endY > tmpl.PageHeight()-tmpl.BottomMargin() {
		t.Fatalf("endY = %.1fmm exceeds usable page height for empty data", endY)
	}
}

// TestBennettSkillsGridFits verifies five skill groups stay on one line each
// in the three-column grid and fit the page.
func TestBennettSkillsGridFits(t *testing.T) {
	data := bennettSample()
	data.Skills = []resume.SkillGroup{
		{Category: "Languages", Values: "Go, TypeScript, Python"},
		{Category: "Frameworks", Values: "React, Node.js"},
		{Category: "Infrastructure", Values: "AWS, Docker, Kubernetes"},
		{Category: "Data", Values: "PostgreSQL, Redis, Kafka"},
		{Category: "Practices", Values: "CI/CD, observability"},
	}

	tmpl := &BennettTemplate{}
	endY := renderBennett(t, data, 1.0)
	if endY > tmpl.PageHeight()-tmpl.BottomMargin() {
		t.Fatalf("endY = %.1fmm exceeds usable page height with 5 skill groups", endY)
	}
}

func TestBennettQuotasUseDefaults(t *testing.T) {
	tmpl := &BennettTemplate{}
	if tmpl.Quotas() != resume.DefaultQuotas {
		t.Fatalf("Quotas() = %+v, want DefaultQuotas %+v", tmpl.Quotas(), resume.DefaultQuotas)
	}
	if tmpl.MinFontScale() > tmpl.MaxFontScale() {
		t.Fatalf("MinFontScale %.2f > MaxFontScale %.2f", tmpl.MinFontScale(), tmpl.MaxFontScale())
	}
}
