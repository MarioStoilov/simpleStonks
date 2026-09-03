package qtui

import (
	"embed"
	"fmt"
	"unsafe"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/uitools"
)

// The widget trees live in Qt Designer forms under ui/, loaded at run time
// by QUiLoader (Qt's own loader, so every Designer property applies:
// text, tooltips, cursors, sizes, stretch factors, promoted widgets). The
// looks live in ui/theme.qss, installed application-wide; state-dependent
// looks key off dynamic properties set with setState. The Go side only
// loads a form, looks widgets up by object name, wires behaviour, and
// fills in data.

//go:embed ui/*.ui
var formFiles embed.FS

//go:embed ui/theme.qss
var themeStyleSheet string

// Form file names.
const (
	tileForm        = "tile.ui"
	sidebarTileForm = "sidebar_tile.ui"
	homeForm        = "home.ui"
	detailForm      = "detail.ui"
	cardForm        = "card.ui"
	alertPillForm   = "alert_pill.ui"
	cardDialogForm  = "dialogcard.ui"
	confirmForm     = "confirm.ui"
	aboutForm       = "about.ui"
	alertForm       = "alert.ui"
	previewForm     = "preview.ui"
	searchForm      = "search.ui"
	resultRowForm   = "resultrow.ui"
	settingsForm    = "settings.ui"
)

// chartClassName is the promoted widget class the forms use for the
// custom-painted chart.
const chartClassName = "ChartWidget"

// formLoader is the process-wide loader (UI thread only). goWidgets maps
// every widget it built to its Go wrapper: miqt only allows virtual-method
// overrides (event handlers) on widgets constructed from Go, so the loader
// builds the classes the forms use itself instead of letting Qt do it, and
// the accessors hand back those same wrappers.
var (
	formLoader *uitools.QUiLoader
	goWidgets  = map[unsafe.Pointer]any{}
)

func loader() *uitools.QUiLoader {
	if formLoader != nil {
		return formLoader
	}
	formLoader = uitools.NewQUiLoader()
	formLoader.OnCreateWidget(func(super func(className string, parent *qt.QWidget, name string) *qt.QWidget, className string, parent *qt.QWidget, name string) *qt.QWidget {
		var wrapper any
		var widget *qt.QWidget
		switch className {
		case "QWidget":
			plain := qt.NewQWidget(parent)
			wrapper, widget = plain, plain
		case "QFrame":
			frame := qt.NewQFrame(parent)
			wrapper, widget = frame, frame.QWidget
		case "QLabel":
			label := qt.NewQLabel(parent)
			wrapper, widget = label, label.QWidget
		case "QPushButton":
			button := qt.NewQPushButton(parent)
			wrapper, widget = button, button.QWidget
		case "QScrollArea":
			scroll := qt.NewQScrollArea(parent)
			// Qt resolves a scrollbar's sub-controls against the nearest
			// styled ancestor: with the theme only on the window card, the
			// track paints opaque instead of letting the translucent window
			// through. Giving the scroll area the theme keeps it clear.
			scroll.SetStyleSheet(themeStyleSheet)
			wrapper, widget = scroll, scroll.QWidget
		case "QCheckBox":
			box := qt.NewQCheckBox(parent)
			wrapper, widget = box, box.QWidget
		case "QComboBox":
			box := qt.NewQComboBox(parent)
			wrapper, widget = box, box.QWidget
		case "QLineEdit":
			edit := qt.NewQLineEdit(parent)
			wrapper, widget = edit, edit.QWidget
		case "QSlider":
			slider := qt.NewQSlider(parent)
			wrapper, widget = slider, slider.QWidget
		case "QStackedWidget":
			stack := qt.NewQStackedWidget(parent)
			wrapper, widget = stack, stack.QWidget
		case "QListWidget":
			list := qt.NewQListWidget(parent)
			wrapper, widget = list, list.QWidget
		case chartClassName:
			chart := newChartWidget(parent)
			wrapper, widget = chart, chart.QWidget
		default:
			return super(className, parent, name)
		}
		widget.SetObjectName(*qt.NewQAnyStringView3(name))
		goWidgets[widget.UnsafePointer()] = wrapper
		return widget
	})
	return formLoader
}

// form is a loaded Designer form: its root widget and every named object
// in it.
type form struct {
	root    *qt.QWidget
	objects map[string]*qt.QObject
}

