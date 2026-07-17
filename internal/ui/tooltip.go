package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// hoverTip wraps a content object and shows a small text popup while the pointer
// hovers over it. Fyne has no native tooltips; this is a minimal generic
// stand-in, to be improved later.
type hoverTip struct {
	widget.BaseWidget
	content fyne.CanvasObject
	text    string
	popup   *widget.PopUp
}

func newHoverTip(content fyne.CanvasObject, text string) *hoverTip {
	h := &hoverTip{content: content, text: text}
	h.ExtendBaseWidget(h)
	return h
}

func (h *hoverTip) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(h.content)
}

func (h *hoverTip) MouseIn(ev *desktop.MouseEvent) {
	if h.text == "" {
		return
	}
	c := fyne.CurrentApp().Driver().CanvasForObject(h)
	if c == nil {
		return
	}
	h.popup = widget.NewPopUp(widget.NewLabel(h.text), c)
	h.popup.ShowAtPosition(ev.AbsolutePosition.Add(fyne.NewPos(10, 10)))
}

func (h *hoverTip) MouseMoved(*desktop.MouseEvent) {}

func (h *hoverTip) MouseOut() {
	if h.popup != nil {
		h.popup.Hide()
		h.popup = nil
	}
}
