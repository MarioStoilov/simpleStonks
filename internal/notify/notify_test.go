package notify

import "testing"

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