// loadForm instantiates an embedded form. A missing file, a form that does
// not load, or a later lookup of a name the form lacks is a programming
// error and panics.
func loadForm(name string) *form {
	data, err := formFiles.ReadFile("ui/" + name)
	if err != nil {
		panic(fmt.Sprintf("form %s: %v", name, err))
	}
	buffer := qt.NewQBuffer()
	buffer.SetData(data)
	buffer.Open(qt.QIODeviceBase__ReadOnly)
	root := loader().Load(buffer.QIODevice)
	buffer.Delete()
	if root == nil {
		panic(fmt.Sprintf("form %s did not load", name))
	}
	loaded := &form{root: root, objects: map[string]*qt.QObject{}}
	loaded.index(root.QObject)
	return loaded
}

func (loaded *form) index(object *qt.QObject) {
	if name := object.ObjectName(); name != "" {
		loaded.objects[name] = object
	}
	for _, child := range object.Children() {
		loaded.index(child)
	}
}

// object returns the Go wrapper of a named widget the loader built.
func (loaded *form) object(name string) any {
	object, ok := loaded.objects[name]
	if !ok {
		panic(fmt.Sprintf("form has no object named %q", name))
	}
	wrapper, ok := goWidgets[object.UnsafePointer()]
	if !ok {
		panic(fmt.Sprintf("form object %q is not a loader-built widget", name))
	}
	return wrapper
}

// typed resolves a named widget to the expected wrapper type.
func typed[T any](loaded *form, name string) T {
	wrapper, ok := loaded.object(name).(T)
	if !ok {
		panic(fmt.Sprintf("form object %q is a %T, not a %T", name, loaded.object(name), wrapper))
	}
	return wrapper
}

func (loaded *form) widget(name string) *qt.QWidget     { return typed[*qt.QWidget](loaded, name) }
func (loaded *form) frame(name string) *qt.QFrame       { return typed[*qt.QFrame](loaded, name) }
func (loaded *form) label(name string) *qt.QLabel       { return typed[*qt.QLabel](loaded, name) }
func (loaded *form) button(name string) *qt.QPushButton { return typed[*qt.QPushButton](loaded, name) }
func (loaded *form) chart(name string) *chartWidget     { return typed[*chartWidget](loaded, name) }
func (loaded *form) scrollArea(name string) *qt.QScrollArea {
	return typed[*qt.QScrollArea](loaded, name)
}
func (loaded *form) checkBox(name string) *qt.QCheckBox { return typed[*qt.QCheckBox](loaded, name) }
func (loaded *form) comboBox(name string) *qt.QComboBox { return typed[*qt.QComboBox](loaded, name) }
func (loaded *form) lineEdit(name string) *qt.QLineEdit { return typed[*qt.QLineEdit](loaded, name) }
func (loaded *form) slider(name string) *qt.QSlider     { return typed[*qt.QSlider](loaded, name) }
func (loaded *form) listWidget(name string) *qt.QListWidget {
	return typed[*qt.QListWidget](loaded, name)
}
func (loaded *form) stack(name string) *qt.QStackedWidget {
	return typed[*qt.QStackedWidget](loaded, name)
}

// The layout accessors wrap layouts the loader built. Layouts need no Go
// construction (nothing overrides their virtuals), so they are wrapped
// straight from the object tree. Views use them to add the widgets that
// only exist at run time: tiles, alert pills.
func (loaded *form) layoutObject(name string) unsafe.Pointer {
	object, ok := loaded.objects[name]
	if !ok {
		panic(fmt.Sprintf("form has no object named %q", name))
	}
	return object.UnsafePointer()
}

func (loaded *form) grid(name string) *qt.QGridLayout {
	return qt.UnsafeNewQGridLayout(loaded.layoutObject(name))
}
func (loaded *form) vbox(name string) *qt.QVBoxLayout {
	return qt.UnsafeNewQVBoxLayout(loaded.layoutObject(name))
}
func (loaded *form) hbox(name string) *qt.QHBoxLayout {
	return qt.UnsafeNewQHBoxLayout(loaded.layoutObject(name))
}

// setState sets a dynamic property the theme's selectors key off (e.g.
// direction="up") and repolishes the widget so the new look applies.
func setState(widget *qt.QWidget, property, value string) {
	widget.SetProperty(property, qt.NewQVariant14(value))
	style := widget.Style()
	style.Unpolish(widget)
	style.Polish(widget)
}
