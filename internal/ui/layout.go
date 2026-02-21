package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type responsiveModal struct {
	*tview.Flex
	content       tview.Primitive
	minWidth      int
	minHeight     int
	maxWidth      int
	maxHeight     int
	widthPercent  float64
	heightPercent float64
	lastW         int
	lastH         int
}

func newResponsiveModal(p tview.Primitive, minWidth, minHeight, maxWidth, maxHeight int, widthPercent, heightPercent float64) *responsiveModal {
	r := &responsiveModal{
		Flex:          tview.NewFlex(),
		content:       p,
		minWidth:      minWidth,
		minHeight:     minHeight,
		maxWidth:      maxWidth,
		maxHeight:     maxHeight,
		widthPercent:  widthPercent,
		heightPercent: heightPercent,
	}

	innerFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(p, minHeight, 1, true).
		AddItem(nil, 0, 1, false)

	r.Flex.AddItem(nil, 0, 1, false).
		AddItem(innerFlex, minWidth, 1, true).
		AddItem(nil, 0, 1, false)

	return r
}

func (r *responsiveModal) Draw(screen tcell.Screen) {
	_, _, w, h := r.GetRect()

	if w != r.lastW || h != r.lastH {
		modalW := int(float64(w) * r.widthPercent)
		if modalW < r.minWidth {
			modalW = r.minWidth
		}
		if r.maxWidth > 0 && modalW > r.maxWidth {
			modalW = r.maxWidth
		}
		if modalW > w {
			modalW = w
		}

		modalH := int(float64(h) * r.heightPercent)
		if modalH < r.minHeight {
			modalH = r.minHeight
		}
		if r.maxHeight > 0 && modalH > r.maxHeight {
			modalH = r.maxHeight
		}
		if modalH > h {
			modalH = h
		}

		padLeft := (w - modalW) / 2
		padRight := w - modalW - padLeft
		padTop := (h - modalH) / 2
		padBottom := h - modalH - padTop

		r.Flex.Clear()

		innerFlex := tview.NewFlex().SetDirection(tview.FlexRow)
		if padTop > 0 {
			innerFlex.AddItem(nil, padTop, 0, false)
		}
		innerFlex.AddItem(r.content, modalH, 0, true)
		if padBottom > 0 {
			innerFlex.AddItem(nil, padBottom, 0, false)
		}

		if padLeft > 0 {
			r.Flex.AddItem(nil, padLeft, 0, false)
		}
		r.Flex.AddItem(innerFlex, modalW, 0, true)
		if padRight > 0 {
			r.Flex.AddItem(nil, padRight, 0, false)
		}

		r.lastW, r.lastH = w, h
	}

	r.Flex.Draw(screen)
}

type responsiveSplit struct {
	*tview.Flex
	left, right tview.Primitive
	leftRatio   float64
	minLeft     int
	minRight    int
	lastW       int
	lastH       int
}

func newResponsiveSplit(left, right tview.Primitive, leftRatio float64, minLeft, minRight int) *responsiveSplit {
	r := &responsiveSplit{
		Flex:      tview.NewFlex(),
		left:      left,
		right:     right,
		leftRatio: leftRatio,
		minLeft:   minLeft,
		minRight:  minRight,
	}
	r.Flex.AddItem(left, 0, 1, true)
	r.Flex.AddItem(right, 0, 1, false)
	return r
}

func (r *responsiveSplit) Draw(screen tcell.Screen) {
	x, y, w, h := r.GetRect()
	if w != r.lastW || h != r.lastH {
		leftW := int(float64(w) * r.leftRatio)
		if leftW < r.minLeft {
			leftW = r.minLeft
		}
		if w-leftW < r.minRight {
			leftW = w - r.minRight
		}
		if leftW < 0 {
			leftW = 0
		}
		if w-leftW < 0 {
			leftW = 0
		}

		r.Flex.SetRect(x, y, w, h)
		r.Flex.ResizeItem(r.left, leftW, 0)
		r.Flex.ResizeItem(r.right, 0, 1)

		r.lastW, r.lastH = w, h
	}
	r.Flex.Draw(screen)
}
