package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func styleButton(b *tview.Button) *tview.Button {
	b.SetBackgroundColor(colorUnfocusedBg)
	b.SetLabelColor(tcell.ColorWhite)
	b.SetFocusFunc(func() {
		b.SetLabelColor(colorFocusedBg)
		b.SetBackgroundColor(tcell.ColorWhite)
	})
	b.SetBlurFunc(func() {
		b.SetBackgroundColor(colorUnfocusedBg)
		b.SetLabelColor(tcell.ColorWhite)
	})
	return b
}

func styleInput(f *tview.InputField) *tview.InputField {
	f.SetFieldBackgroundColor(colorUnfocusedBg)
	f.SetFocusFunc(func() { f.SetFieldBackgroundColor(colorFocusedBg) })
	f.SetBlurFunc(func() { f.SetFieldBackgroundColor(colorUnfocusedBg) })
	return f
}

func styleForm(f *tview.Form) {
	for i := 0; i < f.GetFormItemCount(); i++ {
		if input, ok := f.GetFormItem(i).(*tview.InputField); ok {
			styleInput(input)
		}
	}
	for i := 0; i < f.GetButtonCount(); i++ {
		styleButton(f.GetButton(i))
	}
}

func enableButtonNav(form *tview.Form) {
	prev := form.GetInputCapture()
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		_, btn := form.GetFocusedItemIndex()
		if btn >= 0 {
			switch event.Key() {
			case tcell.KeyLeft:
				if btn > 0 {
					uiApp.SetFocus(form.GetButton(btn - 1))
				}
				return nil
			case tcell.KeyRight:
				if btn < form.GetButtonCount()-1 {
					uiApp.SetFocus(form.GetButton(btn + 1))
				}
				return nil
			}
		}
		if prev != nil {
			return prev(event)
		}
		return event
	})
}

func enableModalButtonNav(modal *tview.Modal) {
	prev := modal.GetInputCapture()
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			return tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone)
		case tcell.KeyRight:
			return tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone)
		}
		if prev != nil {
			return prev(event)
		}
		return event
	})
}

func makeRow(label string, content *tview.TextView, buttons ...*tview.Button) *tview.Flex {
	f := tview.NewFlex().SetDirection(tview.FlexColumn)
	f.AddItem(tview.NewTextView().SetText(label).SetTextColor(tcell.ColorYellow), 12, 0, false)
	f.AddItem(content, 0, 1, false)
	for _, b := range buttons {
		f.AddItem(tview.NewTextView().SetText(" "), 1, 0, false)
		f.AddItem(b, 5, 0, false)
	}
	return f
}
