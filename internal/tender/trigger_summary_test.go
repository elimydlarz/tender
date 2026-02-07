package tender

import "testing"

func TestTriggerSummary(t *testing.T) {
	t.Run("generates summaries for different trigger configurations", func(t *testing.T) {
		cases := []struct {
			name   string
			cron   string
			manual bool
			want   string
		}{
			{name: "on-demand only", cron: "", manual: true, want: "on-demand"},
			{name: "hourly", cron: "15 * * * *", manual: false, want: "every hour at :15 UTC"},
			{name: "daily plus on-demand", cron: "30 9 * * *", manual: true, want: "daily at 09:30 UTC + on-demand"},
			{name: "weekly", cron: "45 6 * * 1,3", manual: false, want: "weekly Mon,Wed at 06:45 UTC"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got := TriggerSummary(tc.cron, tc.manual)
				if got != tc.want {
					t.Fatalf("unexpected trigger summary: got=%q want=%q", got, tc.want)
				}
			})
		}
	})

	t.Run("handles edge cases", func(t *testing.T) {
		t.Run("no triggers configured", func(t *testing.T) {
			got := TriggerSummary("", false)
			want := "none"
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})

		t.Run("invalid cron format", func(t *testing.T) {
			got := TriggerSummary("invalid", false)
			want := "scheduled"
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})

		t.Run("complex weekly schedule", func(t *testing.T) {
			got := TriggerSummary("0 12 * * 0,1,2,3,4,5,6", false)
			want := "weekly Sun,Mon,Tue,Wed,Thu,Fri,Sat at 12:00 UTC"
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})

		t.Run("midnight hourly", func(t *testing.T) {
			got := TriggerSummary("0 * * * *", false)
			want := "every hour at :00 UTC"
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})

		t.Run("single day weekly", func(t *testing.T) {
			got := TriggerSummary("30 14 * * 5", false)
			want := "weekly Fri at 14:30 UTC"
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})

		t.Run("manual with empty cron", func(t *testing.T) {
			got := TriggerSummary("", true)
			want := "on-demand"
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})

		t.Run("manual with valid cron", func(t *testing.T) {
			got := TriggerSummary("15 10 * * *", true)
			want := "daily at 10:15 UTC + on-demand"
			if got != want {
				t.Fatalf("expected %q, got %q", want, got)
			}
		})
	})
}
