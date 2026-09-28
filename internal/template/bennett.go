package template

import (
	"strings"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/go-pdf/fpdf"
)

// BennettTemplate renders a modern sans-serif resume: centered uppercase name
// with a headline, a contact strip between full-width rules, uppercase section
// headings over rules, and a multi-column skills grid.
type BennettTemplate struct{}

const (
	bennettPageWidth    = 215.9 // Letter width in mm
	bennettPageHeight   = 279.4 // Letter height in mm
	bennettMargin       = 18.0
	bennettBottomMargin = 18.0
	bennettContentWidth = bennettPageWidth - 2*bennettMargin
)

// Palette (RGB).
const (
	bennettInkR, bennettInkG, bennettInkB       = 30, 30, 30    // name, headings, entry titles
	bennettBodyR, bennettBodyG, bennettBodyB    = 55, 55, 55    // body text
	bennettMetaR, bennettMetaG, bennettMetaB    = 90, 90, 90    // company/institution, icons
	bennettMutedR, bennettMutedG, bennettMutedB = 152, 152, 152 // dates, fragments
	bennettRuleR, bennettRuleG, bennettRuleB    = 70, 70, 70
)

// Base sizes in points, before font scaling.
const (
	bennettNamePT         = 25.0
	bennettHeadlinePT     = 13.5
	bennettContactPT      = 9.5
	bennettSectionPT      = 13.5
	bennettMetaPT         = 10.0
	bennettTitlePT        = 11.0
	bennettBodyPT         = 10.0
	bennettIconMM         = 3.4
	bennettBulletIndentMM = 4.0
)

func (t *BennettTemplate) Name() string { return "bennett" }

func (t *BennettTemplate) Quotas() resume.Quotas { return resume.DefaultQuotas }

func (t *BennettTemplate) PageHeight() float64 { return bennettPageHeight }

func (t *BennettTemplate) BottomMargin() float64 { return bennettBottomMargin }

// MaxFontScale ceilings body text (10pt base) at 14pt.
func (t *BennettTemplate) MaxFontScale() float64 { return 1.4 }

// MinFontScale floors body text (10pt base) at 10pt.
func (t *BennettTemplate) MinFontScale() float64 { return 1.0 }

func (t *BennettTemplate) scale(v, fontScale float64) float64 { return v * fontScale }

// ptMM converts points to millimetres.
func ptMM(pt float64) float64 { return pt * 0.3528 }

func (t *BennettTemplate) Render(pdf *fpdf.Fpdf, data resume.ResumeData, fontScale float64) float64 {
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetAutoPageBreak(false, 0)
	pdf.SetCellMargin(0)
	pdf.AddPage()

	y := bennettMargin
	y = t.renderName(pdf, data, tr, fontScale, y)
	y = t.renderHeadline(pdf, data, tr, fontScale, y)
	y = t.renderContactBlock(pdf, data, tr, fontScale, y)

	sectionGap := t.scale(5.0, fontScale)

	if summary := strings.TrimSpace(data.Summary); summary != "" {
		y = t.renderSectionHeader(pdf, "About Me", tr, fontScale, y)
		y = t.renderParagraph(pdf, summary, tr, fontScale, y)
		y += sectionGap
	}

	if len(data.Education) > 0 {
		y = t.renderSectionHeader(pdf, "Education", tr, fontScale, y)
		for _, edu := range data.Education {
			y = t.renderEducation(pdf, edu, tr, fontScale, y)
		}
		y += sectionGap
	}

	if len(data.Experiences) > 0 {
		y = t.renderSectionHeader(pdf, "Work Experience", tr, fontScale, y)
		for _, exp := range data.Experiences {
			y = t.renderExperience(pdf, exp, tr, fontScale, y)
		}
		y += sectionGap
	}

	if len(data.Skills) > 0 {
		y = t.renderSectionHeader(pdf, "Skills", tr, fontScale, y)
		y = t.renderSkills(pdf, data.Skills, tr, fontScale, y)
		y += sectionGap
	}

	if len(data.Projects) > 0 {
		y = t.renderSectionHeader(pdf, "Projects", tr, fontScale, y)
		for _, proj := range data.Projects {
			y = t.renderProject(pdf, proj, tr, fontScale, y)
		}
	}

	return y
}

