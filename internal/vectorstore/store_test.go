package vectorstore

import "testing"

func TestParseChron(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"Present", 2147483647},
		{"present", 2147483647},
		{"", 2147483647},
		{"Dec. 2025", 2025*12 + 12},
		{"Jan 2024", 2024*12 + 1},
		{"2024", 2024*12 + 12},
		{"Sept. 2023", 2023*12 + 9},
		{"May 2020", 2020*12 + 5},
	}
	for _, tc := range tests {
		got := parseChron(tc.input)
		if got != tc.want {
			t.Errorf("parseChron(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestChronAfter_PresentVsDated(t *testing.T) {
	if !chronAfter("Present", "Dec. 2025") {
		t.Error("Present should sort after Dec. 2025")
	}
	if !chronAfter("Present", "Jan 2020") {
		t.Error("Present should sort after Jan 2020")
	}
	if chronAfter("Jan 2020", "Present") {
		t.Error("Jan 2020 should NOT sort after Present")
	}
}

func TestChronAfter_MonthOrdering(t *testing.T) {
	if !chronAfter("Dec. 2025", "Jan 2025") {
		t.Error("Dec. 2025 should sort after Jan 2025")
	}
	if chronAfter("Jan 2025", "Dec. 2025") {
		t.Error("Jan 2025 should NOT sort after Dec. 2025")
	}
}

func TestChronAfter_SameEndDifferentStart(t *testing.T) {
	if !chronAfter("May 2025", "Jan 2025") {
		t.Error("May 2025 should sort after Jan 2025 (same year)")
	}
}
