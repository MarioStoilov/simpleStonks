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
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/MarioStoilov/simplestonks/internal/assets"
	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// Alert is a one-shot price alert on a tracked symbol: it fires when the
// live price reaches or passes Price and is then removed. An alert with an
// expiry date that never fired is dropped once that day is over.
type Alert struct {
	// Symbol is the tracked ticker/index the alert watches.
	Symbol string `json:"symbol"`

	// Price is the alert threshold.
	Price float64 `json:"price"`

	// Above is true when the alert was set above the market price and fires
	// at or above Price; false when set below, firing at or below it.
	Above bool `json:"above"`

	// Expires is the optional last day the alert stays pending, as a local
	// calendar date (constants.DateFmtISO, e.g. "2026-12-31"); empty means
	// it never expires.
	Expires string `json:"expires,omitempty"`
}

// ParseExpiry validates an expiry-date entry and returns it in the stored
// form. Empty input is valid and means no expiry.
func ParseExpiry(text string) (expires string, ok bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", true
	}
	day, err := time.Parse(constants.DateFmtISO, trimmed)
	if err != nil {
		return "", false
	}
	return day.Format(constants.DateFmtISO), true
}

// ExpiredAt reports whether the alert's expiry day is over at now (in now's
// time zone): the alert stays live through the whole of its expiry date. An
// alert without an expiry — or with an unreadable one, e.g. from a hand
// edit of the config — never expires.
func (alert Alert) ExpiredAt(now time.Time) bool {
	if alert.Expires == "" {
		return false
	}
	if _, err := time.Parse(constants.DateFmtISO, alert.Expires); err != nil {
		return false
	}
	// ISO dates order like the days they name.
	return now.Format(constants.DateFmtISO) > alert.Expires
}

// Expired splits the alerts whose expiry day is over at now from the rest.
func Expired(alerts []Alert, now time.Time) (expired, remaining []Alert) {
	for _, alert := range alerts {
		if alert.ExpiredAt(now) {
			expired = append(expired, alert)
		} else {
			remaining = append(remaining, alert)
		}
	}
	return expired, remaining
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

// Sound selects what an alert notification plays.
type Sound string

const (
	SoundMessage     Sound = "message"
	SoundBell        Sound = "bell"
	SoundComplete    Sound = "complete"
	SoundInformation Sound = "information"
	SoundWarning     Sound = "warning"
	SoundAlarm       Sound = "alarm"
	SoundNone        Sound = "none"
)

// Sounds lists the notification-sound choices in menu order.
var Sounds = []Sound{
	SoundMessage, SoundBell, SoundComplete, SoundInformation, SoundWarning, SoundAlarm, SoundNone,
}

// soundNames maps each audible choice to its event in the XDG sound naming
// specification (a protocol mapping); the desktop's sound theme supplies
// the audio, and every name here is part of the base freedesktop theme all
// others fall back to.
var soundNames = map[Sound]string{
	SoundMessage:     constants.NotifySoundName,
	SoundBell:        "bell",
	SoundComplete:    "complete",
	SoundInformation: "dialog-information",
	SoundWarning:     "dialog-warning",
	SoundAlarm:       "alarm-clock-elapsed",
}

// hints returns the freedesktop notification hints that make the desktop
// play the sound: its XDG event name, or the suppress-sound flag for
// SoundNone. Unknown values (e.g. from a hand-edited config) play the
// standard message sound.
func (sound Sound) hints() map[string]dbus.Variant {
	if sound == SoundNone {
		return map[string]dbus.Variant{constants.NotifyHintSuppressSound: dbus.MakeVariant(true)}
	}
	name, known := soundNames[sound]
	if !known {
		name = constants.NotifySoundName
	}
	return map[string]dbus.Variant{constants.NotifyHintSoundName: dbus.MakeVariant(name)}
}

// SoundFor picks the sound of a fired alert by its direction: rose for an
// alert the price rose up to, fell for one it fell down to.
func SoundFor(alert Alert, rose, fell Sound) Sound {
	if alert.Above {
		return rose
	}
	return fell
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
// per the duration setting and playing sound.
func SendAlert(alert Alert, price float64, duration Duration, sound Sound) error {
	direction := constants.MsgAlertRose
	if !alert.Above {
		direction = constants.MsgAlertFell
	}
	id, err := send(
		fmt.Sprintf(constants.FmtAlertSummary, alert.Symbol),
		fmt.Sprintf(constants.FmtAlertBody, direction, alert.Price, price),
		true, duration.expireMs(), sound)
	if err == nil {
		pendingMu.Lock()
		pendingAlerts[id] = alert
		pendingMu.Unlock()
	}
	return err
}

// SendSoundTest shows a plain, non-clickable sample notification playing
// sound, so a sound choice can be heard before it is saved. It is safe to
// call from any goroutine.
func SendSoundTest(sound Sound, duration Duration) error {
	_, err := send(constants.TitleSoundTest, fmt.Sprintf(constants.FmtSoundTestBody, sound),
		false, duration.expireMs(), sound)
	return err
}

// send delivers a notification; clickable ones carry the default action and
// have the signal watcher running so clicks reach the OnActivate callback.
func send(summary, body string, clickable bool, expireMs int32, sound Sound) (uint32, error) {
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
	// The sound hint plays (or silences) the chosen sound, and desktop-entry
	// ties the notification to the app — desktops keep such notifications in
	// the notification history and offer per-app settings.
	hints := sound.hints()
	hints[constants.NotifyHintDesktopEntry] = dbus.MakeVariant(desktopEntry())
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