// renderName draws the centered uppercase name with slight letter tracking.
func (t *BennettTemplate) renderName(pdf *fpdf.Fpdf, data resume.ResumeData, tr func(string) string, fontScale, y float64) float64 {
	name := strings.TrimSpace(data.Name)
	if name == "" {
		return y
	}
	size := t.scale(bennettNamePT, fontScale)
	tracking := t.scale(0.7, fontScale)

	pdf.SetFont("Helvetica", "B", size)
	pdf.SetTextColor(bennettInkR, bennettInkG, bennettInkB)

	text := tr(strings.ToUpper(name))
	width := bennettTrackedWidth(pdf, text, tracking)
	baseline := y + ptMM(size)*0.75
	bennettDrawTracked(pdf, bennettPageWidth/2-width/2, baseline, text, tracking)

	return y + ptMM(size)*1.15
}

// renderHeadline draws the centered subtitle under the name.
func (t *BennettTemplate) renderHeadline(pdf *fpdf.Fpdf, data resume.ResumeData, tr func(string) string, fontScale, y float64) float64 {
	headline := strings.TrimSpace(data.Headline)
	if headline == "" {
		return y
	}
	size := t.scale(bennettHeadlinePT, fontScale)
	h := t.scale(ptMM(bennettHeadlinePT)*1.4, fontScale)

	pdf.SetFont("Helvetica", "", size)
	pdf.SetTextColor(bennettMetaR, bennettMetaG, bennettMetaB)
	pdf.SetXY(bennettMargin, y)
	pdf.CellFormat(bennettContentWidth, h, tr(headline), "", 1, "C", false, 0, "")

	return y + h
}

// renderContactBlock draws the rule / contact strip / rule band under the
// header. With no contact data it collapses to a single rule.
func (t *BennettTemplate) renderContactBlock(pdf *fpdf.Fpdf, data resume.ResumeData, tr func(string) string, fontScale, y float64) float64 {
	thickness := t.scale(0.3, fontScale)
	items := bennettContactItems(data, tr)

	y += t.scale(2.2, fontScale)
	if len(items) == 0 {
		bennettRule(pdf, y, thickness)
		return y + t.scale(4.0, fontScale)
	}

	bennettRule(pdf, y, thickness)
	y += t.scale(1.7, fontScale)
	y = t.renderContactRow(pdf, items, fontScale, y)
	y += t.scale(1.5, fontScale)
	bennettRule(pdf, y, thickness)

	return y + t.scale(5.5, fontScale)
}

func (t *BennettTemplate) renderSectionHeader(pdf *fpdf.Fpdf, title string, tr func(string) string, fontScale, y float64) float64 {
	size := t.scale(bennettSectionPT, fontScale)
	h := t.scale(ptMM(bennettSectionPT)*1.2, fontScale)

	pdf.SetFont("Helvetica", "B", size)
	pdf.SetTextColor(bennettInkR, bennettInkG, bennettInkB)
	pdf.SetXY(bennettMargin, y)
	pdf.CellFormat(bennettContentWidth, h, tr(strings.ToUpper(title)), "", 1, "L", false, 0, "")

	y += h + t.scale(0.9, fontScale)
	bennettRule(pdf, y, t.scale(0.3, fontScale))

	return y + t.scale(3.0, fontScale)
}

// renderParagraph draws a justified full-width body paragraph.
func (t *BennettTemplate) renderParagraph(pdf *fpdf.Fpdf, text string, tr func(string) string, fontScale, y float64) float64 {
	size := t.scale(bennettBodyPT, fontScale)
	lineH := t.scale(ptMM(bennettBodyPT)*1.45, fontScale)

	pdf.SetFont("Helvetica", "", size)
	pdf.SetTextColor(bennettBodyR, bennettBodyG, bennettBodyB)

	body := tr(text)
	lines := pdf.SplitLines([]byte(body), bennettContentWidth)
	pdf.SetXY(bennettMargin, y)
	pdf.MultiCell(bennettContentWidth, lineH, body, "", "J", false)

	return y + float64(len(lines))*lineH
}

func (t *BennettTemplate) renderEducation(pdf *fpdf.Fpdf, edu resume.Education, tr func(string) string, fontScale, y float64) float64 {
	runs := make([]bennettRun, 0, 3)
	if institution := strings.TrimSpace(edu.Institution); institution != "" {
		runs = append(runs, bennettPrimaryRun(tr(institution), edu.Link))
	}
	if dates := bennettDateRange(edu.Start, edu.End); dates != "" {
		runs = append(runs, bennettMutedRuns(tr(dates))...)
	}
	y = t.renderRuns(pdf, runs, fontScale, y)

	if degree := strings.TrimSpace(edu.Degree); degree != "" {
		y = t.renderEntryTitle(pdf, tr(degree), fontScale, y)
	}

	return y + t.scale(2.6, fontScale)
}

