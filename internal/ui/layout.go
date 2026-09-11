package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func centerModal(content string, termW, termH int, minW, minH, maxW, maxH int, wPct, hPct float64) string {
	modalW := int(float64(termW) * wPct)
	if modalW < minW {
		modalW = minW
	}
	if maxW > 0 && modalW > maxW {
		modalW = maxW
	}
	if modalW > termW {
		modalW = termW
	}

	modalH := int(float64(termH) * hPct)
	if modalH < minH {
		modalH = minH
	}
	if maxH > 0 && modalH > maxH {
		modalH = maxH
	}
	if modalH > termH {
		modalH = termH
	}

	box := borderStyle.Width(modalW - 2).Height(modalH - 2).Render(content)
	return lipgloss.Place(termW, termH, lipgloss.Center, lipgloss.Center, box)
}

func splitView(left, right string, termW, termH int, leftRatio float64, minLeft, minRight int) string {
	leftW := int(float64(termW) * leftRatio)
	if leftW < minLeft {
		leftW = minLeft
	}
	if termW-leftW < minRight {
		leftW = termW - minRight
	}
	if leftW < 0 {
		leftW = 0
	}
	rightW := termW - leftW
	if rightW < 0 {
		rightW = 0
	}

	leftBox := borderStyle.Width(leftW - 2).Height(termH - 2).Render(left)
	rightBox := borderStyle.Width(rightW - 2).Height(termH - 2).Render(right)

	leftLines := strings.Split(leftBox, "\n")
	rightLines := strings.Split(rightBox, "\n")

	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	var out strings.Builder
	for i := 0; i < maxLines; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		lPad := leftW - lipgloss.Width(l)
		if lPad < 0 {
			lPad = 0
		}
		out.WriteString(l)
		out.WriteString(strings.Repeat(" ", lPad))
		out.WriteString(r)
		out.WriteString("\n")
	}
	return strings.TrimRight(out.String(), "\n")
}

func padHeight(content string, height int) string {
	lines := strings.Split(content, "\n")
	for len(lines) < height {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
