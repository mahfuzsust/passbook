package ui

import "testing"

func TestCardFieldsValidation(t *testing.T) {
	resetEditorTestState()
	uiEditingEnt = &Entry{Type: string(TypeCard)}
	addCardFields(uiEditingEnt)

	uiEditorCardNumber.SetText("123")
	if err := validateCardFields(); err == nil {
		t.Fatalf("expected error for short card number")
	}

	uiEditorCardNumber.SetText("1234567812345678")
	uiEditorExpiry.SetText("13/99")
	if err := validateCardFields(); err == nil {
		t.Fatalf("expected error for invalid month")
	}

	uiEditorExpiry.SetText("12/34")
	uiEditorCVV.SetText("12")
	if err := validateCardFields(); err == nil {
		t.Fatalf("expected error for short CVV")
	}

	uiEditorCVV.SetText("123")
	if err := validateCardFields(); err != nil {
		t.Fatalf("expected valid card fields, got: %v", err)
	}

	// Amex: 15-digit card number with 4-digit CVV
	uiEditorCardNumber.SetText("371449635398431")
	uiEditorCVV.SetText("1234")
	if err := validateCardFields(); err != nil {
		t.Fatalf("expected valid amex card fields, got: %v", err)
	}

	// 4-digit CVV alone with 16-digit card number should also be valid
	uiEditorCardNumber.SetText("4111111111111111")
	uiEditorCVV.SetText("1234")
	if err := validateCardFields(); err != nil {
		t.Fatalf("expected valid card fields with 4-digit CVV, got: %v", err)
	}

	// 5-digit CVV should fail
	uiEditorCVV.SetText("12345")
	if err := validateCardFields(); err == nil {
		t.Fatalf("expected error for 5-digit CVV")
	}
}

func TestCollectCardFieldsTrims(t *testing.T) {
	resetEditorTestState()
	uiEditingEnt = &Entry{Type: string(TypeCard)}
	addCardFields(uiEditingEnt)

	uiEditorCardNumber.SetText(" 1234567812345678 ")
	uiEditorExpiry.SetText(" 12/34 ")
	uiEditorCVV.SetText(" 123 ")

	num, exp, cvv := collectCardFields()
	if num != "1234567812345678" || exp != "12/34" || cvv != "123" {
		t.Fatalf("expected trimmed values, got %q %q %q", num, exp, cvv)
	}
}
