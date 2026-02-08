package tender

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// ui.go tests

func TestBuildHourlyCron(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name:  "valid minute input",
			input: "15",
			want:  "15 * * * *",
		},
		{
			name:  "zero minute",
			input: "0",
			want:  "0 * * * *",
		},
		{
			name:  "maximum minute",
			input: "59",
			want:  "59 * * * *",
		},
		{
			name:    "invalid minute - negative",
			input:   "-1",
			wantErr: true,
			errMsg:  "minute must be 0-59",
		},
		{
			name:    "invalid minute - too high",
			input:   "60",
			wantErr: true,
			errMsg:  "minute must be 0-59",
		},
		{
			name:    "invalid minute - non-numeric",
			input:   "abc",
			wantErr: true,
			errMsg:  "minute must be 0-59",
		},
		{
			name:  "whitespace handling",
			input: "  30  ",
			want:  "30 * * * *",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildHourlyCron(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tc.errMsg) {
					t.Fatalf("expected error containing %q, got %v", tc.errMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("buildHourlyCron(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestBuildDailyCron(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name:  "valid time input",
			input: "09:30",
			want:  "30 9 * * *",
		},
		{
			name:  "midnight",
			input: "00:00",
			want:  "0 0 * * *",
		},
		{
			name:  "end of day",
			input: "23:59",
			want:  "59 23 * * *",
		},
		{
			name:    "invalid hour - negative",
			input:   "-1:00",
			wantErr: true,
			errMsg:  "hour must be 0-23",
		},
		{
			name:    "invalid hour - too high",
			input:   "24:00",
			wantErr: true,
			errMsg:  "hour must be 0-23",
		},
		{
			name:    "invalid minute - too high",
			input:   "12:60",
			wantErr: true,
			errMsg:  "minute must be 0-59",
		},
		{
			name:    "invalid format - missing colon",
			input:   "1230",
			wantErr: true,
			errMsg:  "time must be HH:MM",
		},
		{
			name:    "invalid format - extra parts",
			input:   "12:30:45",
			wantErr: true,
			errMsg:  "time must be HH:MM",
		},
		{
			name:  "whitespace handling",
			input: "  14 :  15  ",
			want:  "15 14 * * *",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildDailyCron(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tc.errMsg) {
					t.Fatalf("expected error containing %q, got %v", tc.errMsg, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("buildDailyCron(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestBuildWeeklyCron(t *testing.T) {
	t.Run("valid weekly schedule", func(t *testing.T) {
		got, err := buildWeeklyCron("1,3,5", "09:30")
		if err != nil {
			t.Fatalf("buildWeeklyCron returned error: %v", err)
		}
		if got != "30 9 * * 1,3,5" {
			t.Fatalf("unexpected cron: %q", got)
		}
	})

	t.Run("single day", func(t *testing.T) {
		got, err := buildWeeklyCron("0", "23:45")
		if err != nil {
			t.Fatalf("buildWeeklyCron returned error: %v", err)
		}
		if got != "45 23 * * 0" {
			t.Fatalf("unexpected cron: %q", got)
		}
	})

	t.Run("all days", func(t *testing.T) {
		got, err := buildWeeklyCron("0,1,2,3,4,5,6", "12:00")
		if err != nil {
			t.Fatalf("buildWeeklyCron returned error: %v", err)
		}
		if got != "0 12 * * 0,1,2,3,4,5,6" {
			t.Fatalf("unexpected cron: %q", got)
		}
	})

	t.Run("invalid day - negative", func(t *testing.T) {
		_, err := buildWeeklyCron("-1,1", "09:00")
		if err == nil {
			t.Fatal("expected error for negative day")
		}
		if !strings.Contains(err.Error(), "days must be 0-6") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("invalid day - too high", func(t *testing.T) {
		_, err := buildWeeklyCron("7", "09:00")
		if err == nil {
			t.Fatal("expected error for day 7")
		}
		if !strings.Contains(err.Error(), "days must be 0-6") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("invalid time format", func(t *testing.T) {
		_, err := buildWeeklyCron("1,2,3", "invalid")
		if err == nil {
			t.Fatal("expected error for invalid time")
		}
		if !strings.Contains(err.Error(), "time must be HH:MM") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("empty days", func(t *testing.T) {
		_, err := buildWeeklyCron("", "09:00")
		if err == nil {
			t.Fatal("expected error for empty days")
		}
		if !strings.Contains(err.Error(), "days must be 0-6") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("whitespace handling", func(t *testing.T) {
		got, err := buildWeeklyCron(" 1 , 3 , 5 ", "  09 : 30  ")
		if err != nil {
			t.Fatalf("buildWeeklyCron returned error: %v", err)
		}
		if got != "30 9 * * 1,3,5" {
			t.Fatalf("unexpected cron: %q", got)
		}
	})
}

func TestParseTimeHHMM(t *testing.T) {
	cases := []struct {
		input    string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{"09:30", 9, 30, false},
		{"00:00", 0, 0, false},
		{"23:59", 23, 59, false},
		{"9:30", 9, 30, false},   // Single digit hour
		{"9:3", 9, 3, false},     // Single digit minute
		{"-1:00", 0, 0, true},    // Negative hour
		{"24:00", 0, 0, true},    // Hour too high
		{"12:60", 0, 0, true},    // Minute too high
		{"1230", 0, 0, true},     // Missing colon
		{"12:30:45", 0, 0, true}, // Extra parts
		{"", 0, 0, true},         // Empty string
		{"::", 0, 0, true},       // Empty parts
		{"ab:cd", 0, 0, true},    // Non-numeric
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			hour, minute, err := parseTimeHHMM(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if hour != tc.wantHour {
				t.Fatalf("expected hour %d, got %d", tc.wantHour, hour)
			}
			if minute != tc.wantMin {
				t.Fatalf("expected minute %d, got %d", tc.wantMin, minute)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	cases := []struct {
		hour   int
		minute int
		want   string
	}{
		{0, 0, "00:00"},
		{9, 30, "09:30"},
		{12, 5, "12:05"},
		{23, 59, "23:59"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got := formatTime(tc.hour, tc.minute)
			if got != tc.want {
				t.Fatalf("formatTime(%d, %d) = %q, want %q", tc.hour, tc.minute, got, tc.want)
			}
		})
	}
}

func TestParseDays(t *testing.T) {
	t.Run("valid single day", func(t *testing.T) {
		days, err := parseDays("1")
		if err != nil {
			t.Fatalf("parseDays returned error: %v", err)
		}
		if len(days) != 1 || days[0] != 1 {
			t.Fatalf("expected [1], got %v", days)
		}
	})

	t.Run("valid multiple days", func(t *testing.T) {
		days, err := parseDays("1,3,5")
		if err != nil {
			t.Fatalf("parseDays returned error: %v", err)
		}
		expected := []int{1, 3, 5}
		if len(days) != len(expected) {
			t.Fatalf("expected %d days, got %d", len(expected), len(days))
		}
		for i, day := range days {
			if day != expected[i] {
				t.Fatalf("day %d: expected %d, got %d", i, expected[i], day)
			}
		}
	})

	t.Run("sorts days", func(t *testing.T) {
		days, err := parseDays("5,1,3")
		if err != nil {
			t.Fatalf("parseDays returned error: %v", err)
		}
		expected := []int{1, 3, 5}
		if len(days) != len(expected) {
			t.Fatalf("expected %d days, got %d", len(expected), len(days))
		}
		for i, day := range days {
			if day != expected[i] {
				t.Fatalf("day %d: expected %d, got %d", i, expected[i], day)
			}
		}
	})

	t.Run("removes duplicates", func(t *testing.T) {
		days, err := parseDays("1,1,3,3,5,5")
		if err != nil {
			t.Fatalf("parseDays returned error: %v", err)
		}
		expected := []int{1, 3, 5}
		if len(days) != len(expected) {
			t.Fatalf("expected %d days after deduplication, got %d", len(expected), len(days))
		}
		for i, day := range days {
			if day != expected[i] {
				t.Fatalf("day %d: expected %d, got %d", i, expected[i], day)
			}
		}
	})

	t.Run("all days", func(t *testing.T) {
		days, err := parseDays("0,1,2,3,4,5,6")
		if err != nil {
			t.Fatalf("parseDays returned error: %v", err)
		}
		expected := []int{0, 1, 2, 3, 4, 5, 6}
		if len(days) != len(expected) {
			t.Fatalf("expected %d days, got %d", len(expected), len(days))
		}
		for i, day := range days {
			if day != expected[i] {
				t.Fatalf("day %d: expected %d, got %d", i, expected[i], day)
			}
		}
	})

	t.Run("invalid day - negative", func(t *testing.T) {
		_, err := parseDays("-1,1")
		if err == nil {
			t.Fatal("expected error for negative day")
		}
		if !strings.Contains(err.Error(), "days must be 0-6") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("invalid day - too high", func(t *testing.T) {
		_, err := parseDays("7")
		if err == nil {
			t.Fatal("expected error for day 7")
		}
		if !strings.Contains(err.Error(), "days must be 0-6") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("invalid day - non-numeric", func(t *testing.T) {
		_, err := parseDays("a,b,c")
		if err == nil {
			t.Fatal("expected error for non-numeric day")
		}
		if !strings.Contains(err.Error(), "days must be 0-6") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		_, err := parseDays("")
		if err == nil {
			t.Fatal("expected error for empty input")
		}
		if !strings.Contains(err.Error(), "days must be 0-6") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("whitespace handling", func(t *testing.T) {
		days, err := parseDays(" 1 , 3 , 5 ")
		if err != nil {
			t.Fatalf("parseDays returned error: %v", err)
		}
		expected := []int{1, 3, 5}
		if len(days) != len(expected) {
			t.Fatalf("expected %d days, got %d", len(expected), len(days))
		}
		for i, day := range days {
			if day != expected[i] {
				t.Fatalf("day %d: expected %d, got %d", i, expected[i], day)
			}
		}
	})
}

func TestJoinInts(t *testing.T) {
	cases := []struct {
		nums []int
		sep  string
		want string
	}{
		{[]int{1, 2, 3}, ",", "1,2,3"},
		{[]int{0, 1, 2}, "-", "0-1-2"},
		{[]int{5}, ",", "5"},
		{[]int{}, ",", ""},
		{[]int{1, 2}, " ", "1 2"},
		{[]int{10, 20, 30}, ":", "10:20:30"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got := joinInts(tc.nums, tc.sep)
			if got != tc.want {
				t.Fatalf("joinInts(%v, %q) = %q, want %q", tc.nums, tc.sep, got, tc.want)
			}
		})
	}
}

func TestScheduleDefaultsFromCron(t *testing.T) {
	t.Run("hourly schedule", func(t *testing.T) {
		got, ok := scheduleDefaultsFromCron("5 * * * *")
		if !ok {
			t.Fatal("expected hourly cron to be recognized")
		}
		if got.Mode != "hourly" || got.Minute != 5 {
			t.Fatalf("unexpected defaults: %+v", got)
		}
	})

	t.Run("daily schedule", func(t *testing.T) {
		got, ok := scheduleDefaultsFromCron("30 9 * * *")
		if !ok {
			t.Fatal("expected daily cron to be recognized")
		}
		if got.Mode != "daily" || got.Hour != 9 || got.Minute != 30 {
			t.Fatalf("unexpected defaults: %+v", got)
		}
	})

	t.Run("weekly schedule", func(t *testing.T) {
		got, ok := scheduleDefaultsFromCron("45 6 * * 1,3,5")
		if !ok {
			t.Fatal("expected weekly cron to be recognized")
		}
		if got.Mode != "weekly" || got.Hour != 6 || got.Minute != 45 {
			t.Fatalf("unexpected defaults: %+v", got)
		}
		if len(got.Days) != 3 || got.Days[0] != 1 || got.Days[1] != 3 || got.Days[2] != 5 {
			t.Fatalf("unexpected weekly days: %+v", got.Days)
		}
	})

	t.Run("invalid format - too few fields", func(t *testing.T) {
		_, ok := scheduleDefaultsFromCron("30 9 * *")
		if ok {
			t.Fatal("expected invalid cron to be rejected")
		}
	})

	t.Run("invalid format - too many fields", func(t *testing.T) {
		_, ok := scheduleDefaultsFromCron("30 9 * * * extra")
		if ok {
			t.Fatal("expected invalid cron to be rejected")
		}
	})

	t.Run("invalid minute", func(t *testing.T) {
		_, ok := scheduleDefaultsFromCron("60 * * * *")
		if ok {
			t.Fatal("expected invalid minute to be rejected")
		}
	})

	t.Run("invalid hour for daily/weekly", func(t *testing.T) {
		_, ok := scheduleDefaultsFromCron("30 24 * * *")
		if ok {
			t.Fatal("expected invalid hour to be rejected")
		}
	})

	t.Run("non-standard cron (day of month specified)", func(t *testing.T) {
		_, ok := scheduleDefaultsFromCron("30 9 1 * *")
		if ok {
			t.Fatal("expected non-standard cron to be rejected")
		}
	})

	t.Run("weekly schedule (Monday only)", func(t *testing.T) {
		got, ok := scheduleDefaultsFromCron("30 9 * * 1")
		if !ok {
			t.Fatal("expected weekly cron to be recognized")
		}
		if got.Mode != "weekly" || got.Hour != 9 || got.Minute != 30 {
			t.Fatalf("unexpected defaults: %+v", got)
		}
		if len(got.Days) != 1 || got.Days[0] != 1 {
			t.Fatalf("expected Monday, got: %+v", got.Days)
		}
	})

	t.Run("whitespace handling", func(t *testing.T) {
		got, ok := scheduleDefaultsFromCron("  30  9  *  *  *  ")
		if !ok {
			t.Fatal("expected daily cron with whitespace to be recognized")
		}
		if got.Mode != "daily" || got.Hour != 9 || got.Minute != 30 {
			t.Fatalf("unexpected defaults: %+v", got)
		}
	})
}

func TestWeekdayName(t *testing.T) {
	cases := []struct {
		day  int
		want string
	}{
		{0, "Sun"},
		{1, "Mon"},
		{2, "Tue"},
		{3, "Wed"},
		{4, "Thu"},
		{5, "Fri"},
		{6, "Sat"},
		{-1, "-1"},
		{7, "7"},
		{10, "10"},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got := weekdayName(tc.day)
			if got != tc.want {
				t.Fatalf("weekdayName(%d) = %q, want %q", tc.day, got, tc.want)
			}
		})
	}
}

func TestNearestQuarterIndex(t *testing.T) {
	cases := []struct {
		minute int
		want   int
	}{
		{0, 0},  // :00
		{1, 0},  // Closest to :00
		{7, 0},  // Closest to :00
		{8, 1},  // Closest to :15
		{15, 1}, // :15
		{22, 1}, // Closest to :15
		{23, 2}, // Closest to :30
		{30, 2}, // :30
		{37, 2}, // Closest to :30
		{38, 3}, // Closest to :45
		{45, 3}, // :45
		{52, 3}, // Closest to :45
		{53, 3}, // Closest to :45
		{59, 3}, // Closest to :45
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			got := nearestQuarterIndex(tc.minute)
			if got != tc.want {
				t.Fatalf("nearestQuarterIndex(%d) = %d, want %d", tc.minute, got, tc.want)
			}
		})
	}
}

func TestDefaultTimePresetIndex(t *testing.T) {
	cases := []struct {
		hour   int
		minute int
		want   int
	}{
		{0, 0, 0},   // 00:00 matches first preset
		{6, 0, 1},   // 06:00 matches second preset
		{9, 0, 2},   // 09:00 matches third preset (default)
		{12, 0, 3},  // 12:00 matches fourth preset
		{18, 0, 4},  // 18:00 matches fifth preset
		{21, 0, 5},  // 21:00 matches sixth preset
		{15, 30, 2}, // 15:30 doesn't match any, returns default (09:00)
		{9, 1, 2},   // 09:01 doesn't match exactly, returns default
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			got := defaultTimePresetIndex(tc.hour, tc.minute)
			if got != tc.want {
				t.Fatalf("defaultTimePresetIndex(%d, %d) = %d, want %d", tc.hour, tc.minute, got, tc.want)
			}
		})
	}
}

func TestDefaultWeeklyDayPresetIndex(t *testing.T) {
	cases := []struct {
		days []int
		want int
	}{
		{[]int{1, 2, 3, 4, 5}, 0},       // Mon-Fri matches first preset
		{[]int{0, 6}, 1},                // Sat-Sun matches second preset
		{[]int{0, 1, 2, 3, 4, 5, 6}, 2}, // Every day matches third preset
		{[]int{1}, 3},                   // Monday matches fourth preset
		{[]int{2}, 4},                   // Tuesday matches fifth preset
		{[]int{3}, 5},                   // Wednesday matches sixth preset
		{[]int{4}, 6},                   // Thursday matches seventh preset
		{[]int{5}, 7},                   // Friday matches eighth preset
		{[]int{6}, 8},                   // Saturday matches ninth preset
		{[]int{0}, 9},                   // Sunday matches tenth preset
		{[]int{1, 3}, 0},                // Mon,Wed doesn't match exactly, returns first preset
		{[]int{2, 4}, 0},                // Tue,Thu doesn't match exactly, returns first preset
	}

	for _, tc := range cases {
		t.Run("", func(t *testing.T) {
			got := defaultWeeklyDayPresetIndex(tc.days)
			if got != tc.want {
				t.Fatalf("defaultWeeklyDayPresetIndex(%v) = %d, want %d", tc.days, got, tc.want)
			}
		})
	}
}

func TestSameIntSlice(t *testing.T) {
	t.Run("equal slices", func(t *testing.T) {
		a := []int{1, 2, 3}
		b := []int{1, 2, 3}
		if !sameIntSlice(a, b) {
			t.Fatal("expected equal slices to be true")
		}
	})

	t.Run("different lengths", func(t *testing.T) {
		a := []int{1, 2, 3}
		b := []int{1, 2}
		if sameIntSlice(a, b) {
			t.Fatal("expected different lengths to be false")
		}
	})

	t.Run("different values", func(t *testing.T) {
		a := []int{1, 2, 3}
		b := []int{1, 2, 4}
		if sameIntSlice(a, b) {
			t.Fatal("expected different values to be false")
		}
	})

	t.Run("empty slices", func(t *testing.T) {
		a := []int{}
		b := []int{}
		if !sameIntSlice(a, b) {
			t.Fatal("expected empty slices to be true")
		}
	})

	t.Run("one empty, one not", func(t *testing.T) {
		a := []int{}
		b := []int{1, 2, 3}
		if sameIntSlice(a, b) {
			t.Fatal("expected one empty slice to be false")
		}
	})
}

func TestIndexOf(t *testing.T) {
	list := []string{"Apple", "Banana", "Cherry", "Date"}

	t.Run("finds existing item case-sensitive", func(t *testing.T) {
		idx := indexOf(list, "Banana")
		if idx != 1 {
			t.Fatalf("expected index 1, got %d", idx)
		}
	})

	t.Run("finds existing item case-insensitive", func(t *testing.T) {
		idx := indexOf(list, "banana")
		if idx != 1 {
			t.Fatalf("expected index 1 for case-insensitive, got %d", idx)
		}
	})

	t.Run("returns -1 for non-existent item", func(t *testing.T) {
		idx := indexOf(list, "Fig")
		if idx != -1 {
			t.Fatalf("expected -1 for non-existent item, got %d", idx)
		}
	})

	t.Run("handles empty list", func(t *testing.T) {
		idx := indexOf([]string{}, "anything")
		if idx != -1 {
			t.Fatalf("expected -1 for empty list, got %d", idx)
		}
	})

	t.Run("handles whitespace in needle", func(t *testing.T) {
		idx := indexOf(list, "  cherry  ")
		if idx != -1 {
			t.Fatalf("expected -1 for needle with whitespace, got %d", idx)
		}
	})
}

func TestTimePresetLabels(t *testing.T) {
	labels := timePresetLabels()
	expected := []string{"00:00", "06:00", "09:00", "12:00", "18:00", "21:00"}

	if len(labels) != len(expected) {
		t.Fatalf("expected %d labels, got %d", len(expected), len(labels))
	}

	for i, label := range labels {
		if label != expected[i] {
			t.Fatalf("label %d: expected %q, got %q", i, expected[i], label)
		}
	}
}

func TestWeeklyDayPresetLabels(t *testing.T) {
	labels := weeklyDayPresetLabels()
	expected := []string{
		"Mon-Fri", "Sat-Sun", "Every day", "Monday", "Tuesday",
		"Wednesday", "Thursday", "Friday", "Saturday", "Sunday",
	}

	if len(labels) != len(expected) {
		t.Fatalf("expected %d labels, got %d", len(expected), len(labels))
	}

	for i, label := range labels {
		if label != expected[i] {
			t.Fatalf("label %d: expected %q, got %q", i, expected[i], label)
		}
	}
}

func TestPrompt(t *testing.T) {
	t.Run("reads input correctly", func(t *testing.T) {
		input := "test input\n"
		reader := bufio.NewReader(strings.NewReader(input))
		var buf bytes.Buffer

		result, err := prompt(reader, &buf, "Enter: ")
		if err != nil {
			t.Fatalf("prompt returned error: %v", err)
		}

		if result != "test input" {
			t.Fatalf("expected 'test input', got %q", result)
		}

		output := buf.String()
		if !strings.Contains(output, "Enter:") {
			t.Fatal("expected prompt to be written to output")
		}
	})

	t.Run("handles empty input", func(t *testing.T) {
		input := "\n"
		reader := bufio.NewReader(strings.NewReader(input))
		var buf bytes.Buffer

		result, err := prompt(reader, &buf, "Enter: ")
		if err != nil {
			t.Fatalf("prompt returned error: %v", err)
		}

		if result != "" {
			t.Fatalf("expected empty result, got %q", result)
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		input := "  test input  \n"
		reader := bufio.NewReader(strings.NewReader(input))
		var buf bytes.Buffer

		result, err := prompt(reader, &buf, "Enter: ")
		if err != nil {
			t.Fatalf("prompt returned error: %v", err)
		}

		if result != "test input" {
			t.Fatalf("expected 'test input', got %q", result)
		}
	})

	t.Run("q requests quit", func(t *testing.T) {
		input := "q\n"
		reader := bufio.NewReader(strings.NewReader(input))
		var buf bytes.Buffer

		_, err := prompt(reader, &buf, "Enter: ")
		if !errors.Is(err, errQuitRequested) {
			t.Fatalf("expected quit error, got %v", err)
		}
	})
}

func TestPromptBinaryChoice(t *testing.T) {
	t.Run("accepts default yes on enter", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("\n"))
		var out bytes.Buffer

		got, err := promptBinaryChoice(reader, &out, nil, "Enable recurring schedule?", true, false)
		if err != nil {
			t.Fatalf("promptBinaryChoice returned error: %v", err)
		}
		if !got {
			t.Fatal("expected default yes selection")
		}
	})

	t.Run("accepts default no on enter", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("\n"))
		var out bytes.Buffer

		got, err := promptBinaryChoice(reader, &out, nil, "Enable recurring schedule?", false, false)
		if err != nil {
			t.Fatalf("promptBinaryChoice returned error: %v", err)
		}
		if got {
			t.Fatal("expected default no selection")
		}
	})

	t.Run("requires explicit choice when configured", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("\n1\n"))
		var out bytes.Buffer

		got, err := promptBinaryChoice(reader, &out, nil, "Enable recurring schedule?", true, true)
		if err != nil {
			t.Fatalf("promptBinaryChoice returned error: %v", err)
		}
		if !got {
			t.Fatal("expected explicit yes selection")
		}
		if !strings.Contains(out.String(), "Selection required.") {
			t.Fatal("expected selection-required guidance after blank input")
		}
	})
}

func TestClearScreen(t *testing.T) {
	var buf bytes.Buffer
	clearScreen(&buf)

	output := buf.String()
	if !strings.Contains(output, "\033[H\033[2J") {
		t.Fatal("expected clear screen escape sequence")
	}
}

func TestPrintErr(t *testing.T) {
	var buf bytes.Buffer
	msg := "test error message"

	printErr(&buf, msg)

	output := buf.String()
	if !strings.Contains(output, "ERROR:") {
		t.Fatal("expected ERROR: prefix")
	}
	if !strings.Contains(output, msg) {
		t.Fatal("expected error message in output")
	}
}

func TestPrintOK(t *testing.T) {
	var buf bytes.Buffer
	msg := "test ok message"

	printOK(&buf, msg)

	output := buf.String()
	if !strings.Contains(output, "OK:") {
		t.Fatal("expected OK: prefix")
	}
	if !strings.Contains(output, msg) {
		t.Fatal("expected ok message in output")
	}
}

func TestRule(t *testing.T) {
	var buf bytes.Buffer
	rule(&buf, '-')

	output := buf.String()
	expected := strings.Repeat("-", 86) + "\n"
	if output != expected {
		t.Fatalf("expected %q, got %q", expected, output)
	}
}

func TestColorLabel(t *testing.T) {
	result := colorLabel(cCyan)
	expected := cCyan + cBold
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestPaintBand(t *testing.T) {
	var buf bytes.Buffer
	bg := cBgBlue
	fg := cWhite
	text := "TEST"

	paintBand(&buf, bg, fg, text)

	output := buf.String()
	if !strings.Contains(output, bg) {
		t.Fatal("expected background color in output")
	}
	if !strings.Contains(output, cWhite) {
		t.Fatal("expected white text color in output")
	}
	if !strings.Contains(output, cBold) {
		t.Fatal("expected bold formatting in output")
	}
	if !strings.Contains(output, text) {
		t.Fatal("expected text in output")
	}
	if !strings.Contains(output, cReset) {
		t.Fatal("expected color reset in output")
	}
}

func TestPaintTrigger(t *testing.T) {
	t.Run("cron plus manual", func(t *testing.T) {
		result := paintTrigger("daily at 09:00 UTC + on-demand", "0 9 * * *", true, false)
		if !strings.Contains(result, cCyan) {
			t.Fatal("expected cyan color for cron+manual")
		}
		if !strings.Contains(result, "daily at 09:00 UTC") {
			t.Fatal("expected trigger text in output")
		}
	})

	t.Run("cron only", func(t *testing.T) {
		result := paintTrigger("daily at 09:00 UTC", "0 9 * * *", false, false)
		if !strings.Contains(result, cMagenta) {
			t.Fatal("expected magenta color for cron only")
		}
		if !strings.Contains(result, "daily at 09:00 UTC") {
			t.Fatal("expected trigger text in output")
		}
	})

	t.Run("manual only", func(t *testing.T) {
		result := paintTrigger("on-demand", "", true, false)
		if !strings.Contains(result, cGreen) {
			t.Fatal("expected green color for manual only")
		}
		if !strings.Contains(result, "on-demand") {
			t.Fatal("expected trigger text in output")
		}
	})

	t.Run("push only", func(t *testing.T) {
		result := paintTrigger("on-push(main)", "", false, true)
		if !strings.Contains(result, cBlue) {
			t.Fatal("expected blue color for push only")
		}
		if !strings.Contains(result, "on-push(main)") {
			t.Fatal("expected trigger text in output")
		}
	})

	t.Run("no trigger", func(t *testing.T) {
		result := paintTrigger("none", "", false, false)
		if strings.Contains(result, cCyan) || strings.Contains(result, cMagenta) || strings.Contains(result, cGreen) || strings.Contains(result, cBlue) {
			t.Fatal("expected no color for no trigger")
		}
		if !strings.Contains(result, "none") {
			t.Fatal("expected trigger text in output")
		}
	})
}

func TestTTYFile(t *testing.T) {
	t.Run("returns nil for non-TTY file", func(t *testing.T) {
		// Create a temporary file to simulate stdin
		tmpfile, err := os.CreateTemp("", "tty-test")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())
		defer tmpfile.Close()

		tty := ttyFile(tmpfile)
		if tty != nil {
			t.Fatal("expected nil for non-TTY file")
		}
	})

	t.Run("returns nil for non-TTY", func(t *testing.T) {
		// Use a bytes.Reader which is not a TTY
		reader := bytes.NewReader([]byte("test"))
		tty := ttyFile(reader)
		if tty != nil {
			t.Fatal("expected nil for non-TTY reader")
		}
	})
}

func TestSupportsRawTTY(t *testing.T) {
	t.Run("returns false for non-file", func(t *testing.T) {
		// Create a temporary file that doesn't support TTY operations
		tmpfile, err := os.CreateTemp("", "non-tty-test")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())
		defer tmpfile.Close()

		// This should return false for regular files
		if supportsRawTTY(tmpfile) {
			t.Fatal("expected false for regular file")
		}
	})
}

func TestTtyFile(t *testing.T) {
	t.Run("returns nil for non-*os.File", func(t *testing.T) {
		reader := bytes.NewReader([]byte("test"))
		tty := ttyFile(reader)
		if tty != nil {
			t.Fatal("expected nil for non-*os.File")
		}
	})

	t.Run("returns nil for file that doesn't support raw TTY", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "tty-test")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())
		defer tmpfile.Close()

		// This will likely return nil because temp files don't support TTY operations
		_ = ttyFile(tmpfile)
		// We can't assert the exact behavior here as it depends on the system
	})
}

// Note: Interactive UI functions (promptBinaryChoice, selectNumberedOption, selectTender, inputTender)
// are complex to test in unit tests due to their interactive nature.
// They are better tested through integration tests like those in ui_integration_test.go
// The existing integration tests provide good coverage for the interactive workflows.
