package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
	"github.com/pquerna/otp/totp"

	tea "github.com/charmbracelet/bubbletea"

	"passbook/internal/platform"
)

// fieldRow is one row of the entry-detail grid: a label, its value, and an
// optional dim hint of the keys that act on it (e.g. "[c] copy").
type fieldRow struct {
	label string
	value string
	hint  string
}

// renderFieldRows prints rows as an aligned two-column grid, padding labels
// to the widest label so values line up in a single column.
func renderFieldRows(b *strings.Builder, rows []fieldRow) {
	maxLabel := 0
	for _, r := range rows {
		if w := lipgloss.Width(r.label); w > maxLabel {
			maxLabel = w
		}
	}
	for _, r := range rows {
		b.WriteString(labelStyle.Render(padRight(r.label, maxLabel)))
		b.WriteString("  ")
		b.WriteString(r.value)
		if r.hint != "" {
			b.WriteString(" ")
			b.WriteString(dimStyle.Render(r.hint))
		}
		b.WriteString("\n")
	}
}

func padRight(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

func renderEntryView(m Model) string {
	ent := m.main.currentEnt
	if ent == nil {
		return ""
	}

	var b strings.Builder
	icon := entryTypeIcon(ent.Type)
	b.WriteString(titleStyle.Render(" " + icon + " " + ent.Title + " "))
	b.WriteString("\n\n")

	rows := []fieldRow{{label: "Title:", value: ent.Title}}
	var totpBar string
	switch EntryType(ent.Type) {
	case TypeLogin:
		var extra []fieldRow
		extra, totpBar = loginViewRows(m)
		rows = append(rows, extra...)
	case TypeCard:
		rows = append(rows, cardViewRows(m)...)
	}
	renderFieldRows(&b, rows)
	if totpBar != "" {
		b.WriteString(totpBar)
		b.WriteString("\n")
	}

	if len(ent.Attachments) > 0 {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Attachments:"))
		b.WriteString("\n")
		for i, att := range ent.Attachments {
			if i < 9 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("[%d] ", i+1)))
			}
			b.WriteString(linkStyle.Render("➤ " + att.FileName))
			b.WriteString(dimStyle.Render(fmt.Sprintf(" (%s)", formatBytes(att.Size))))
			b.WriteString("\n")
		}
		b.WriteString(dimStyle.Render("press number to download to ~/Downloads"))
		b.WriteString("\n")
	}

	if strings.TrimSpace(ent.CustomText) != "" {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Notes:"))
		b.WriteString("\n")
		b.WriteString(ent.CustomText)
		b.WriteString("\n")
	}

	if m.main.viewStatus != "" {
		b.WriteString("\n")
		b.WriteString(m.main.viewStatus)
	}

	return b.String()
}

// loginViewRows returns the grid rows for a login entry plus the TOTP
// progress bar, which is rendered on its own line below the grid.
func loginViewRows(m Model) (rows []fieldRow, totpBar string) {
	ent := m.main.currentEnt
	if ent.Username != "" {
		rows = append(rows, fieldRow{label: "Username:", value: ent.Username, hint: "[u] copy"})
	}

	if ent.Password != "" {
		pass := strings.Repeat("*", len(ent.Password))
		if m.main.showSensitive {
			pass = ent.Password
		}
		rows = append(rows, fieldRow{label: "Password:", value: pass, hint: "[v] reveal [c] copy [h] history"})
	}

	if strings.TrimSpace(ent.Link) != "" {
		rows = append(rows, fieldRow{label: "Link:", value: linkStyle.Render(ent.Link), hint: "[o] open [l] copy"})
	}

	secret := strings.ReplaceAll(ent.TotpSecret, " ", "")
	if secret != "" {
		rows = append(rows, fieldRow{label: "TOTP:", value: m.main.totpCode, hint: "[t] copy"})
		totpBar = m.main.totpBar
	}
	return rows, totpBar
}

func cardViewRows(m Model) []fieldRow {
	ent := m.main.currentEnt
	num := ent.CardNumber
	if !m.main.showSensitive && len(num) > 4 {
		num = "**** **** **** " + num[len(num)-4:]
	}
	cvv := "***"
	if m.main.showSensitive {
		cvv = ent.CVV
	}
	return []fieldRow{
		{label: "Number:", value: num, hint: "[v] reveal [c] copy"},
		{label: "Expiry:", value: ent.Expiry},
		{label: "CVV:", value: cvv},
	}
}

func formatTOTPDisplay(secret string) (code, bar string) {
	codeVal, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		return "", ""
	}
	sec := time.Now().Unix() % 30
	remain := 30 - sec
	barStr := renderProgressBar(float64(remain)/30.0, 24)

	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	if remain <= 5 {
		return codeVal, errorStyle.Render(fmt.Sprintf("%02ds %s", remain, barStr))
	}
	if remain <= 10 {
		return codeVal, yellowStyle.Render(fmt.Sprintf("%02ds %s", remain, barStr))
	}
	return codeVal, successStyle.Render(fmt.Sprintf("%02ds %s", remain, barStr))
}

