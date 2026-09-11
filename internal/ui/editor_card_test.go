package ui

import "testing"

func TestCardFieldsValidation(t *testing.T) {
	if err := validateCardFieldValues("123", "", "", TypeCard); err == nil {
		t.Fatalf("expected error for short card number")
	}

	if err := validateCardFieldValues("1234567812345678", "13/99", "", TypeCard); err == nil {
		t.Fatalf("expected error for invalid month")
	}

	if err := validateCardFieldValues("1234567812345678", "12/34", "12", TypeCard); err == nil {
		t.Fatalf("expected error for short CVV")
	}

	if err := validateCardFieldValues("1234567812345678", "12/34", "123", TypeCard); err != nil {
		t.Fatalf("expected valid card fields, got: %v", err)
	}

	if err := validateCardFieldValues("371449635398431", "12/34", "1234", TypeCard); err != nil {
		t.Fatalf("expected valid amex card fields, got: %v", err)
	}

	if err := validateCardFieldValues("4111111111111111", "12/34", "1234", TypeCard); err != nil {
		t.Fatalf("expected valid card fields with 4-digit CVV, got: %v", err)
	}

	if err := validateCardFieldValues("4111111111111111", "12/34", "12345", TypeCard); err == nil {
		t.Fatalf("expected error for 5-digit CVV")
	}
}

func TestCollectCardFieldsTrims(t *testing.T) {
	e := editorModel{entryType: TypeCard}
	e.cardNumber.SetValue(" 1234567812345678 ")
	e.expiry.SetValue(" 12/34 ")
	e.cvv.SetValue(" 123 ")

	num, exp, cvv := collectCardFields(e)
	if num != "1234567812345678" || exp != "12/34" || cvv != "123" {
		t.Fatalf("expected trimmed values, got %q %q %q", num, exp, cvv)
	}
}
