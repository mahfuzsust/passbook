package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorUnfocusedBg = lipgloss.Color("236")
	colorFocusedBg   = lipgloss.Color("24")

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("117")).
		Padding(0, 1)

	borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("117"))

	focusedBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39"))

	labelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("11"))

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9"))

	successStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("10"))

	skyStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("117"))

	linkStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Underline(true)

	selectedStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("39")).
		Foreground(lipgloss.Color("0"))

	buttonStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(colorUnfocusedBg).
		Padding(0, 1)

	buttonFocusedStyle = lipgloss.NewStyle().
		Foreground(colorFocusedBg).
		Background(lipgloss.Color("15")).
		Padding(0, 1)

	buttonDangerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("167")).
		Background(colorUnfocusedBg).
		Padding(0, 1)

	buttonDangerFocusedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("167")).
		Background(lipgloss.Color("15")).
		Padding(0, 1)

	qrModuleStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("0")).
		Foreground(lipgloss.Color("0"))
)

func renderButton(label string, focused bool, danger bool) string {
	if danger {
		if focused {
			return buttonDangerFocusedStyle.Render(label)
		}
		return buttonDangerStyle.Render(label)
	}
	if focused {
		return buttonFocusedStyle.Render(label)
	}
	return buttonStyle.Render(label)
}
