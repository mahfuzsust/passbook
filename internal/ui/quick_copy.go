package ui

import (
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/pquerna/otp/totp"
	tea "github.com/charmbracelet/bubbletea"
)

type quickCopyItem struct {
	label  string
	key    string
	action func(*Model) tea.Cmd
}

type quickCopyModel struct {
	items  []quickCopyItem
	cursor int
}

func (m *Model) showQuickCopy() {
	if m.main.currentEnt == nil || m.main.currentEntryID == 0 {
		return
	}
	qc := quickCopyModel{}
	ent := m.main.currentEnt

	switch EntryType(ent.Type) {
	case TypeLogin:
		if ent.Username != "" {
			qc.items = append(qc.items, quickCopyItem{"Username", "u", func(m *Model) tea.Cmd {
				_ = clipboard.WriteAll(ent.Username)
				m.notifyCopied("Username")
				return nil
			}})
		}
		if ent.Password != "" {
			qc.items = append(qc.items, quickCopyItem{"Password", "c", func(m *Model) tea.Cmd {
				return m.copySensitiveWithClear(ent.Password, "Password")
			}})
		}
		if strings.TrimSpace(ent.Link) != "" {
			qc.items = append(qc.items, quickCopyItem{"Link", "l", func(m *Model) tea.Cmd {
				_ = clipboard.WriteAll(ent.Link)
				m.notifyCopied("Link")
				return nil
			}})
		}
		secret := strings.ReplaceAll(ent.TotpSecret, " ", "")
		if secret != "" {
			qc.items = append(qc.items, quickCopyItem{"TOTP Code", "t", func(m *Model) tea.Cmd {
				code, err := totp.GenerateCode(secret, time.Now())
				if err != nil {
					return nil
				}
				return m.copySensitiveWithClear(code, "TOTP")
			}})
		}
	case TypeCard:
		if ent.CardNumber != "" {
			qc.items = append(qc.items, quickCopyItem{"Card Number", "c", func(m *Model) tea.Cmd {
				return m.copySensitiveWithClear(ent.CardNumber, "Card Number")
			}})
		}
		if ent.CVV != "" {
			qc.items = append(qc.items, quickCopyItem{"CVV", "v", func(m *Model) tea.Cmd {
				return m.copySensitiveWithClear(ent.CVV, "CVV")
			}})
		}
	case TypeNote:
		if strings.TrimSpace(ent.CustomText) != "" {
			qc.items = append(qc.items, quickCopyItem{"Note", "n", func(m *Model) tea.Cmd {
				_ = clipboard.WriteAll(ent.CustomText)
				m.notifyCopied("Note")
				return nil
			}})
		}
	}

	if len(qc.items) == 0 {
		return
	}
	m.quickCopy = qc
	m.overlay = overlayQuickCopy
}

func (m *Model) updateQuickCopy(key string) (Model, tea.Cmd) {
	switch key {
	case "esc":
		m.overlay = overlayNone
	case "up", "k":
		if m.quickCopy.cursor > 0 {
			m.quickCopy.cursor--
		}
	case "down", "j":
		if m.quickCopy.cursor < len(m.quickCopy.items)-1 {
			m.quickCopy.cursor++
		}
	case "enter":
		if len(m.quickCopy.items) > 0 {
			cmd := m.quickCopy.items[m.quickCopy.cursor].action(m)
			m.overlay = overlayNone
			return *m, cmd
		}
	default:
		for _, item := range m.quickCopy.items {
			if key == item.key {
				cmd := item.action(m)
				m.overlay = overlayNone
				return *m, cmd
			}
		}
	}
	return *m, nil
}

func (m Model) viewQuickCopy() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Quick Copy "))
	b.WriteString("\n\n")
	for i, item := range m.quickCopy.items {
		line := item.label + " (" + item.key + ")"
		if i == m.quickCopy.cursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return centerModal(b.String(), m.width, m.height, 35, 8, 50, 14, 0.35, 0.35)
}