func (t *BennettTemplate) renderExperience(pdf *fpdf.Fpdf, exp resume.Experience, tr func(string) string, fontScale, y float64) float64 {
	runs := make([]bennettRun, 0, 6)
	if company := strings.TrimSpace(exp.Company); company != "" {
		runs = append(runs, bennettPrimaryRun(tr(company), exp.Link))
	}
	if location := strings.TrimSpace(exp.Location); location != "" {
		runs = append(runs, bennettMutedRuns(tr(location))...)
	}
	if dates := bennettDateRange(exp.Start, exp.End); dates != "" {
		runs = append(runs, bennettMutedRuns(tr(dates))...)
	}
	y = t.renderRuns(pdf, runs, fontScale, y)

	if role := strings.TrimSpace(exp.Role); role != "" {
		y = t.renderEntryTitle(pdf, tr(role), fontScale, y)
	}
	for _, bullet := range exp.Bullets {
		if strings.TrimSpace(bullet) == "" {
			continue
		}
		y = t.renderBullet(pdf, bullet, tr, fontScale, y)
	}

	return y + t.scale(2.6, fontScale)
}

func (t *BennettTemplate) renderProject(pdf *fpdf.Fpdf, proj resume.Project, tr func(string) string, fontScale, y float64) float64 {
	runs := make([]bennettRun, 0, 6)
	if name := strings.TrimSpace(proj.Name); name != "" {
		runs = append(runs, bennettPrimaryRun(tr(name), proj.Link))
	}
	if tech := strings.TrimSpace(proj.Tech); tech != "" {
		runs = append(runs, bennettMutedRuns(tr(tech))...)
	}
	if date := strings.TrimSpace(proj.Date); date != "" {
		runs = append(runs, bennettMutedRuns(tr(date))...)
	}
	y = t.renderRuns(pdf, runs, fontScale, y)

	for _, bullet := range proj.Bullets {
		if strings.TrimSpace(bullet) == "" {
			continue
		}
		y = t.renderBullet(pdf, bullet, tr, fontScale, y)
	}

	return y + t.scale(2.6, fontScale)
}

// renderEntryTitle draws a bold ink line (degree, role).
func (t *BennettTemplate) renderEntryTitle(pdf *fpdf.Fpdf, title string, fontScale, y float64) float64 {
	size := t.scale(bennettTitlePT, fontScale)
	h := t.scale(ptMM(bennettTitlePT)*1.3, fontScale)

	pdf.SetFont("Helvetica", "B", size)
	pdf.SetTextColor(bennettInkR, bennettInkG, bennettInkB)
	pdf.SetXY(bennettMargin, y)
	pdf.CellFormat(bennettContentWidth, h, title, "", 1, "L", false, 0, "")

	return y + h
}

func (t *BennettTemplate) renderBullet(pdf *fpdf.Fpdf, text string, tr func(string) string, fontScale, y float64) float64 {
	size := t.scale(bennettBodyPT, fontScale)
	lineH := t.scale(ptMM(bennettBodyPT)*1.45, fontScale)
	indent := t.scale(bennettBulletIndentMM, fontScale)

	pdf.SetFont("Helvetica", "", size)
	pdf.SetTextColor(bennettBodyR, bennettBodyG, bennettBodyB)

	body := tr(strings.TrimSpace(text))
	w := bennettContentWidth - indent
	lines := pdf.SplitLines([]byte(body), w)

	pdf.SetXY(bennettMargin, y)
	pdf.CellFormat(indent, lineH, tr("\u2022"), "", 0, "L", false, 0, "")
	pdf.SetXY(bennettMargin+indent, y)
	pdf.MultiCell(w, lineH, body, "", "L", false)

	return y + float64(len(lines))*lineH + t.scale(0.7, fontScale)
}

