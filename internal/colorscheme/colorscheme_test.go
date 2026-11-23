package colorscheme

import (
	"testing"

	"github.com/godbus/dbus/v5"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

func TestFromPortal(t *testing.T) {
	cases := []struct {
		name  string
		value any
		want  Scheme
	}{
		{"no preference", uint32(0), Unknown},
		{"prefer dark", uint32(1), Dark},
		{"prefer light", uint32(2), Light},
		{"future value", uint32(7), Unknown},
		{"wrong type", "dark", Unknown},
		{"single variant", dbus.MakeVariant(uint32(1)), Dark},
		{"nested variant", dbus.MakeVariant(dbus.MakeVariant(uint32(2))), Light},
	}
	for _, testCase := range cases {
		if got := fromPortal(testCase.value); got != testCase.want {
			t.Errorf("%s: fromPortal = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}

func TestFromSignal(t *testing.T) {
	changed := func(namespace, key string, value any) *dbus.Signal {
		return &dbus.Signal{
			Name: constants.PortalSignalSettingChanged,
			Body: []any{namespace, key, dbus.MakeVariant(value)},
		}
	}
	scheme, ok := fromSignal(changed(constants.PortalAppearanceNamespace, constants.PortalColorSchemeKey, uint32(1)))
	if !ok || scheme != Dark {
		t.Errorf("color-scheme change = %v, %v; want Dark, true", scheme, ok)
	}
	if _, ok := fromSignal(changed(constants.PortalAppearanceNamespace, "accent-color", uint32(1))); ok {
		t.Error("another appearance key was taken for a color-scheme change")
	}
	if _, ok := fromSignal(changed("org.gnome.desktop.interface", constants.PortalColorSchemeKey, "x")); ok {
		t.Error("another namespace was taken for a color-scheme change")
	}
	// The shared bus connection also delivers the notification signals.
	foreign := &dbus.Signal{Name: constants.NotifySignalActionInvoked, Body: []any{uint32(3), "default"}}
	if _, ok := fromSignal(foreign); ok {
		t.Error("a notification signal was taken for a color-scheme change")
	}
	if _, ok := fromSignal(nil); ok {
		t.Error("a nil signal was taken for a color-scheme change")
	}
}
