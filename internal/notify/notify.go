// Package notify owns the one-shot price alerts: the persisted alert model,
// the pure trigger check against fresh quotes, and the desktop notification a
// fired alert raises. Notifications go over the session bus using the
// freedesktop org.freedesktop.Notifications service — the standard path for
// both native runs and strict snaps (granted by the desktop interface).
package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/MarioStoilov/simplestonks/internal/assets"
	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// Alert is a one-shot price alert on a tracked symbol: it fires when the
// live price reaches or passes Price and is then removed.
type Alert struct {
	// Symbol is the tracked ticker/index the alert watches.
	Symbol string `json:"symbol"`

	// Price is the alert threshold.
	Price float64 `json:"price"`

	// Above is true when the alert was set above the market price and fires
	// at or above Price; false when set below, firing at or below it.
	Above bool `json:"above"`
}

// Duration selects how long an alert notification stays on screen.
type Duration string

const (
	DurationQuick    Duration = "quick"
	DurationModerate Duration = "moderate"
	DurationLong     Duration = "long"
)

// Durations lists the notification-duration choices in menu order.
var Durations = []Duration{DurationQuick, DurationModerate, DurationLong}

// expireMs maps the setting to the freedesktop expire_timeout in
// milliseconds; unknown values fall back to moderate.
func (duration Duration) expireMs() int32 {
	switch duration {
	case DurationQuick:
		return int32(constants.NotifyQuickDuration / time.Millisecond)
	case DurationLong:
		return int32(constants.NotifyLongDuration / time.Millisecond)
	default:
		return int32(constants.NotifyModerateDuration / time.Millisecond)
	}
}

// Triggered splits the alerts hit by a fresh price for symbol from the rest.
// An above-alert fires at or beyond its price, a below-alert at or below it —
// so a tick that jumps past the threshold still fires. Alerts on other
// symbols always stay.
func Triggered(alerts []Alert, symbol string, price float64) (fired, remaining []Alert) {
	for _, alert := range alerts {
		hit := alert.Symbol == symbol &&
			((alert.Above && price >= alert.Price) || (!alert.Above && price <= alert.Price))
		if hit {
			fired = append(fired, alert)
		} else {
			remaining = append(remaining, alert)
		}
	}
	return fired, remaining
}

// Sent notifications awaiting a click, and the activation callback; all
// guarded by pendingMu (the signal watcher runs on a bus goroutine).
var (
	pendingMu     sync.Mutex
	pendingAlerts = map[uint32]Alert{}
	onActivate    func(Alert)
	watcherOnce   sync.Once
)

// OnActivate registers the callback invoked when an alert notification is
// clicked. It runs on a D-Bus goroutine — hop to the UI thread as needed.
func OnActivate(callback func(Alert)) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	onActivate = callback
}

// SendAlert raises the clickable desktop notification for a fired alert,
// reporting its threshold and the live price that hit it, staying on screen
// per the duration setting.
func SendAlert(alert Alert, price float64, duration Duration) error {
	direction := constants.MsgAlertRose
	if !alert.Above {
		direction = constants.MsgAlertFell
	}
	id, err := send(
		fmt.Sprintf(constants.FmtAlertSummary, alert.Symbol),
		fmt.Sprintf(constants.FmtAlertBody, direction, alert.Price, price),
		true, duration.expireMs())
	if err == nil {
		pendingMu.Lock()
		pendingAlerts[id] = alert
		pendingMu.Unlock()
	}
	return err
}

// Send shows a plain, non-clickable desktop notification with the desktop's
// default timeout. It is safe to call from any goroutine.
func Send(summary, body string) error {
	_, err := send(summary, body, false, int32(constants.NotifyExpireDefault))
	return err
}

// send delivers a notification; clickable ones carry the default action and
// have the signal watcher running so clicks reach the OnActivate callback.
func send(summary, body string, clickable bool, expireMs int32) (uint32, error) {
	// SessionBus returns a shared connection owned by the dbus package, so
	// it must not be closed here.
	conn, err := dbus.SessionBus()
	if err != nil {
		return 0, err
	}
	actions := []string{}
	if clickable {
		actions = []string{constants.NotifyActionDefault, constants.LabelNotifyOpen}
		watchActivations(conn)
	}
	// The sound-name hint plays the desktop's standard message sound, and
	// desktop-entry ties the notification to the app — desktops keep such
	// notifications in the notification history and offer per-app settings.
	hints := map[string]dbus.Variant{
		constants.NotifyHintSoundName:    dbus.MakeVariant(constants.NotifySoundName),
		constants.NotifyHintDesktopEntry: dbus.MakeVariant(desktopEntry()),
	}
	service := conn.Object(constants.NotifyDBusService, dbus.ObjectPath(constants.NotifyDBusPath))
	var id uint32
	err = service.Call(constants.NotifyDBusMethod, 0,
		constants.AppName, // app_name
		uint32(0),         // replaces_id: always a new notification
		iconPath(),        // app_icon: the materialized app logo
		summary,           // summary
		body,              // body
		actions,           // actions
		hints,             // hints
		expireMs,          // expire_timeout
	).Store(&id)
	return id, err
}

// desktopEntry names the desktop file owning our notifications: snapd
// installs snap desktop entries as <snap>_<app>, and native runs use the
// plain app name.
func desktopEntry() string {
	if snapName := os.Getenv(constants.EnvSnapName); snapName != "" {
		return fmt.Sprintf(constants.FmtSnapDesktopEntry, snapName, snapName)
	}
	return constants.AppDirName
}

// watchActivations subscribes (once) to the notification service's signals:
// a click on one of our notifications hands its alert to the OnActivate
// callback; closed/expired ones are forgotten.
func watchActivations(conn *dbus.Conn) {
	watcherOnce.Do(func() {
		for _, member := range []string{constants.NotifyMemberActionInvoked, constants.NotifyMemberClosed} {
			if err := conn.AddMatchSignal(
				dbus.WithMatchInterface(constants.NotifyDBusService),
				dbus.WithMatchMember(member),
			); err != nil {
				return // clicks degrade to no-ops; the notifications still show
			}
		}
		signals := make(chan *dbus.Signal, constants.NotifySignalBuffer)
		conn.Signal(signals)
		go func() {
			for signal := range signals {
				if len(signal.Body) == 0 {
					continue
				}
				id, ok := signal.Body[0].(uint32)
				if !ok {
					continue
				}
				pendingMu.Lock()
				alert, known := pendingAlerts[id]
				callback := onActivate
				delete(pendingAlerts, id)
				pendingMu.Unlock()
				if signal.Name == constants.NotifySignalActionInvoked && known && callback != nil {
					callback(alert)
				}
			}
		}()
	})
}

// iconPath materializes the embedded app logo into the user cache once and
// returns its absolute path for the notification icon. The notification
// daemon reads the file itself, so it must live on disk; on failure the
// path is empty and notifications simply show without an icon.
var iconPath = sync.OnceValue(func() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(cacheDir, constants.AppDirName, constants.NotifyIconFileName)
	if err := os.MkdirAll(filepath.Dir(path), constants.DirPerm); err != nil {
		return ""
	}
	if err := os.WriteFile(path, assets.IconSVG, constants.FilePerm); err != nil {
		return ""
	}
	return path
})
