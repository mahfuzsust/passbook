package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rivo/tview"
)

// addCardFields adds card-specific form fields to the editor.
func addCardFields(ent *Entry) {
	ent.Attachments = nil

	cardNumberField := tview.NewInputField().SetLabel("Card Number").SetText(ent.CardNumber).SetFieldWidth(40)
	cardNumberField.SetAcceptanceFunc(func(text string, last rune) bool {
		if last == 0 {
			return len(text) <= 19
		}
		if len(text) > 19 {
			return false
		}
		for _, r := range text {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	})
	cardNumberField.SetChangedFunc(func(string) { updateEditorSaveState() })
	uiEditorCardNumber = cardNumberField
	uiEditorForm.AddFormItem(cardNumberField)

	expiryField := tview.NewInputField().SetLabel("Expiry (MM/YY)").SetText(ent.Expiry).SetFieldWidth(10)
	expiryField.SetAcceptanceFunc(func(text string, last rune) bool {
		if last == 0 {
			return len(text) <= 5
		}
		if len(text) > 5 {
			return false
		}
		for i, r := range text {
			if i == 2 {
				if r != '/' {
					return false
				}
				continue
			}
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	})
	expiryField.SetChangedFunc(func(string) { updateEditorSaveState() })
	uiEditorExpiry = expiryField
	uiEditorForm.AddFormItem(expiryField)

	cvvField := tview.NewInputField().SetLabel("CVV").SetText(ent.Cvv).SetFieldWidth(5)
	cvvField.SetAcceptanceFunc(func(text string, last rune) bool {
		if last == 0 {
			return len(text) <= 4
		}
		if len(text) > 4 {
			return false
		}
		for _, r := range text {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	})
	cvvField.SetChangedFunc(func(string) { updateEditorSaveState() })
	uiEditorCVV = cvvField
	uiEditorForm.AddFormItem(cvvField)
}

// collectCardFields reads card form values and returns them.
func collectCardFields() (string, string, string) {
	if uiEditorCardNumber == nil || uiEditorExpiry == nil || uiEditorCVV == nil {
		return "", "", ""
	}
	return strings.TrimSpace(uiEditorCardNumber.GetText()),
		strings.TrimSpace(uiEditorExpiry.GetText()),
		strings.TrimSpace(uiEditorCVV.GetText())
}

// validateCardFields validates card-specific fields. Empty values are allowed.
func validateCardFields() error {
	if uiEditingEnt == nil || EntryType(uiEditingEnt.Type) != TypeCard {
		return nil
	}
	if uiEditorCardNumber == nil || uiEditorExpiry == nil || uiEditorCVV == nil {
		return fmt.Errorf("card fields unavailable")
	}

	number := strings.TrimSpace(uiEditorCardNumber.GetText())
	expiry := strings.TrimSpace(uiEditorExpiry.GetText())
	cvv := strings.TrimSpace(uiEditorCVV.GetText())

	if number != "" {
		if len(number) < 13 || len(number) > 19 || !isDigits(number) {
			return fmt.Errorf("card number must be 13-19 digits")
		}
	}

	if expiry != "" {
		if len(expiry) != 5 || expiry[2] != '/' {
			return fmt.Errorf("expiry must be MM/YY")
		}
		mm, yy := expiry[:2], expiry[3:]
		if !isDigits(mm) || !isDigits(yy) {
			return fmt.Errorf("expiry must be MM/YY")
		}
		month, _ := strconv.Atoi(mm)
		if month < 1 || month > 12 {
			return fmt.Errorf("expiry must be MM/YY")
		}
	}

	if cvv != "" {
		if (len(cvv) != 3 && len(cvv) != 4) || !isDigits(cvv) {
			return fmt.Errorf("CVV must be 3 or 4 digits")
		}
	}

	return nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// renderCardView renders the card-type view pane content.
func renderCardView() {
	num := uiCurrentEnt.CardNumber
	if !uiShowSensitive && len(num) > 4 {
		num = "**** **** **** " + num[len(num)-4:]
	}
	uiViewSubtitle.SetText(num)
	btnCopy := styleButton(tview.NewButton("cp").SetSelectedFunc(func() { copySensitive(uiCurrentEnt.CardNumber, "Card") }))
	btnShow := styleButton(tview.NewButton("vw").SetSelectedFunc(func() { uiShowSensitive = !uiShowSensitive; updateViewPane() }))
	uiViewFlex.AddItem(makeRow("Number:", uiViewSubtitle, btnShow, btnCopy), 1, 0, false)

	uiViewDetails.SetText(uiCurrentEnt.Expiry)
	uiViewFlex.AddItem(makeRow("Expiry:", uiViewDetails), 1, 0, false)

	cvv := "***"
	if uiShowSensitive {
		cvv = uiCurrentEnt.Cvv
	}
	uiViewPassword.SetText(cvv)
	uiViewFlex.AddItem(makeRow("CVV:", uiViewPassword), 1, 0, false)
}
