package ui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type editorFocusTarget int

const (
	eftTitle      editorFocusTarget = 0
	eftFolder     editorFocusTarget = 1
	eftUsername   editorFocusTarget = 2
	eftPassword   editorFocusTarget = 3
	eftLink       editorFocusTarget = 4
	eftTotpSecret editorFocusTarget = 5
	eftCardNumber editorFocusTarget = 6
	eftExpiry     editorFocusTarget = 7
	eftCVV        editorFocusTarget = 8
	eftNotes      editorFocusTarget = 9
	eftButtons    editorFocusTarget = 10
)

func (e *editorModel) buildFocusOrder() []editorFocusTarget {
	order := []editorFocusTarget{eftTitle, eftFolder}
	switch e.entryType {
	case TypeLogin:
		order = append(order, eftUsername, eftPassword, eftLink, eftTotpSecret)
	case TypeCard:
		order = append(order, eftCardNumber, eftExpiry, eftCVV)
	}
	order = append(order, eftNotes, eftButtons)
	return order
}

func (e *editorModel) initFocus() tea.Cmd {
	e.focusOrder = e.buildFocusOrder()
	e.focusPos = 0
	return e.applyFocus()
}

func (e *editorModel) blurAll() {
	e.title.Blur()
	e.username.Blur()
	e.password.Blur()
	e.link.Blur()
	e.totpSecret.Blur()
	e.cardNumber.Blur()
	e.expiry.Blur()
	e.cvv.Blur()
	e.notes.Blur()
}

func (e *editorModel) applyFocus() tea.Cmd {
	e.blurAll()
	if len(e.focusOrder) == 0 {
		return nil
	}
	target := e.focusOrder[e.focusPos]
	e.focusTarget = target

	var cmd tea.Cmd
	switch target {
	case eftTitle:
		e.title.Focus()
		cmd = textinput.Blink
	case eftUsername:
		e.username.Focus()
		cmd = textinput.Blink
	case eftPassword:
		e.password.Focus()
		cmd = textinput.Blink
	case eftLink:
		e.link.Focus()
		cmd = textinput.Blink
	case eftTotpSecret:
		e.totpSecret.Focus()
		cmd = textinput.Blink
	case eftCardNumber:
		e.cardNumber.Focus()
		cmd = textinput.Blink
	case eftExpiry:
		e.expiry.Focus()
		cmd = textinput.Blink
	case eftCVV:
		e.cvv.Focus()
		cmd = textinput.Blink
	case eftNotes:
		e.notes.Focus()
		cmd = textarea.Blink
	}
	return cmd
}

func (e *editorModel) focusNext() tea.Cmd {
	if len(e.focusOrder) == 0 {
		return nil
	}
	e.focusPos = (e.focusPos + 1) % len(e.focusOrder)
	return e.applyFocus()
}

func (e *editorModel) focusPrev() tea.Cmd {
	if len(e.focusOrder) == 0 {
		return nil
	}
	e.focusPos--
	if e.focusPos < 0 {
		e.focusPos = len(e.focusOrder) - 1
	}
	return e.applyFocus()
}

func (e *editorModel) updateFocused(msg tea.Msg) tea.Cmd {
	switch e.focusTarget {
	case eftTitle:
		e.title, _ = e.title.Update(msg)
		return textinput.Blink
	case eftUsername:
		e.username, _ = e.username.Update(msg)
		return textinput.Blink
	case eftPassword:
		e.password, _ = e.password.Update(msg)
		e.strength = formatStrengthBar(e.password.Value())
		return textinput.Blink
	case eftLink:
		e.link, _ = e.link.Update(msg)
		return textinput.Blink
	case eftTotpSecret:
		e.totpSecret, _ = e.totpSecret.Update(msg)
		return textinput.Blink
	case eftCardNumber:
		e.cardNumber, _ = e.cardNumber.Update(msg)
		return textinput.Blink
	case eftExpiry:
		e.expiry, _ = e.expiry.Update(msg)
		return textinput.Blink
	case eftCVV:
		e.cvv, _ = e.cvv.Update(msg)
		return textinput.Blink
	case eftNotes:
		e.notes, _ = e.notes.Update(msg)
		return textarea.Blink
	}
	return nil
}

func isTabForward(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyTab || msg.String() == "tab"
}

func isTabBackward(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyShiftTab || msg.String() == "shift+tab"
}
