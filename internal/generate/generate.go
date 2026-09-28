package generate

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/LordFarquaadtheCreator/resume-builder/internal/template"
	"github.com/go-pdf/fpdf"
)

// Input is the agent-facing generate_resume tool input.
type Input struct {
	Data      resume.ResumeData `json:"data"`
	Template  string            `json:"template" jsonschema:"required,Template name (e.g. 'fahad')"`
	OutputDir string            `json:"outputDir,omitempty" jsonschema:"Output directory. Defaults to /tmp."`
}

// Output is returned to the agent.
type Output struct {
	Message    string   `json:"message"`
	OutputPath string   `json:"outputPath"`
	Filename   string   `json:"filename"`
	Trimmed    TrimInfo `json:"trimmed"`
}

// TrimInfo records what was dropped during one-page enforcement.
type TrimInfo struct {
	DroppedBullets     []string `json:"droppedBullets"`
	DroppedExperiences []string `json:"droppedExperiences"`
	DroppedProjects    []string `json:"droppedProjects"`
	FontScale          float64  `json:"fontScale"`
	FitsOnePage        bool     `json:"fitsOnePage"`
}

// Run builds the PDF with one-page enforcement and writes it to disk.
func Run(data resume.ResumeData, templateName, outputDir string) (Output, error) {
	log.Printf("generate.Run: template=%s name=%s outputDir=%s", templateName, data.Name, outputDir)

	tmpl := template.Get(templateName)
	if tmpl == nil {
		log.Printf("generate.Run: ERROR unknown template %q (available: %v)", templateName, template.AvailableTemplates())
		return Output{}, fmt.Errorf("unknown template: %s (available: %v)", templateName, template.AvailableTemplates())
	}

	if err := resume.Validate(data, tmpl.Quotas()); err != nil {
		log.Printf("generate.Run: VALIDATION FAILED: %v", err)
		return Output{}, fmt.Errorf("validation: %w", err)
	}
	log.Printf("generate.Run: validation passed")

	// One-page enforcement: measure → trim → scale
	log.Printf("generate.Run: starting one-page enforcement loop")
	fitted, trimInfo := fitToPage(data, tmpl)
	log.Printf("generate.Run: fitToPage done — fitsOnePage=%v fontScale=%.2f droppedBullets=%d droppedExp=%d droppedProj=%d",
		trimInfo.FitsOnePage, trimInfo.FontScale,
		len(trimInfo.DroppedBullets), len(trimInfo.DroppedExperiences), len(trimInfo.DroppedProjects))

	// Render final PDF
	pdf := fpdf.New("P", "mm", "Letter", "")
	tmpl.Render(pdf, fitted, trimInfo.FontScale)

	// Output path
	dir := outputDir
	if dir == "" {
		dir = "/tmp"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("generate.Run: ERROR creating output dir: %v", err)
		return Output{}, fmt.Errorf("create output dir: %w", err)
	}

	fname := strings.ReplaceAll(data.Name, " ", "") + "Resume.pdf"
	outPath := filepath.Join(dir, fname)
	if err := pdf.OutputFileAndClose(outPath); err != nil {
		log.Printf("generate.Run: ERROR writing PDF: %v", err)
		return Output{}, fmt.Errorf("write pdf: %w", err)
	}
	// fpdf v0.9.0 writes link annotation rects as [x, top, x+w, bottom]
	// instead of [x, bottom, x+w, top]. macOS Preview ignores inverted rects.
	// Fix: swap y1 and y2 in every /Rect inside /Annots.
	if err := fixLinkRects(outPath); err != nil {
		log.Printf("generate.Run: WARNING failed to fix link rects: %v", err)
	}
	log.Printf("generate.Run: PDF written to %s", outPath)

	return Output{
		Message:    fmt.Sprintf("Resume saved to %s", outPath),
		OutputPath: outPath,
		Filename:   fname,
		Trimmed:    trimInfo,
	}, nil
}