// renderSkills lays skill groups out as a bullet grid: one column for a single
// group, two up to four, three beyond that.
func (t *BennettTemplate) renderSkills(pdf *fpdf.Fpdf, skills []resume.SkillGroup, tr func(string) string, fontScale, y float64) float64 {
	items := make([]string, 0, len(skills))
	for _, group := range skills {
		category := strings.TrimSpace(group.Category)
		values := strings.TrimSpace(group.Values)
		switch {
		case category != "" && values != "":
			items = append(items, tr(category)+": "+tr(values))
		case category != "":
			items = append(items, tr(category))
		case values != "":
			items = append(items, tr(values))
		}
	}
	if len(items) == 0 {
		return y
	}

	cols := 2
	if len(items) == 1 {
		cols = 1
	} else if len(items) > 4 {
		cols = 3
	}

	gap := t.scale(8.0, fontScale)
	colW := (bennettContentWidth - float64(cols-1)*gap) / float64(cols)
	indent := t.scale(bennettBulletIndentMM, fontScale)
	size := t.scale(bennettBodyPT, fontScale)
	lineH := t.scale(ptMM(bennettBodyPT)*1.45, fontScale)

	pdf.SetFont("Helvetica", "", size)
	pdf.SetTextColor(bennettBodyR, bennettBodyG, bennettBodyB)

	rows := (len(items) + cols - 1) / cols
	for r := 0; r < rows; r++ {
		rowTop := y
		rowH := 0.0
		for c := 0; c < cols; c++ {
			i := r*cols + c
			if i >= len(items) {
				break
			}
			x := bennettMargin + float64(c)*(colW+gap)
			w := colW - indent
			lines := pdf.SplitLines([]byte(items[i]), w)
			if h := float64(len(lines)) * lineH; h > rowH {
				rowH = h
			}
			pdf.SetXY(x, rowTop)
			pdf.CellFormat(indent, lineH, tr("\u2022"), "", 0, "L", false, 0, "")
			pdf.SetXY(x+indent, rowTop)
			pdf.MultiCell(w, lineH, items[i], "", "L", false)
		}
		y = rowTop + rowH + t.scale(1.4, fontScale)
	}

	return y
}

// renderRuns draws styled runs left-aligned on a single line.
func (t *BennettTemplate) renderRuns(pdf *fpdf.Fpdf, runs []bennettRun, fontScale, y float64) float64 {
	size := t.scale(bennettMetaPT, fontScale)
	h := t.scale(ptMM(bennettMetaPT)*1.4, fontScale)

	x := bennettMargin
	for _, run := range runs {
		if run.text == "" {
			continue
		}
		style := ""
		if run.bold {
			style = "B"
		}
		pdf.SetFont("Helvetica", style, size)
		pdf.SetTextColor(run.r, run.g, run.b)
		w := pdf.GetStringWidth(run.text)
		pdf.SetXY(x, y)
		pdf.CellFormat(w, h, run.text, "", 0, "L", false, 0, run.link)
		x += w
	}

	return y + h
}

// bennettRun is one styled fragment of a single-line entry header.
type bennettRun struct {
	text string
	bold bool
	r    int
	g    int
	b    int
	link string
}

func bennettPrimaryRun(text, link string) bennettRun {
	return bennettRun{text: text, r: bennettMetaR, g: bennettMetaG, b: bennettMetaB, link: link}
}

// bennettMutedRuns returns the separator plus a muted fragment, so callers can
// append fragments without tracking separators themselves.
func bennettMutedRuns(text string) []bennettRun {
	muted := bennettRun{r: bennettMutedR, g: bennettMutedG, b: bennettMutedB}
	sep := muted
	sep.text = " | "
	frag := muted
	frag.text = text
	return []bennettRun{sep, frag}
}

// bennettDateRange formats an entry's dates, e.g. "2033 - 2035".
func bennettDateRange(start, end string) string {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	switch {
	case start != "" && end != "" && start != end:
		return start + " - " + end
	case end != "":
		return end
	default:
		return start
	}
}

func bennettRule(pdf *fpdf.Fpdf, y, thickness float64) {
	pdf.SetDrawColor(bennettRuleR, bennettRuleG, bennettRuleB)
	pdf.SetLineWidth(thickness)
	pdf.Line(bennettMargin, y, bennettPageWidth-bennettMargin, y)
}

// bennettTrackedWidth measures text drawn with extra letter tracking.
func bennettTrackedWidth(pdf *fpdf.Fpdf, text string, tracking float64) float64 {
	width := 0.0
	for _, r := range text {
		width += pdf.GetStringWidth(string(r)) + tracking
	}
	if width > 0 {
		width -= tracking
	}
	return width
}

// bennettDrawTracked draws text one glyph at a time at an absolute baseline.
func bennettDrawTracked(pdf *fpdf.Fpdf, x, baselineY float64, text string, tracking float64) {
	for _, r := range text {
		ch := string(r)
		pdf.Text(x, baselineY, ch)
		x += pdf.GetStringWidth(ch) + tracking
	}
}
