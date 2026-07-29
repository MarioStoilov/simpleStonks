package qtui

import "testing"

func TestOfflineNow(t *testing.T) {
	cases := []struct {
		name    string
		results map[string]bool
		want    bool
	}{
		{"no results yet", map[string]bool{}, false},
		{"all succeeding", map[string]bool{"MU": true, "AAPL": true}, false},
		{"one bad ticker keeps the app online", map[string]bool{"MU": true, "BAD": false}, false},
		{"all failing means offline", map[string]bool{"MU": false, "AAPL": false}, true},
		{"single failing symbol counts", map[string]bool{"MU": false}, true},
	}
	for _, testCase := range cases {
		if got := offlineNow(testCase.results); got != testCase.want {
			t.Errorf("%s: offlineNow = %v, want %v", testCase.name, got, testCase.want)
		}
	}
}
