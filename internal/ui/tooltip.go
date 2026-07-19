package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/MarioStoilov/simplestonks/internal/constants"
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
	tip := &hoverTip{content: content, text: text}
	tip.ExtendBaseWidget(tip)
	return tip
}

func (tip *hoverTip) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(tip.content)
}

func (tip *hoverTip) MouseIn(event *desktop.MouseEvent) {
	if tip.text == "" {
		return
	}
	tipCanvas := fyne.CurrentApp().Driver().CanvasForObject(tip)
	if tipCanvas == nil {
		return
	}
	tip.popup = widget.NewPopUp(widget.NewLabel(tip.text), tipCanvas)
	tip.popup.ShowAtPosition(event.AbsolutePosition.Add(fyne.NewPos(constants.TooltipOffset, constants.TooltipOffset)))
}

func (tip *hoverTip) MouseMoved(*desktop.MouseEvent) {}

func (tip *hoverTip) MouseOut() {
	if tip.popup != nil {
		tip.popup.Hide()
		tip.popup = nil
	}
}
