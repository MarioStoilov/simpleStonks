//go:build manual

package notify

import (
	"testing"
	"time"

	"github.com/MarioStoilov/simplestonks/internal/constants"
)

// TestSendManual pops a real, clickable desktop notification; run by hand
// with -tags manual to verify the session-bus path end to end.
func TestSendManual(t *testing.T) {
	clicked := make(chan Alert, 1)
	OnActivate(func(alert Alert) { clicked <- alert })
	if err := SendAlert(Alert{Symbol: "TEST", Price: 123.45, Above: true}, 130, DurationLong); err != nil {
		t.Fatalf("SendAlert: %v", err)
	}
	select {
	case alert := <-clicked:
		t.Logf("notification clicked: %+v", alert)
	case <-time.After(constants.NotifyLongDuration):
		t.Log("notification not clicked (fine — click it to test activation)")
	}
}
