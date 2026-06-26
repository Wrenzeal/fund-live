package main

import "testing"

func TestNormalizeHistoryDays(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "default", in: 0, want: 30},
		{name: "negative", in: -7, want: 30},
		{name: "passthrough", in: 15, want: 15},
		{name: "cap", in: 365, want: 180},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeHistoryDays(tt.in); got != tt.want {
				t.Fatalf("normalizeHistoryDays(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestUniqueFundCodesTrimsDedupesAndSorts(t *testing.T) {
	got := uniqueFundCodes([]string{"005827", " 003095 ", "", "005827", "320007"})
	want := []string{"003095", "005827", "320007"}
	if len(got) != len(want) {
		t.Fatalf("got = %+v, want %+v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("got = %+v, want %+v", got, want)
		}
	}
}
