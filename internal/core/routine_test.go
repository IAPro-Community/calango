package core

import (
	"errors"
	"testing"
	"time"
)

var utc = time.UTC

func TestParseCronRejectsInvalidExpressions(t *testing.T) {
	cases := map[string]string{
		"four fields":    "* * * *",
		"minute too big": "60 * * * *",
		"zero step":      "*/0 * * * *",
		"range reversed": "30-10 * * * *",
		"not a number":   "a * * * *",
		"hour too big":   "0 24 * * *",
		"dom zero":       "0 0 0 * *",
	}
	for name, expr := range cases {
		if _, err := ParseCron(expr); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s (%q): error = %v, want ErrInvalidInput", name, expr, err)
		}
	}
}

func TestCronNextMatchesStandardSemantics(t *testing.T) {
	ny := time.FixedZone("UTC-3", -3*3600)
	cases := []struct {
		name string
		expr string
		from time.Time
		loc  *time.Location
		want time.Time
	}{
		{
			name: "every 15 minutes is strictly after the instant",
			expr: "*/15 * * * *",
			from: time.Date(2026, 1, 1, 9, 7, 0, 0, utc),
			loc:  utc,
			want: time.Date(2026, 1, 1, 9, 15, 0, 0, utc),
		},
		{
			name: "weekdays skip the weekend",
			expr: "0 9 * * 1-5",
			from: time.Date(2026, 1, 3, 10, 0, 0, 0, utc), // Saturday
			loc:  utc,
			want: time.Date(2026, 1, 5, 9, 0, 0, 0, utc), // Monday
		},
		{
			name: "day of week 7 means Sunday",
			expr: "0 9 * * 7",
			from: time.Date(2026, 1, 3, 10, 0, 0, 0, utc), // Saturday
			loc:  utc,
			want: time.Date(2026, 1, 4, 9, 0, 0, 0, utc), // Sunday
		},
		{
			name: "day of month OR day of week when both are restricted",
			expr: "0 0 13 * 5",
			from: time.Date(2026, 1, 1, 0, 0, 0, 0, utc), // Thursday
			loc:  utc,
			want: time.Date(2026, 1, 2, 0, 0, 0, 0, utc), // Friday comes before the 13th
		},
		{
			name: "fires in the account time zone",
			expr: "0 9 * * *",
			from: time.Date(2026, 1, 1, 0, 0, 0, 0, utc),
			loc:  ny,
			want: time.Date(2026, 1, 1, 12, 0, 0, 0, utc), // 09:00 at UTC-3
		},
		{
			name: "29 February next leap year",
			expr: "0 0 29 2 *",
			from: time.Date(2026, 3, 1, 0, 0, 0, 0, utc),
			loc:  utc,
			want: time.Date(2028, 2, 29, 0, 0, 0, 0, utc),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec, err := ParseCron(tc.expr)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got, err := spec.Next(tc.from, tc.loc)
			if err != nil {
				t.Fatalf("next: %v", err)
			}
			if !got.Equal(tc.want) {
				t.Fatalf("next = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestCronWithoutMatchFailsInsteadOfLooping(t *testing.T) {
	spec, err := ParseCron("0 0 30 2 *") // 30 February never happens
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := spec.Next(time.Date(2026, 1, 1, 0, 0, 0, 0, utc), utc); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func validRoutine() Routine {
	r := Routine{
		Name:        "digest",
		Prompt:      "summarise the inbox",
		TriggerKind: TriggerCron,
		Schedule:    "0 9 * * 1-5",
		RunOn:       RunOnPC,
		NotifyOn:    NotifyNews,
		MaxRetries:  2,
	}
	return r
}

func TestRoutineValidateAcceptsValidDefinitions(t *testing.T) {
	cases := map[string]func(*Routine){
		"cron":         func(r *Routine) {},
		"interval":     func(r *Routine) { r.TriggerKind = TriggerInterval; r.Schedule = "1h" },
		"quiet window": func(r *Routine) { s, e := 1320, 420; r.QuietStart, r.QuietEnd = &s, &e },
		"zero retries": func(r *Routine) { r.MaxRetries = 0 },
	}
	for name, mutate := range cases {
		r := validRoutine()
		mutate(&r)
		if err := r.Validate(utc); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestRoutineValidateRejectsInvalidDefinitions(t *testing.T) {
	half := 600
	cases := map[string]func(*Routine){
		"empty name":           func(r *Routine) { r.Name = "" },
		"empty prompt":         func(r *Routine) { r.Prompt = "" },
		"bad cron":             func(r *Routine) { r.Schedule = "every day" },
		"interval too short":   func(r *Routine) { r.TriggerKind = TriggerInterval; r.Schedule = "30s" },
		"unknown trigger":      func(r *Routine) { r.TriggerKind = "webhook" },
		"unknown run_on":       func(r *Routine) { r.RunOn = "cloud" },
		"unknown notify_on":    func(r *Routine) { r.NotifyOn = "never" },
		"quiet start only":     func(r *Routine) { r.QuietStart = &half },
		"quiet equal bounds":   func(r *Routine) { r.QuietStart, r.QuietEnd = &half, &half },
		"quiet minute too big": func(r *Routine) { big := 1440; r.QuietStart, r.QuietEnd = &big, &half },
		"too many retries":     func(r *Routine) { r.MaxRetries = MaxRoutineRetries + 1 },
		"negative retries":     func(r *Routine) { r.MaxRetries = -1 },
	}
	for name, mutate := range cases {
		r := validRoutine()
		mutate(&r)
		if err := r.Validate(utc); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}

func TestOccurrenceIDIsDeterministicAndTimezoneFree(t *testing.T) {
	at := time.Date(2026, 1, 1, 9, 0, 0, 0, utc)
	local := at.In(time.FixedZone("UTC-3", -3*3600))
	if OccurrenceID("r1", at) != OccurrenceID("r1", local) {
		t.Fatalf("same instant produced different occurrence IDs")
	}
	if OccurrenceID("r1", at) == OccurrenceID("r2", at) {
		t.Fatalf("different routines produced the same occurrence ID")
	}
}

func TestRetryBackoffFollowsTheDocumentedSchedule(t *testing.T) {
	cases := map[int]time.Duration{
		1: time.Minute,
		2: 5 * time.Minute,
		3: 30 * time.Minute,
		9: 30 * time.Minute,
	}
	for attempt, want := range cases {
		if got := RetryBackoff(attempt); got != want {
			t.Errorf("attempt %d: backoff = %s, want %s", attempt, got, want)
		}
	}
}

func TestQuietWindowPostponesToItsEnd(t *testing.T) {
	night := func(start, end int) (*int, *int) { return &start, &end }
	s, e := night(22*60, 7*60) // crosses midnight
	cases := []struct {
		name string
		at   time.Time
		want time.Time
	}{
		{"late evening moves to next morning", time.Date(2026, 1, 1, 23, 30, 0, 0, utc), time.Date(2026, 1, 2, 7, 0, 0, 0, utc)},
		{"early morning moves to same morning", time.Date(2026, 1, 1, 3, 0, 0, 0, utc), time.Date(2026, 1, 1, 7, 0, 0, 0, utc)},
		{"midday is untouched", time.Date(2026, 1, 1, 12, 0, 0, 0, utc), time.Date(2026, 1, 1, 12, 0, 0, 0, utc)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PostponeOutsideQuietWindow(tc.at, utc, s, e); !got.Equal(tc.want) {
				t.Fatalf("postponed to %s, want %s", got, tc.want)
			}
		})
	}
	if got := PostponeOutsideQuietWindow(time.Date(2026, 1, 1, 9, 30, 0, 0, utc), utc, nil, nil); !got.Equal(time.Date(2026, 1, 1, 9, 30, 0, 0, utc)) {
		t.Fatalf("no window must not move the time, got %s", got)
	}
}
