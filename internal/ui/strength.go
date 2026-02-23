package ui

import (
	"fmt"
	"strings"

	"passbook/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const strengthBarWidth = 15

// strengthMeter keeps references to text views that display a password
// strength bar.  It can drive both an inline form text-view (added via
// Form.AddTextView) and a standalone tview.TextView (used in Flex rows).
type strengthMeter struct {
	views []*tview.TextView
}

func newStrengthMeter() *strengthMeter {
	return &strengthMeter{}
}

// AddTo inserts a 1-row text-view into a tview.Form right at the current
// position.  The label is a single space so it does not widen the form's
// label column.
func (m *strengthMeter) AddTo(form *tview.Form) {
	tv := tview.NewTextView().SetDynamicColors(true)
	tv.SetLabel(" ")
	tv.SetSize(1, 0)
	tv.SetScrollable(false)
	form.AddFormItem(tv)
	m.views = append(m.views, tv)
}

// NewTextView creates a standalone TextView (for Flex rows like the
// password generator).
func (m *strengthMeter) NewTextView() *tview.TextView {
	tv := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	m.views = append(m.views, tv)
	return tv
}

// Update evaluates the password and refreshes every attached view.
func (m *strengthMeter) Update(password string) {
	bar := formatStrengthBar(password)
	for _, tv := range m.views {
		tv.SetText(bar)
	}
}

// formatStrengthBar builds the colored bar string for a given password.
func formatStrengthBar(password string) string {
	if password == "" {
		return ""
	}

	score, level, label := utils.PasswordStrength(password)

	var color string
	switch level {
	case utils.StrengthWeak:
		color = "red"
	case utils.StrengthFair:
		color = "yellow"
	case utils.StrengthGood:
		color = "blue"
	case utils.StrengthStrong:
		color = "green"
	default:
		return ""
	}

	filled := score * strengthBarWidth / 100
	if filled < 1 && score > 0 {
		filled = 1
	}
	empty := strengthBarWidth - filled

	return fmt.Sprintf("[%s]%s[gray]%s[-]  [%s]%s[-]",
		color, strings.Repeat("━", filled),
		strings.Repeat("━", empty),
		color, label,
	)
}

// makeStrengthDisplayRow creates a standalone Flex row with a label and the
// meter's TextView (used in the password generator).
func makeStrengthDisplayRow(meter *strengthMeter) *tview.Flex {
	tv := meter.NewTextView()
	f := tview.NewFlex().SetDirection(tview.FlexColumn)
	lbl := tview.NewTextView().SetText("Strength:").SetTextColor(tcell.ColorDimGray)
	f.AddItem(lbl, 12, 0, false)
	f.AddItem(tv, 0, 1, false)
	return f
}
