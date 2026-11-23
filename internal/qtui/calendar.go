package qtui

import (
	"fmt"
	"strconv"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// Calendar days are plain dates: midnight UTC of the day they name, so date
// arithmetic and comparisons never meet a time zone or a DST shift.

// dayOf is the calendar day of an instant in its own time zone.
func dayOf(instant time.Time) time.Time {
	return time.Date(instant.Year(), instant.Month(), instant.Day(), 0, 0, 0, 0, time.UTC)
}

// monthOf is the first day of a day's month.
func monthOf(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// monthGrid lists the days a month page shows: constants.CalendarWeeks full
// Monday-first weeks starting with the week of the month's first day, so the
// page also carries the neighbouring months' adjacent days.
func monthGrid(month time.Time) []time.Time {
	first := monthOf(month)
	// time.Weekday counts from Sunday; shift so Monday is column zero.
	column := (int(first.Weekday()) + constants.DaysPerWeek - int(time.Monday)) % constants.DaysPerWeek
	start := first.AddDate(0, 0, -column)
	days := make([]time.Time, 0, constants.CalendarWeeks*constants.DaysPerWeek)
	for idx := 0; idx < constants.CalendarWeeks*constants.DaysPerWeek; idx++ {
		days = append(days, start.AddDate(0, 0, idx))
	}
	return days
}

// showCalendarPopup opens the date picker (calendar.ui) under anchor on the
// month of selected, which is highlighted. Days before today are disabled —
// the theme strikes them out — and the page cannot go back past today's
// month. Clicking a day reports it through onPick and closes the popup, as
// does clicking anywhere outside it.
func showCalendarPopup(anchor *qt.QWidget, selected, today time.Time, onPick func(picked time.Time)) {
	loaded := loadForm(calendarForm)
	popup := loaded.frame("calendarPopup")
	popup.SetParent2(anchor.Window(), qt.Popup)
	popup.SetAttribute(qt.WA_DeleteOnClose)
	// A popup is its own window, so it carries the theme itself, like the
	// dialog cards.
	popup.SetStyleSheet(themeStyleSheet)
	setState(popup.QWidget, colorSchemeProperty, activeColorScheme)

	monthLabel := loaded.label("monthLabel")
	previousButton := loaded.button("previousMonthButton")
	nextButton := loaded.button("nextMonthButton")

	// One button per grid cell, under the form's weekday header row; show
	// refills them for the month on display.
	grid := loaded.grid("daysGrid")
	shown := monthOf(selected)
	var cells []time.Time
	buttons := make([]*qt.QPushButton, 0, constants.CalendarWeeks*constants.DaysPerWeek)
	for idx := 0; idx < constants.CalendarWeeks*constants.DaysPerWeek; idx++ {
		cell := idx // capture per iteration for the click callback
		button := loadForm(calendarDayForm).button("dayButton")
		button.OnClicked(func() {
			onPick(cells[cell])
			popup.Close()
		})
		grid.AddWidget2(button.QWidget, 1+idx/constants.DaysPerWeek, idx%constants.DaysPerWeek)
		buttons = append(buttons, button)
	}
	show := func() {
		cells = monthGrid(shown)
		monthLabel.SetText(shown.Format(constants.TimeFmtMonthYear))
		previousButton.SetEnabled(shown.After(monthOf(today)))
		for idx, day := range cells {
			button := buttons[idx]
			button.SetText(strconv.Itoa(day.Day()))
			button.SetEnabled(!day.Before(today))
			setState(button.QWidget, "outside", fmt.Sprint(day.Month() != shown.Month()))
			setState(button.QWidget, "today", fmt.Sprint(day.Equal(today)))
			setState(button.QWidget, "selected", fmt.Sprint(day.Equal(selected)))
		}
	}
	previousButton.OnClicked(func() {
		shown = shown.AddDate(0, -1, 0)
		show()
	})
	nextButton.OnClicked(func() {
		shown = shown.AddDate(0, 1, 0)
		show()
	})
	show()

	popup.MoveWithQPoint(anchor.MapToGlobalWithQPoint(qt.NewQPoint2(0, anchor.Height())))
	popup.Show()
}
