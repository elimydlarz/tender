package tender

import "testing"

func TestTriggerSummary(t *testing.T) {
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
}
