package template

import (
	"math"

	"github.com/LordFarquaadtheCreator/resume-builder/internal/resume"
	"github.com/go-pdf/fpdf"
)

// Contact icons are drawn as vector shapes so the template needs no icon font.
// Each helper draws into a square box of side s whose left edge is x, centered
// vertically on cy.

// contactItem is one icon + label pair in the header contact strip.
type contactItem struct {
	icon func(pdf *fpdf.Fpdf, x, cy, s float64)
	text string
}

// bennettContactItems builds the header strip: phone, email, location, then
// any remaining profile links in a stable order.
func bennettContactItems(data resume.ResumeData, tr func(string) string) []contactItem {
	items := make([]contactItem, 0, 6)
	if phone := data.Contact.Links["phone"]; phone != "" {
		items = append(items, contactItem{icon: drawPhoneIcon, text: tr(phone)})
	}
	if email := data.Contact.Email; email != "" {
		items = append(items, contactItem{icon: drawMailIcon, text: tr(email)})
	}
	if location := data.Contact.Location; location != "" {
		items = append(items, contactItem{icon: drawPinIcon, text: tr(location)})
	}
	for _, key := range []string{"linkedin", "github", "website"} {
		if link := data.Contact.Links[key]; link != "" {
			items = append(items, contactItem{icon: drawGlobeIcon, text: tr(link)})
		}
	}
	return items
}

// renderContactRow lays items across the content width with space between
// them, packing into extra rows when a single row would be too tight.
func (t *BennettTemplate) renderContactRow(pdf *fpdf.Fpdf, items []contactItem, fontScale, y float64) float64 {
	size := t.scale(bennettContactPT, fontScale)
	iconSize := t.scale(bennettIconMM, fontScale)
	iconGap := t.scale(1.2, fontScale)
	rowH := t.scale(ptMM(bennettContactPT)*1.6, fontScale)
	minGap := t.scale(6.0, fontScale)

	pdf.SetFont("Helvetica", "", size)
	widths := make([]float64, len(items))
	for i, item := range items {
		widths[i] = iconSize + iconGap + pdf.GetStringWidth(item.text)
	}

	// Greedy packing: each row must fit with at least minGap between items.
	var rows [][]int
	var row []int
	rowW := 0.0
	for i, w := range widths {
		needed := w
		if len(row) > 0 {
			needed += minGap
		}
		if len(row) > 0 && rowW+needed > bennettContentWidth {
			rows = append(rows, row)
			row = nil
			rowW = 0
			needed = w
		}
		row = append(row, i)
		rowW += needed
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}

	for _, r := range rows {
		sum := 0.0
		for _, i := range r {
			sum += widths[i]
		}
		if len(r) == 1 {
			x := bennettMargin + (bennettContentWidth-widths[r[0]])/2
			t.drawContactItem(pdf, items[r[0]], widths[r[0]], x, y, rowH, iconSize, iconGap, size)
		} else {
			gap := (bennettContentWidth - sum) / float64(len(r)-1)
			x := bennettMargin
			for _, i := range r {
				t.drawContactItem(pdf, items[i], widths[i], x, y, rowH, iconSize, iconGap, size)
				x += widths[i] + gap
			}
		}
		y += rowH + t.scale(0.8, fontScale)
	}

	return y
}

func (t *BennettTemplate) drawContactItem(pdf *fpdf.Fpdf, item contactItem, w, x, y, rowH, iconSize, iconGap, textSize float64) {
	item.icon(pdf, x, y+rowH/2, iconSize)

	pdf.SetFont("Helvetica", "", textSize)
	pdf.SetTextColor(bennettMetaR, bennettMetaG, bennettMetaB)
	pdf.SetXY(x+iconSize+iconGap, y)
	pdf.CellFormat(w-iconSize-iconGap, rowH, item.text, "", 1, "L", false, 0, "")
}

// drawMailIcon draws a filled envelope with a light flap.
func drawMailIcon(pdf *fpdf.Fpdf, x, cy, s float64) {
	top := cy - s*0.32
	h := s * 0.64

	pdf.SetFillColor(bennettMetaR, bennettMetaG, bennettMetaB)
	pdf.RoundedRect(x, top, s, h, s*0.12, "1234", "F")

	pdf.SetDrawColor(252, 252, 252)
	pdf.SetLineWidth(s * 0.10)
	pdf.SetLineCapStyle("R")
	pdf.MoveTo(x+s*0.16, top+s*0.18)
	pdf.LineTo(x+s*0.5, top+h*0.62)
	pdf.LineTo(x+s*0.84, top+s*0.18)
	pdf.DrawPath("D")
	pdf.SetLineCapStyle("B")
}

// drawPinIcon draws a map pin: filled head with a stem and a punched hole.
func drawPinIcon(pdf *fpdf.Fpdf, x, cy, s float64) {
	cx := x + s*0.5
	headCy := cy - s*0.14

	pdf.SetFillColor(bennettMetaR, bennettMetaG, bennettMetaB)
	pdf.Circle(cx, headCy, s*0.30, "F")
	pdf.Polygon([]fpdf.PointType{
		{X: cx - s*0.21, Y: headCy + s*0.21},
		{X: cx + s*0.21, Y: headCy + s*0.21},
		{X: cx, Y: cy + s*0.48},
	}, "F")

	pdf.SetFillColor(252, 252, 252)
	pdf.Circle(cx, headCy, s*0.11, "F")
}

// drawGlobeIcon draws a globe: circle, meridian, equator.
func drawGlobeIcon(pdf *fpdf.Fpdf, x, cy, s float64) {
	cx := x + s*0.5
	r := s * 0.42

	pdf.SetDrawColor(bennettMetaR, bennettMetaG, bennettMetaB)
	pdf.SetLineWidth(s * 0.09)
	pdf.Circle(cx, cy, r, "D")
	pdf.Ellipse(cx, cy, r*0.45, r, 0, "D")
	pdf.Line(cx-r, cy, cx+r, cy)
}

// drawPhoneIcon draws a handset: an arc with round earpieces at both ends.
func drawPhoneIcon(pdf *fpdf.Fpdf, x, cy, s float64) {
	cx := x + s*0.5
	radius := s * 0.30
	pad := s * 0.16

	pdf.SetDrawColor(bennettMetaR, bennettMetaG, bennettMetaB)
	pdf.SetFillColor(bennettMetaR, bennettMetaG, bennettMetaB)
	pdf.SetLineWidth(s * 0.20)
	pdf.SetLineCapStyle("R")
	// Earpiece up-right, mouthpiece down-left, handle bowing to the lower-right.
	pdf.Arc(cx, cy, radius, radius, 0, 225, 405, "D")
	pdf.SetLineCapStyle("B")

	for _, deg := range []float64{225, 405} {
		rad := deg * math.Pi / 180
		pdf.Circle(cx+radius*math.Cos(rad), cy-radius*math.Sin(rad), pad, "F")
	}
}
