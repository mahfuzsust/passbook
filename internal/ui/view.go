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

func renderEntryView(m Model) string {
	ent := m.main.currentEnt
	if ent == nil {
		return ""
	}

	var b strings.Builder
	icon := entryTypeIcon(ent.Type)
	b.WriteString(titleStyle.Render(" " + icon + " " + ent.Title + " "))
	b.WriteString("\n\n")

	b.WriteString(labelStyle.Render("Title:"))
	b.WriteString(" ")
	b.WriteString(ent.Title)
	b.WriteString("\n\n")

	switch EntryType(ent.Type) {
	case TypeLogin:
		renderLoginViewContent(&b, m)
	case TypeCard:
		renderCardViewContent(&b, m)
	}

	if len(ent.Attachments) > 0 {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("Attachments:"))
		b.WriteString("\n")
		for i, att := range ent.Attachments {
			if i < 9 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("[%d] ", i+1)))
			}
			b.WriteString(linkStyle.Render("➤ "+att.FileName))
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

func renderLoginViewContent(b *strings.Builder, m Model) {
	ent := m.main.currentEnt
	if ent.Username != "" {
		b.WriteString(labelStyle.Render("Username:"))
		b.WriteString(" ")
		b.WriteString(ent.Username)
		b.WriteString(dimStyle.Render(" [u] copy"))
		b.WriteString("\n")
	}

	if ent.Password != "" {
		pass := strings.Repeat("*", len(ent.Password))
		if m.main.showSensitive {
			pass = ent.Password
		}
		b.WriteString(labelStyle.Render("Password:"))
		b.WriteString(" ")
		b.WriteString(pass)
		b.WriteString(dimStyle.Render(" [v] reveal [c] copy [h] history"))
		b.WriteString("\n")
	}

	if strings.TrimSpace(ent.Link) != "" {
		b.WriteString(labelStyle.Render("Link:"))
		b.WriteString(" ")
		b.WriteString(linkStyle.Render(ent.Link))
		b.WriteString(dimStyle.Render(" [o] open [l] copy"))
		b.WriteString("\n")
	}

	secret := strings.ReplaceAll(ent.TotpSecret, " ", "")
	if secret != "" {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("TOTP:"))
		b.WriteString(" ")
		b.WriteString(m.main.totpCode)
		b.WriteString(dimStyle.Render(" [t] copy"))
		b.WriteString("\n")
		b.WriteString(m.main.totpBar)
		b.WriteString("\n")
	}
}

func renderCardViewContent(b *strings.Builder, m Model) {
	ent := m.main.currentEnt
	num := ent.CardNumber
	if !m.main.showSensitive && len(num) > 4 {
		num = "**** **** **** " + num[len(num)-4:]
	}
	b.WriteString(labelStyle.Render("Number:"))
	b.WriteString(" ")
	b.WriteString(num)
	b.WriteString(dimStyle.Render(" [v] reveal [c] copy"))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Expiry:"))
	b.WriteString(" ")
	b.WriteString(ent.Expiry)
	b.WriteString("\n")

	cvv := "***"
	if m.main.showSensitive {
		cvv = ent.CVV
	}
	b.WriteString(labelStyle.Render("CVV:"))
	b.WriteString(" ")
	b.WriteString(cvv)
	b.WriteString("\n")
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
		idx := int(key[0]-'1')
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
