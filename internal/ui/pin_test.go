package ui

import (
	"strings"
	"testing"
	"time"

	"passbook/internal/store"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func TestNewPinModel(t *testing.T) {
	p := newPinModel()
	if p.mode != "setup" {
		t.Fatalf("expected setup mode, got %q", p.mode)
	}
}

func TestNewPinVerifyModelPinMode(t *testing.T) {
	p := newPinVerifyModel(&store.PinConfig{Mode: "pin"})
	if p.mode != "verify" {
		t.Fatalf("expected verify mode, got %q", p.mode)
	}
	if p.verifyInput.CharLimit != 6 {
		t.Fatalf("expected char limit 6, got %d", p.verifyInput.CharLimit)
	}
}

func TestUpdatePinSetupNavigatesAndTransitions(t *testing.T) {
	m := &Model{screen: screenPinSetup, pin: newPinModel(), width: 80, height: 24}

	nm, _ := m.updatePinSetup("right")
	*m = nm
	if m.pin.setupBtn != 1 {
		t.Fatalf("expected setupBtn 1, got %d", m.pin.setupBtn)
	}
	nm, _ = m.updatePinSetup("left")
	*m = nm
	if m.pin.setupBtn != 0 {
		t.Fatalf("expected setupBtn 0, got %d", m.pin.setupBtn)
	}

	nm, _ = m.updatePinSetup("enter")
	*m = nm
	if m.screen != screenPinCreate || m.pin.mode != "create" {
		t.Fatalf("expected to move to pin create, got screen=%v mode=%q", m.screen, m.pin.mode)
	}
}

func TestUpdatePinSetupTotpChoice(t *testing.T) {
	m := &Model{screen: screenPinSetup, pin: newPinModel(), width: 80, height: 24}
	m.pin.setupBtn = 1

	nm, _ := m.updatePinSetup("enter")
	*m = nm
	if m.screen != screenTotpSetup || m.pin.mode != "totp" {
		t.Fatalf("expected totp setup, got screen=%v mode=%q", m.screen, m.pin.mode)
	}
	if m.pin.pendingSecret == "" {
		t.Fatalf("expected a pending secret to be generated")
	}
}

func TestUpdatePinSetupEscQuits(t *testing.T) {
	m := &Model{screen: screenPinSetup, pin: newPinModel()}
	_, cmd := m.updatePinSetup("esc")
	if cmd == nil {
		t.Fatalf("expected esc to return a quit cmd")
	}
}

func newPinCreateTestModel(t *testing.T) *Model {
	t.Helper()
	m := &Model{screen: screenPinCreate, store: newTestStore(t), width: 80, height: 24}
	m.pin = newPinModel()
	nm, _ := m.updatePinSetup("enter")
	*m = nm
	return m
}

func TestUpdatePinCreateFocusNavigation(t *testing.T) {
	m := newPinCreateTestModel(t)

	nm, _ := m.updatePinCreate(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm
	if m.pin.createFocus != 1 {
		t.Fatalf("expected createFocus 1, got %d", m.pin.createFocus)
	}
	nm, _ = m.updatePinCreate(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm
	if m.pin.createFocus != 2 {
		t.Fatalf("expected createFocus 2 (buttons), got %d", m.pin.createFocus)
	}
	nm, _ = m.updatePinCreate(tea.KeyMsg{Type: tea.KeyShiftTab})
	*m = nm
	if m.pin.createFocus != 1 {
		t.Fatalf("expected shift+tab to move back to 1, got %d", m.pin.createFocus)
	}
}

func TestUpdatePinCreateButtonNavigation(t *testing.T) {
	m := newPinCreateTestModel(t)
	m.pin.createFocus = 2

	nm, _ := m.updatePinCreate(tea.KeyMsg{Type: tea.KeyRight})
	*m = nm
	if m.pin.createBtn != 1 {
		t.Fatalf("expected createBtn 1, got %d", m.pin.createBtn)
	}
	nm, _ = m.updatePinCreate(tea.KeyMsg{Type: tea.KeyLeft})
	*m = nm
	if m.pin.createBtn != 0 {
		t.Fatalf("expected createBtn 0, got %d", m.pin.createBtn)
	}
}

func TestUpdatePinCreateEscReturnsToSetup(t *testing.T) {
	m := newPinCreateTestModel(t)
	nm, _ := m.updatePinCreate(tea.KeyMsg{Type: tea.KeyEsc})
	*m = nm
	if m.screen != screenPinSetup || m.pin.mode != "setup" {
		t.Fatalf("expected to return to pin setup, got screen=%v mode=%q", m.screen, m.pin.mode)
	}
}

func TestDoSavePinValidation(t *testing.T) {
	m := newPinCreateTestModel(t)
	m.pin.pinInput.SetValue("123")
	m.pin.confirmInput.SetValue("123")
	m.doSavePin()
	if !strings.Contains(m.pin.createStatus, "6 digits") {
		t.Fatalf("expected 6-digit validation error, got %q", m.pin.createStatus)
	}

	m.pin.pinInput.SetValue("123456")
	m.pin.confirmInput.SetValue("654321")
	m.doSavePin()
	if !strings.Contains(m.pin.createStatus, "not match") {
		t.Fatalf("expected mismatch error, got %q", m.pin.createStatus)
	}
}

func TestDoSavePinSuccessEntersMain(t *testing.T) {
	m := newPinCreateTestModel(t)
	m.pin.pinInput.SetValue("123456")
	m.pin.confirmInput.SetValue("123456")
	m.doSavePin()
	if m.screen != screenMain {
		t.Fatalf("expected to enter main screen, got %v (status=%q)", m.screen, m.pin.createStatus)
	}
	if !m.store.PinConfigExists() {
		t.Fatalf("expected pin config to be persisted")
	}
}

func TestUpdatePinCreateEnterAdvancesThroughFieldsAndSaves(t *testing.T) {
	m := newPinCreateTestModel(t)
	m.pin.pinInput.SetValue("111111")
	m.pin.confirmInput.SetValue("111111")

	nm, _ := m.updatePinCreate(tea.KeyMsg{Type: tea.KeyEnter}) // title -> confirm
	*m = nm
	if m.pin.createFocus != 1 {
		t.Fatalf("expected focus on confirm field, got %d", m.pin.createFocus)
	}
	nm, _ = m.updatePinCreate(tea.KeyMsg{Type: tea.KeyEnter}) // confirm -> buttons
	*m = nm
	if m.pin.createFocus != 2 {
		t.Fatalf("expected focus on buttons, got %d", m.pin.createFocus)
	}
	nm, _ = m.updatePinCreate(tea.KeyMsg{Type: tea.KeyEnter}) // save
	*m = nm
	if m.screen != screenMain {
		t.Fatalf("expected save to enter main, got %v", m.screen)
	}
}

func newTotpSetupTestModel(t *testing.T) *Model {
	t.Helper()
	m := &Model{screen: screenPinSetup, store: newTestStore(t), width: 80, height: 24}
	m.pin = newPinModel()
	m.pin.setupBtn = 1
	nm, _ := m.updatePinSetup("enter")
	*m = nm
	return m
}

func TestStartTotpSetupPopulatesQR(t *testing.T) {
	m := newTotpSetupTestModel(t)
	if m.pin.qrText == "" {
		t.Fatalf("expected qr text to be rendered")
	}
	if m.pin.pendingSecret == "" {
		t.Fatalf("expected pending secret")
	}
}

func TestUpdateTotpSetupCopySecret(t *testing.T) {
	m := newTotpSetupTestModel(t)
	nm, _ := m.updateTotpSetup(tea.KeyMsg{Type: tea.KeyCtrlY})
	*m = nm
	if !strings.Contains(m.pin.totpStatus, "copied") {
		t.Fatalf("expected copied status, got %q", m.pin.totpStatus)
	}
}

func TestUpdateTotpSetupNavigation(t *testing.T) {
	m := newTotpSetupTestModel(t)
	nm, _ := m.updateTotpSetup(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm
	if m.pin.totpFocus != 1 {
		t.Fatalf("expected totpFocus 1, got %d", m.pin.totpFocus)
	}
	nm, _ = m.updateTotpSetup(tea.KeyMsg{Type: tea.KeyRight})
	*m = nm
	if m.pin.totpBtn != 1 {
		t.Fatalf("expected totpBtn 1, got %d", m.pin.totpBtn)
	}
	nm, _ = m.updateTotpSetup(tea.KeyMsg{Type: tea.KeyLeft})
	*m = nm
	if m.pin.totpBtn != 0 {
		t.Fatalf("expected totpBtn 0, got %d", m.pin.totpBtn)
	}
	nm, _ = m.updateTotpSetup(tea.KeyMsg{Type: tea.KeyShiftTab})
	*m = nm
	if m.pin.totpFocus != 0 {
		t.Fatalf("expected shift+tab to return focus to code input, got %d", m.pin.totpFocus)
	}
}

func TestUpdateTotpSetupEscClearsSecret(t *testing.T) {
	m := newTotpSetupTestModel(t)
	nm, _ := m.updateTotpSetup(tea.KeyMsg{Type: tea.KeyEsc})
	*m = nm
	if m.pin.pendingSecret != "" || m.screen != screenPinSetup {
		t.Fatalf("expected esc to clear secret and return to setup")
	}
}

func TestDoSaveTotpValidation(t *testing.T) {
	m := newTotpSetupTestModel(t)
	m.pin.totpCode.SetValue("123")
	m.doSaveTotp()
	if !strings.Contains(m.pin.totpStatus, "6-digit code") {
		t.Fatalf("expected 6-digit validation message, got %q", m.pin.totpStatus)
	}

	m.pin.totpCode.SetValue("000000")
	m.doSaveTotp()
	if !strings.Contains(m.pin.totpStatus, "Invalid code") {
		t.Fatalf("expected invalid code message, got %q", m.pin.totpStatus)
	}
}

func TestDoSaveTotpSuccess(t *testing.T) {
	m := newTotpSetupTestModel(t)
	code, err := totp.GenerateCodeCustom(m.pin.pendingSecret, time.Now(), totp.ValidateOpts{
		Period: 30, Skew: 1, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	m.pin.totpCode.SetValue(code)
	m.doSaveTotp()
	if m.screen != screenMain {
		t.Fatalf("expected to enter main after valid totp, got %v (status=%q)", m.screen, m.pin.totpStatus)
	}
}

func newPinVerifyTestModel(t *testing.T, cfg *store.PinConfig) *Model {
	t.Helper()
	m := &Model{screen: screenPinVerify, store: newTestStore(t), width: 80, height: 24}
	m.pin = newPinVerifyModel(cfg)
	return m
}

func TestUpdatePinVerifyEscQuits(t *testing.T) {
	m := newPinVerifyTestModel(t, &store.PinConfig{Mode: "pin"})
	_, cmd := m.updatePinVerify(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("expected esc to quit")
	}
}

func TestUpdatePinVerifyNavigation(t *testing.T) {
	m := newPinVerifyTestModel(t, &store.PinConfig{Mode: "pin"})
	nm, _ := m.updatePinVerify(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm
	if m.pin.verifyFocus != 1 {
		t.Fatalf("expected verifyFocus 1, got %d", m.pin.verifyFocus)
	}
	nm, _ = m.updatePinVerify(tea.KeyMsg{Type: tea.KeyTab})
	*m = nm
	if m.pin.verifyBtn != 1 {
		t.Fatalf("expected verifyBtn 1, got %d", m.pin.verifyBtn)
	}
	nm, _ = m.updatePinVerify(tea.KeyMsg{Type: tea.KeyLeft})
	*m = nm
	if m.pin.verifyBtn != 0 {
		t.Fatalf("expected verifyBtn back to 0, got %d", m.pin.verifyBtn)
	}
}

func TestDoVerifyPinWrongPin(t *testing.T) {
	m := newPinVerifyTestModel(t, &store.PinConfig{Mode: "pin", PinKey: []byte("k"), PinTag: "wontmatch"})
	m.pin.verifyInput.SetValue("000000")
	m.doVerifyPin()
	if !strings.Contains(m.pin.verifyStatus, "Wrong PIN") {
		t.Fatalf("expected wrong pin status, got %q", m.pin.verifyStatus)
	}
}

func TestDoVerifyPinShortCode(t *testing.T) {
	m := newPinVerifyTestModel(t, &store.PinConfig{Mode: "pin"})
	m.pin.verifyInput.SetValue("123")
	m.doVerifyPin()
	if !strings.Contains(m.pin.verifyStatus, "6-digit code") {
		t.Fatalf("expected 6-digit code message, got %q", m.pin.verifyStatus)
	}
}

func TestValidateTOTP(t *testing.T) {
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "PassBook", AccountName: "v"})
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	code, err := totp.GenerateCode(key.Secret(), time.Now())
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	if !validateTOTP(code, key.Secret()) {
		t.Fatalf("expected valid code to pass validation")
	}
	if validateTOTP("000000", key.Secret()) {
		t.Fatalf("did not expect an arbitrary code to validate")
	}
}

func TestEnterMain(t *testing.T) {
	m := &Model{store: newTestStore(t)}
	m.enterMain()
	if m.screen != screenMain {
		t.Fatalf("expected screenMain, got %v", m.screen)
	}
}

func TestPinViewsRenderWithoutPanicking(t *testing.T) {
	m := newPinCreateTestModel(t)
	if v := m.viewPinSetup(); v == "" {
		t.Fatalf("expected non-empty pin setup view")
	}
	if v := m.viewPinCreate(); !strings.Contains(v, "PIN") {
		t.Fatalf("expected pin create view to mention PIN, got %q", v)
	}

	totpM := newTotpSetupTestModel(t)
	if v := totpM.viewTotpSetup(); !strings.Contains(v, "Authenticator") {
		t.Fatalf("expected totp setup view to mention Authenticator, got %q", v)
	}

	verifyM := newPinVerifyTestModel(t, &store.PinConfig{Mode: "totp"})
	if v := verifyM.viewPinVerify(); !strings.Contains(v, "Authenticator Code") {
		t.Fatalf("expected totp verify title, got %q", v)
	}
}
