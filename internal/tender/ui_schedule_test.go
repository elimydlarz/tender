package tender

import "testing"

// This file contains basic tests for schedule-related functions.
// More comprehensive tests are in ui_unit_test.go.

func TestBuildHourlyCron_Basic(t *testing.T) {
	got, err := buildHourlyCron("15")
	if err != nil {
		t.Fatalf("buildHourlyCron returned error: %v", err)
	}
	if got != "15 * * * *" {
		t.Fatalf("unexpected cron: %q", got)
	}
}

func TestBuildDailyCron_Basic(t *testing.T) {
	got, err := buildDailyCron("09:30")
	if err != nil {
		t.Fatalf("buildDailyCron returned error: %v", err)
	}
	if got != "30 9 * * *" {
		t.Fatalf("unexpected cron: %q", got)
	}
}

func TestScheduleDefaultsFromCron_Basic(t *testing.T) {
	t.Run("hourly", func(t *testing.T) {
		got, ok := scheduleDefaultsFromCron("5 * * * *")
		if !ok {
			t.Fatal("expected hourly cron to be recognized")
		}
		if got.Mode != "hourly" || got.Minute != 5 {
			t.Fatalf("unexpected defaults: %+v", got)
		}
	})

	t.Run("daily", func(t *testing.T) {
		got, ok := scheduleDefaultsFromCron("30 9 * * *")
		if !ok {
			t.Fatal("expected daily cron to be recognized")
		}
		if got.Mode != "daily" || got.Hour != 9 || got.Minute != 30 {
			t.Fatalf("unexpected defaults: %+v", got)
		}
	})

	t.Run("weekly", func(t *testing.T) {
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
}
