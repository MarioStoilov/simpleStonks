package qtui

import (
	"os"
	"runtime"
	"strings"
	"testing"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"

	"github.com/MarioStoilov/simplestonks/internal/model"
)

// TestMain gives the tests a Qt application on the offscreen platform, so
// they run without a display. The Qt event loop runs on the main thread
// while the tests run beside it, so a test that needs the UI thread (timers,
// tooltips, grabbing pixels) can hop onto it with mainthread.Wait.
func TestMain(main *testing.M) {
	runtime.LockOSThread()
	os.Setenv("QT_QPA_PLATFORM", "offscreen")
	qt.NewQApplication(os.Args)
	// Tests open and close windows; only the end of the run may stop the loop.
	qt.QGuiApplication_SetQuitOnLastWindowClosed(false)
	go func() {
		code := main.Run()
		mainthread.Wait(func() { qt.QCoreApplication_ExitWithRetcode(code) })
	}()
	os.Exit(qt.QApplication_Exec())
}

// formNames lists, per form, the object names the Go code looks up. A
// rename in Qt Designer that the Go side does not follow fails here rather
// than panicking at run time.
var formNames = map[string][]string{
	tileForm: {
		"tile", "symbolLabel", "nameLabel", "priceLabel", "changeLabel", "chart",
		"controls", "moveLeftButton", "moveRightButton", "removeButton",
	},
	sidebarTileForm: {
		"tile", "symbolLabel", "nameLabel", "priceLabel", "changeLabel", "chart",
	},
	homeForm: {
		"homeRoot", "homeBar", "editButton", "addButton", "settingsButton",
		"aboutButton", "scroll", "gridContent", "grid",
	},
	cardForm:      {"card", "topBar", "offlineLabel", "minimiseButton", "closeButton", "stack"},
	alertPillForm: {"pill", "priceLabel", "expiryIcon", "removeButton"},
	calendarForm: {
		"calendarPopup", "previousMonthButton", "monthLabel", "nextMonthButton", "daysGrid",
	},
	calendarDayForm: {"dayButton"},
	alertForm: {
		"alertContent", "currentLabel", "priceEdit", "percentEdit", "expiryEdit",
		"expiryClearButton", "errorLabel",
		"cancelButton", "addButton",
	},
	detailForm: {
		"detailRoot", "sidebarScroll", "sidebarContent", "sidebarBox", "backButton",
		"symbolLabel", "nameLabel", "alertButton", "extendedToggle", "priceLabel",
		"changeLabel", "extendedLabel", "rangeBar", "chart", "alertsBar", "alertsBox",
	},
}

func TestFormsProvideTheExpectedObjects(t *testing.T) {
	for name, expected := range formNames {
		loaded := loadForm(name)
		for _, object := range expected {
			if _, ok := loaded.objects[object]; !ok {
				t.Errorf("form %s has no object named %q", name, object)
			}
		}
	}
}

// TestFormAccessorsMatchWidgetTypes checks that every widget the Go code
// pulls out of a form is the type it expects (the accessors panic on a
// mismatch, e.g. after a widget's class changes in Designer).
func TestFormAccessorsMatchWidgetTypes(t *testing.T) {
	grid := loadForm(tileForm)
	grid.frame("tile")
	grid.label("symbolLabel")
	grid.chart("chart")
	grid.widget("controls")
	grid.button("moveLeftButton")

	sidebar := loadForm(sidebarTileForm)
	sidebar.frame("tile")
	sidebar.chart("chart")

	home := loadForm(homeForm)
	home.button("editButton")
	home.scrollArea("scroll")
	home.widget("gridContent")
	home.grid("grid")

	card := loadForm(cardForm)
	card.widget("card")
	card.label("offlineLabel")
	card.stack("stack")

	detail := loadForm(detailForm)
	detail.checkBox("extendedToggle")
	detail.chart("chart")
	detail.vbox("sidebarBox")
	detail.hbox("alertsBox")
	detail.scrollArea("sidebarScroll")

	settings := loadForm(settingsForm)
	settings.stack("pages")
	settings.comboBox("rangeBox")
	settings.lineEdit("refreshEdit")
	settings.checkBox("extendedBox")
	settings.slider("opacitySlider")
	settings.checkBox("alertLinesBox")
	settings.comboBox("soundRoseBox")
	settings.button("soundRoseTestButton")
	settings.label("soundRoseLabel")
	settings.comboBox("soundFellBox")
	settings.button("soundFellTestButton")
	settings.label("soundFellLabel")
	settings.button("alertLineSwatch")
	settings.slider("alertLineSlider")
	settings.button("saveButton")

	alert := loadForm(alertForm)
	alert.label("currentLabel")
	alert.lineEdit("priceEdit")
	alert.lineEdit("percentEdit")
	alert.lineEdit("expiryEdit")
	alert.button("expiryClearButton")
	calendar := loadForm(calendarForm)
	calendar.button("previousMonthButton")
	calendar.label("monthLabel")
	calendar.grid("daysGrid")
	loadForm(calendarDayForm).button("dayButton")
	alert.button("addButton")

	loadForm(searchForm).listWidget("results")
	loadForm(cardDialogForm).vbox("body")
	loadForm(previewForm).chart("chart")
}

// TestDetailFormHasEveryRangeToggle keeps the form in step with the model:
// the detail view looks each range's button up by object name.
func TestDetailFormHasEveryRangeToggle(t *testing.T) {
	loaded := loadForm(detailForm)
	for _, rangeOption := range model.Ranges {
		loaded.button("range" + string(rangeOption))
	}
}

// TestThemeCarriesTheStateSelectors guards the dynamic-property styling:
// the states the Go code sets must exist in the theme.
func TestThemeCarriesTheStateSelectors(t *testing.T) {
	for _, selector := range []string{
		`[selected="true"]`, `[direction="up"]`, `[direction="down"]`,
		`[flash="up"]`, `[flash="down"]`, `[compact="true"]`,
		`[colorScheme="dark"]`, `[outside="true"]`, `[today="true"]`,
		`[anchor="true"]`,
	} {
		if !strings.Contains(themeStyleSheet, selector) {
			t.Errorf("theme.qss has no rule matching %s", selector)
		}
	}
}
