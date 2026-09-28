package template

import (
	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/go-pdf/fpdf"
)

// Renderer renders resume data onto a PDF. Each template implements this.
type Renderer interface {
	// Name returns the template identifier.
	Name() string

	// Quotas returns the guard rail limits for this template.
	Quotas() resume.Quotas

	// Render draws the resume onto the given PDF. fontScale adjusts all font
	// sizes and spacing by this multiplier (1.0 = normal). Returns the Y
	// position after rendering all content.
	Render(pdf *fpdf.Fpdf, data resume.ResumeData, fontScale float64) float64

	// PageHeight returns the usable page height in mm (after margins).
	PageHeight() float64

	// BottomMargin returns the bottom margin in mm.
	BottomMargin() float64

	// MaxFontScale returns the upper bound for fontScale. The generator may
	// scale up to fill a sparse page but never beyond this.
	MaxFontScale() float64

	// MinFontScale returns the lower bound for fontScale. The generator may
	// scale down to fit overflow but never below this.
	MinFontScale() float64
}

// Get returns a template by name. Returns nil if not found.
func Get(name string) Renderer {
	switch name {
	case "fahad":
		return &FahadTemplate{}
	default:
		return nil
	}
}

// AvailableTemplates returns the list of registered template names.
func AvailableTemplates() []string {
	return []string{"fahad"}
}
