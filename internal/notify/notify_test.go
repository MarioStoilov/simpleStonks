package notify

import (
	"testing"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

func TestTriggered(t *testing.T) {
	alerts := []Alert{
		{Symbol: "MU", Price: 110, Above: true},
		{Symbol: "MU", Price: 90, Above: false},
		{Symbol: "AAPL", Price: 110, Above: true},
	}

	cases := []struct {
		name          string
		symbol        string
		price         float64
		wantFired     int
		wantRemaining int
	}{
		{"between thresholds fires nothing", "MU", 100, 0, 3},
		{"exact above threshold fires", "MU", 110, 1, 2},
		{"jump past above threshold fires", "MU", 113, 1, 2},
		{"exact below threshold fires", "MU", 90, 1, 2},
		{"jump past below threshold fires", "MU", 56, 1, 2},
		{"other symbols stay untouched", "^GSPC", 110, 0, 3},
		{"only the matching symbol fires", "AAPL", 150, 1, 2},
	}
	for _, testCase := range cases {
		fired, remaining := Triggered(alerts, testCase.symbol, testCase.price)
		if len(fired) != testCase.wantFired || len(remaining) != testCase.wantRemaining {
			t.Errorf("%s: fired/remaining = %d/%d, want %d/%d",
				testCase.name, len(fired), len(remaining), testCase.wantFired, testCase.wantRemaining)
		}
	}

	// A fired alert carries its own threshold, not the live price.
	fired, _ := Triggered(alerts, "MU", 113)
	if len(fired) != 1 || fired[0].Price != 110 || !fired[0].Above {
		t.Errorf("fired = %+v, want the 110 above-alert", fired)
	}
}

func TestParseExpiry(t *testing.T) {
	valid := map[string]string{
		"":             "",
		"   ":          "",
		"2026-12-31":   "2026-12-31",
		" 2027-01-05 ": "2027-01-05",
	}
	for input, want := range valid {
		if got, ok := ParseExpiry(input); !ok || got != want {
			t.Errorf("ParseExpiry(%q) = %q/%v, want %q/true", input, got, ok, want)
		}
	}
	for _, input := range []string{"31-12-2026", "2026-13-01", "2026-02-30", "2026/12/31", "tomorrow", "2026-1-5"} {
		if got, ok := ParseExpiry(input); ok {
			t.Errorf("ParseExpiry(%q) = %q/true, want invalid", input, got)
		}
	}
}

func TestExpired(t *testing.T) {
	zone := time.FixedZone("test", 2*60*60)
	alerts := []Alert{
		{Symbol: "MU", Price: 110, Above: true},                          // never expires
		{Symbol: "MU", Price: 90, Expires: "2026-11-23"},                 // through the 23rd
		{Symbol: "AAPL", Price: 300, Above: true, Expires: "2026-11-24"}, // through the 24th
		{Symbol: "AAPL", Price: 100, Expires: "soon"},                    // unreadable: kept
	}
	cases := []struct {
		name        string
		now         time.Time
		wantExpired []float64
	}{
		{"before any expiry", time.Date(2026, 11, 22, 12, 0, 0, 0, zone), nil},
		{"last moment of the expiry day", time.Date(2026, 11, 23, 23, 59, 59, 0, zone), nil},
		{"the day after", time.Date(2026, 11, 24, 0, 0, 0, 0, zone), []float64{90}},
		{"long after", time.Date(2027, 1, 1, 9, 0, 0, 0, zone), []float64{90, 300}},
	}
	for _, testCase := range cases {
		expired, remaining := Expired(alerts, testCase.now)
		if len(expired) != len(testCase.wantExpired) || len(expired)+len(remaining) != len(alerts) {
			t.Errorf("%s: expired/remaining = %d/%d, want %d expired of %d",
				testCase.name, len(expired), len(remaining), len(testCase.wantExpired), len(alerts))
			continue
		}
		for idx, price := range testCase.wantExpired {
			if expired[idx].Price != price {
				t.Errorf("%s: expired[%d] = %+v, want the %v alert", testCase.name, idx, expired[idx], price)
			}
		}
	}
}

// TestExpiredFollowsTheLocalDay: the same instant is a different calendar
// day in different zones, and the alert follows the zone of now.
func TestExpiredFollowsTheLocalDay(t *testing.T) {
	alert := Alert{Symbol: "MU", Price: 90, Expires: "2026-11-23"}
	instant := time.Date(2026, 11, 23, 23, 30, 0, 0, time.UTC)
	if alert.ExpiredAt(instant) {
		t.Error("expired at 23:30 UTC on its expiry day")
	}
	if !alert.ExpiredAt(instant.In(time.FixedZone("east", 3*60*60))) {
		t.Error("still pending at 02:30 the next day, three hours east")
	}
}

func TestSoundHints(t *testing.T) {
	cases := map[Sound]string{
		SoundMessage:  "message-new-instant",
		SoundBell:     "bell",
		SoundAlarm:    "alarm-clock-elapsed",
		Sound("loud"): "message-new-instant", // unknown: the standard sound
		Sound(""):     "message-new-instant",
	}
	for sound, want := range cases {
		hints := sound.hints()
		name, ok := hints[constants.NotifyHintSoundName]
		if !ok || name.Value() != want || len(hints) != 1 {
			t.Errorf("%q hints = %v, want only sound-name %q", sound, hints, want)
		}
	}

	hints := SoundNone.hints()
	if suppress, ok := hints[constants.NotifyHintSuppressSound]; !ok || suppress.Value() != true || len(hints) != 1 {
		t.Errorf("none hints = %v, want only suppress-sound true", hints)
	}

	// Every menu choice but none maps to a sound event.
	for _, sound := range Sounds {
		if _, mapped := soundNames[sound]; !mapped && sound != SoundNone {
			t.Errorf("menu sound %q has no sound event", sound)
		}
	}
}

func TestSoundFor(t *testing.T) {
	if got := SoundFor(Alert{Above: true}, SoundBell, SoundAlarm); got != SoundBell {
		t.Errorf("rise alert plays %q, want the rose sound", got)
	}
	if got := SoundFor(Alert{Above: false}, SoundBell, SoundAlarm); got != SoundAlarm {
		t.Errorf("fall alert plays %q, want the fell sound", got)
	}
}
