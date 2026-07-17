package ui

import "testing"

func TestFlashDirection(t *testing.T) {
	if flash, _ := flashDirection(false, 10, 11); flash {
		t.Error("must not flash without a previous price")
	}
	if flash, _ := flashDirection(true, 10, 10); flash {
		t.Error("must not flash when the price is unchanged")
	}
	if flash, rising := flashDirection(true, 10, 11); !flash || !rising {
		t.Errorf("rise: flash=%v rising=%v, want true/true", flash, rising)
	}
	if flash, rising := flashDirection(true, 11, 10); !flash || rising {
		t.Errorf("drop: flash=%v rising=%v, want true/false", flash, rising)
	}
}