// fitToPage runs the measure-scale-trim loop.
// Phase 1: scale UP to fill page when content is sparse (non-destructive).
// Phase 2: scale DOWN + trim content when content overflows (destructive last resort).
// Returns the fitted resume data and info about what was dropped.
func fitToPage(data resume.ResumeData, tmpl template.Renderer) (resume.ResumeData, TrimInfo) {
	info := TrimInfo{
		FontScale: 1.0,
	}

	maxY := tmpl.PageHeight() - tmpl.BottomMargin()
	maxScale := tmpl.MaxFontScale()
	minScale := tmpl.MinFontScale()

	// --- Phase 1: scale up to fill page ---
	// Binary search for the largest scale that still fits, in increments of 0.05.
	for info.FontScale+0.05 <= maxScale {
		next := info.FontScale + 0.05
		pdf := fpdf.New("P", "mm", "Letter", "")
		endY := tmpl.Render(pdf, data, next)
		if endY <= maxY {
			info.FontScale = next
		} else {
			break
		}
	}
	log.Printf("fitToPage: scale-up settled at %.2f", info.FontScale)

	// --- Phase 2: scale down + trim if still overflowing ---
	for {
		pdf := fpdf.New("P", "mm", "Letter", "")
		endY := tmpl.Render(pdf, data, info.FontScale)

		if endY <= maxY {
			info.FitsOnePage = true
			return data, info
		}

		// Pass 1: trim last bullet from oldest experience
		if dropped := trimLastBulletFromOldestExperience(&data, &info); dropped {
			continue
		}

		// Pass 2: drop oldest experience entirely
		if dropped := dropOldestExperience(&data, &info); dropped {
			continue
		}

		// Pass 3: trim last bullet from lowest-ranked project
		if dropped := trimLastBulletFromLastProject(&data, &info); dropped {
			continue
		}

		// Pass 4: drop last project
		if dropped := dropLastProject(&data, &info); dropped {
			continue
		}

		// Pass 5: font scaling (last resort)
		if info.FontScale > minScale {
			info.FontScale -= 0.05
			continue
		}

		// Can't fit further — return what we have
		info.FitsOnePage = false
		log.Printf("fitToPage: WARNING could not fit to one page — fontScale=%.2f (at floor)", info.FontScale)
		return data, info
	}
}

// trimLastBulletFromOldestExperience removes the last bullet from the last
// (oldest, since reverse chronological) experience that still has bullets.
func trimLastBulletFromOldestExperience(data *resume.ResumeData, info *TrimInfo) bool {
	for i := len(data.Experiences) - 1; i >= 0; i-- {
		if len(data.Experiences[i].Bullets) > 1 {
			last := data.Experiences[i].Bullets[len(data.Experiences[i].Bullets)-1]
			info.DroppedBullets = append(info.DroppedBullets, fmt.Sprintf("%s: %s", data.Experiences[i].Company, last))
			data.Experiences[i].Bullets = data.Experiences[i].Bullets[:len(data.Experiences[i].Bullets)-1]
			return true
		}
	}
	return false
}

// dropOldestExperience removes the last (oldest) experience entirely.
func dropOldestExperience(data *resume.ResumeData, info *TrimInfo) bool {
	if len(data.Experiences) <= 1 {
		return false
	}
	last := data.Experiences[len(data.Experiences)-1]
	info.DroppedExperiences = append(info.DroppedExperiences, last.Company)
	data.Experiences = data.Experiences[:len(data.Experiences)-1]
	return true
}

// trimLastBulletFromLastProject removes the last bullet from the last project.
func trimLastBulletFromLastProject(data *resume.ResumeData, info *TrimInfo) bool {
	if len(data.Projects) == 0 {
		return false
	}
	last := &data.Projects[len(data.Projects)-1]
	if len(last.Bullets) > 1 {
		info.DroppedBullets = append(info.DroppedBullets, fmt.Sprintf("%s: %s", last.Name, last.Bullets[len(last.Bullets)-1]))
		last.Bullets = last.Bullets[:len(last.Bullets)-1]
		return true
	}
	return false
}

// dropLastProject removes the last project entirely.
func dropLastProject(data *resume.ResumeData, info *TrimInfo) bool {
	if len(data.Projects) == 0 {
		return false
	}
	last := data.Projects[len(data.Projects)-1]
	info.DroppedProjects = append(info.DroppedProjects, last.Name)
	data.Projects = data.Projects[:len(data.Projects)-1]
	return true
}

// linkRectRe matches /Rect [x1 y1 x2 y2] inside link annotations.
var linkRectRe = regexp.MustCompile(`/Rect \[([\d.]+) ([\d.]+) ([\d.]+) ([\d.]+)\]`)

// fixLinkRects post-processes a PDF file to swap inverted y-coordinates in
// link annotation /Rect entries. go-pdf/fpdf v0.9.0 writes rects as
// [x, top, x+w, bottom] but PDF spec requires [x, bottom, x+w, top].
func fixLinkRects(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read pdf: %w", err)
	}

	fixed := linkRectRe.ReplaceAllStringFunc(string(data), func(s string) string {
		m := linkRectRe.FindStringSubmatch(s)
		x1, _ := strconv.ParseFloat(m[1], 64)
		y1, _ := strconv.ParseFloat(m[2], 64)
		x2, _ := strconv.ParseFloat(m[3], 64)
		y2, _ := strconv.ParseFloat(m[4], 64)
		// swap y1 and y2 if inverted (top > bottom in PDF coords)
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		return fmt.Sprintf("/Rect [%.2f %.2f %.2f %.2f]", x1, y1, x2, y2)
	})

	if err := os.WriteFile(path, []byte(fixed), 0644); err != nil {
		return fmt.Errorf("write pdf: %w", err)
	}
	return nil
}
