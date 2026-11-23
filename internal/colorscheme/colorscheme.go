// Package colorscheme reports the desktop's light/dark color-scheme
// preference and its changes, read from the freedesktop settings portal
// (org.freedesktop.appearance color-scheme) on the session bus — the
// standard source for both native runs and strict snaps. Desktops without
// the portal, or with no stated preference, report Unknown and the caller
// falls back to its own heuristic.
package colorscheme

import (
	"context"
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// Scheme is the desktop's color-scheme preference.
type Scheme int

const (
	// Unknown means the desktop states no preference (or cannot be asked).
	Unknown Scheme = iota
	Light
	Dark
)

// The portal's color-scheme values (a protocol mapping, per the
// org.freedesktop.appearance specification).
const (
	portalPreferDark  uint32 = 1
	portalPreferLight uint32 = 2
)

// fromPortal maps a portal color-scheme value to a Scheme. The portal wraps
// the value in one or more variants depending on the method used; anything
// that is not a known number is Unknown.
func fromPortal(value any) Scheme {
	for {
		variant, wrapped := value.(dbus.Variant)
		if !wrapped {
			break
		}
		value = variant.Value()
	}
	number, ok := value.(uint32)
	if !ok {
		return Unknown
	}
	switch number {
	case portalPreferDark:
		return Dark
	case portalPreferLight:
		return Light
	default:
		return Unknown
	}
}

// fromSignal extracts the scheme from a portal SettingChanged signal;
// ok is false for any other signal or setting.
func fromSignal(signal *dbus.Signal) (scheme Scheme, ok bool) {
	if signal == nil || signal.Name != constants.PortalSignalSettingChanged ||
		len(signal.Body) < constants.PortalSettingChangedArgs {
		return Unknown, false
	}
	namespace, _ := signal.Body[0].(string)
	key, _ := signal.Body[1].(string)
	if namespace != constants.PortalAppearanceNamespace || key != constants.PortalColorSchemeKey {
		return Unknown, false
	}
	return fromPortal(signal.Body[2]), true
}

// Read asks the settings portal for the current color scheme. An error means
// the portal could not be asked; the scheme is then Unknown.
func Read() (Scheme, error) {
	// SessionBus returns a shared connection owned by the dbus package, so
	// it must not be closed here.
	conn, err := dbus.SessionBus()
	if err != nil {
		return Unknown, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.PortalReadTimeout)
	defer cancel()
	var value dbus.Variant
	err = conn.Object(constants.PortalDBusService, dbus.ObjectPath(constants.PortalDBusPath)).
		CallWithContext(ctx, constants.PortalMethodRead, 0,
			constants.PortalAppearanceNamespace, constants.PortalColorSchemeKey).
		Store(&value)
	if err != nil {
		return Unknown, fmt.Errorf("reading the portal color scheme: %w", err)
	}
	return fromPortal(value), nil
}

var watchOnce sync.Once

// Watch subscribes (once per process) to the portal's color-scheme changes;
// onChange runs on a D-Bus goroutine — hop to the UI thread as needed.
func Watch(onChange func(Scheme)) error {
	conn, err := dbus.SessionBus()
	if err != nil {
		return err
	}
	watchOnce.Do(func() {
		err = conn.AddMatchSignal(
			dbus.WithMatchInterface(constants.PortalSettingsInterface),
			dbus.WithMatchMember(constants.PortalMemberSettingChanged),
		)
		if err != nil {
			return
		}
		signals := make(chan *dbus.Signal, constants.PortalSignalBuffer)
		conn.Signal(signals)
		go func() {
			// The shared connection delivers every subscribed signal here,
			// including the notification service's; fromSignal filters.
			for signal := range signals {
				if scheme, ok := fromSignal(signal); ok {
					onChange(scheme)
				}
			}
		}()
	})
	return err
}
