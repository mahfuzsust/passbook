package ui

import (
	"testing"

	"passbook/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPinUpdateInputsPerMode(t *testing.T) {
	p := pinModel{mode: "create"}
	p.pinInput = newTextInput("PIN", true)
	p.pinInput.Focus()
	p.confirmInput = newTextInput("Confirm", true)
	p.createFocus = 0
	np, _ := p.updateInputs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if np.pinInput.Value() != "1" {
		t.Fatalf("expected pin input updated, got %q", np.pinInput.Value())
	}

	p2 := pinModel{mode: "totp", totpFocus: 0}
	p2.totpCode = newTextInput("code", false)
	p2.totpCode.Focus()
	np2, _ := p2.updateInputs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if np2.totpCode.Value() != "2" {
		t.Fatalf("expected totp code updated, got %q", np2.totpCode.Value())
	}

	p3 := pinModel{mode: "verify"}
	p3.verifyInput = newTextInput("code", false)
	p3.verifyInput.Focus()
	np3, _ := p3.updateInputs(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	if np3.verifyInput.Value() != "3" {
		t.Fatalf("expected verify input updated, got %q", np3.verifyInput.Value())
	}
}

func TestUpdatePinKeyDispatchesByMode(t *testing.T) {
	m := &Model{screen: screenPinSetup, pin: newPinModel(), width: 80, height: 24}
	nm, _ := m.updatePinKey(tea.KeyMsg{Type: tea.KeyRight})
	*m = nm
	if m.pin.setupBtn != 1 {
		t.Fatalf("expected setup dispatch to move setupBtn")
	}

	m2 := newPinCreateTestModel(t)
	nm2, _ := m2.updatePinKey(tea.KeyMsg{Type: tea.KeyTab})
	*m2 = nm2
	if m2.pin.createFocus != 1 {
		t.Fatalf("expected create dispatch to move focus")
	}

	m3 := newTotpSetupTestModel(t)
	nm3, _ := m3.updatePinKey(tea.KeyMsg{Type: tea.KeyTab})
	*m3 = nm3
	if m3.pin.totpFocus != 1 {
		t.Fatalf("expected totp dispatch to move focus")
	}

	m4 := &Model{screen: screenPinVerify, store: newTestStore(t)}
	m4.pin = newPinVerifyModel(&store.PinConfig{Mode: "pin"})
	nm4, _ := m4.updatePinKey(tea.KeyMsg{Type: tea.KeyTab})
	*m4 = nm4
	if m4.pin.verifyFocus != 1 {
		t.Fatalf("expected verify dispatch to move focus")
	}
}
