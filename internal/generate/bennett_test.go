package generate

import (
	"testing"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
)

// TestRunBennettTemplateProducesOnePagePDF covers the bennett template end to
// end, including the new headline/summary fields.
func TestRunBennettTemplateProducesOnePagePDF(t *testing.T) {
	data := sampleResume()
	data.Headline = "Software Engineer"
	data.Summary = "Engineer focused on developer tooling and reliable services."

	out, err := Run(data, "bennett", t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if pages := pdfPageCount(t, out.OutputPath); pages != 1 {
		t.Fatalf("page count = %d, want 1", pages)
	}
	if !out.Trimmed.FitsOnePage {
		t.Fatal("FitsOnePage = false, want true")
	}
}

// TestRunBennettOverflowTrimsToOnePage verifies one-page enforcement drives the
// bennett layout through the same trim loop as fahad.
func TestRunBennettOverflowTrimsToOnePage(t *testing.T) {
	data := overflowResume()
	data.Headline = "Software Engineer"
	data.Summary = "Engineer focused on developer tooling and reliable services."

	out, err := Run(data, "bennett", t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if pages := pdfPageCount(t, out.OutputPath); pages != 1 {
		t.Fatalf("page count = %d, want 1 (overflow should be trimmed)", pages)
	}
	if !out.Trimmed.FitsOnePage {
		t.Fatal("FitsOnePage = false after trimming, want true")
	}
}

// TestRunBennettMinimalData checks a name-only resume renders without panicking.
func TestRunBennettMinimalData(t *testing.T) {
	data := resume.ResumeData{Name: "Minimal User"}

	out, err := Run(data, "bennett", t.TempDir())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if pages := pdfPageCount(t, out.OutputPath); pages != 1 {
		t.Fatalf("page count = %d, want 1", pages)
	}
}