// renderProgressBar draws a smooth progress bar using eighth-block
// characters for sub-cell precision instead of chunky whole-block steps.
func renderProgressBar(fraction float64, width int) string {
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	eighths := []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}
	totalEighths := int(fraction*float64(width)*8 + 0.5)
	filled := totalEighths / 8
	remainder := totalEighths % 8

	var b strings.Builder
	b.WriteString(strings.Repeat("█", filled))
	if filled < width && remainder > 0 {
		b.WriteRune(eighths[remainder])
		filled++
	}
	b.WriteString(strings.Repeat("░", width-filled))
	return b.String()
}

// handleViewAction processes per-action copy/view keys on the entry detail
// pane. It returns whether the key was handled and an optional tea.Cmd
// (e.g. to schedule clearing the clipboard of a sensitive value).
func (m *Model) handleViewAction(key string) (bool, tea.Cmd) {
	if m.main.currentEnt == nil {
		return false, nil
	}
	ent := m.main.currentEnt
	switch key {
	case "u":
		if EntryType(ent.Type) == TypeLogin && ent.Username != "" {
			_ = clipboard.WriteAll(ent.Username)
			m.notifyCopied("Username")
		}
		return true, nil
	case "c":
		switch EntryType(ent.Type) {
		case TypeLogin:
			if ent.Password != "" {
				return true, m.copySensitiveWithClear(ent.Password, "Password")
			}
		case TypeCard:
			if ent.CardNumber != "" {
				return true, m.copySensitiveWithClear(ent.CardNumber, "Card")
			}
		}
		return true, nil
	case "l":
		if EntryType(ent.Type) == TypeLogin && strings.TrimSpace(ent.Link) != "" {
			_ = clipboard.WriteAll(ent.Link)
			m.notifyCopied("Link")
		}
		return true, nil
	case "t":
		if EntryType(ent.Type) == TypeLogin {
			secret := strings.ReplaceAll(ent.TotpSecret, " ", "")
			if secret != "" {
				code, err := totp.GenerateCode(secret, time.Now())
				if err == nil {
					return true, m.copySensitiveWithClear(code, "TOTP")
				}
			}
		}
		return true, nil
	case "v":
		m.main.showSensitive = !m.main.showSensitive
		if m.main.showSensitive {
			m.main.showSensitiveClearAt = time.Now().Add(5 * time.Second)
		} else {
			m.main.showSensitiveClearAt = time.Time{}
		}
		return true, nil
	case "h":
		if EntryType(ent.Type) == TypeLogin && len(ent.History) > 0 {
			m.overlay = overlayHistory
			m.modals = newHistoryModal(ent.History)
		}
		return true, nil
	case "o":
		if ent.Link != "" {
			_ = platform.OpenURL(ent.Link)
		}
		return true, nil
	}
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		idx := int(key[0] - '1')
		if idx < len(ent.Attachments) {
			m.downloadAttachment(ent.Attachments[idx])
		}
		return true, nil
	}
	return false, nil
}

// downloadAttachment writes a stored attachment's bytes to ~/Downloads,
// avoiding overwriting any existing file with the same name.
func (m *Model) downloadAttachment(att Attachment) {
	data, err := m.store.ReadAttachment(att.ID)
	if err != nil {
		m.overlay = overlayError
		m.modals = newErrorModal("Failed to read attachment: " + err.Error())
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		m.overlay = overlayError
		m.modals = newErrorModal("Failed to locate home directory: " + err.Error())
		return
	}
	dir := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(dir, 0700); err != nil {
		m.overlay = overlayError
		m.modals = newErrorModal("Failed to access Downloads folder: " + err.Error())
		return
	}
	dest := uniqueDownloadPath(dir, att.FileName)
	if err := os.WriteFile(dest, data, 0600); err != nil {
		m.overlay = overlayError
		m.modals = newErrorModal("Failed to save file: " + err.Error())
		return
	}
	m.main.viewStatus = successStyle.Render("✓ Saved to " + dest)
	m.main.viewStatusClearAt = time.Now().Add(3 * time.Second)
}

// uniqueDownloadPath returns a path in dir for fileName, appending " (n)"
// before the extension if a file with that name already exists.
func uniqueDownloadPath(dir, fileName string) string {
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	candidate := filepath.Join(dir, fileName)
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
	}
}

// copySensitiveWithClear copies a sensitive value to the clipboard, shows a
// transient status message, and schedules a clipboardClearMsg so the
// clipboard is wiped after 30s if it still holds the copied value.
func (m *Model) copySensitiveWithClear(text, item string) tea.Cmd {
	_ = clipboard.WriteAll(text)
	m.main.viewStatus = successStyle.Render(fmt.Sprintf("✓ %s copied (clipboard clears in 30s)", item))
	m.main.viewStatusClearAt = time.Now().Add(3 * time.Second)
	return tea.Tick(30*time.Second, func(time.Time) tea.Msg { return clipboardClearMsg{text} })
}

func deleteEntry(m *Model) {
	if m.main.currentEntryID != 0 {
		err := m.store.DeleteEntry(m.main.currentEntryID)
		if err != nil {
			return
		}
		m.main.currentEntryID = 0
		m.main.currentEnt = nil
		m.main.showContent = false
		m.main.refreshTree(m.store, m.main.search.Value())
	}
}
