package ui

import (
	"testing"
	"time"
)

func TestParseSettingsForm(t *testing.T) {
	dur, size, keep, err := parseSettingsForm("15", "10", "3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dur != 15*time.Second || size != 10 || keep != 3 {
		t.Fatalf("got (%v, %d, %d), want (15s, 10, 3)", dur, size, keep)
	}
}

func TestParseSettingsFormZeroRotationAllowed(t *testing.T) {
	// Size 0 (rotation disabled) and 0 archives are valid.
	if _, size, keep, err := parseSettingsForm("30", "0", "0"); err != nil || size != 0 || keep != 0 {
		t.Fatalf("got (%d, %d, %v), want (0, 0, nil)", size, keep, err)
	}
}

func TestParseSettingsFormInvalid(t *testing.T) {
	cases := []struct {
		name                    string
		refresh, size, archives string
	}{
		{"refresh not a number", "x", "5", "3"},
		{"refresh below one", "0", "5", "3"},
		{"refresh negative", "-5", "5", "3"},
		{"size negative", "30", "-1", "3"},
		{"size not a number", "30", "abc", "3"},
		{"archives negative", "30", "5", "-2"},
		{"archives not a number", "30", "5", "n"},
		{"empty refresh", "", "5", "3"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, _, _, err := parseSettingsForm(testCase.refresh, testCase.size, testCase.archives); err == nil {
				t.Errorf("expected error for %q/%q/%q, got nil", testCase.refresh, testCase.size, testCase.archives)
			}
		})
	}
}
