package core

import (
	"fmt"
	"strings"
	"time"
)

// TriggerKind says how a routine is scheduled. Event triggers arrive with connectors (phase 3).
type TriggerKind string

// Trigger kinds.
const (
	TriggerCron     TriggerKind = "cron"
	TriggerInterval TriggerKind = "interval"
)

// NotifyOn says when the owner is told about a successful occurrence.
type NotifyOn string

// Notification policies. Exhausted retries always notify the owner, whatever the policy.
const (
	NotifyAlways  NotifyOn = "always"
	NotifyNews    NotifyOn = "news"
	NotifyFailure NotifyOn = "failure"
)

// OccurrenceStatus is the state of one scheduled time of a routine.
type OccurrenceStatus string

// Occurrence states. A pending occurrence with attempt > 0 is waiting for a retry.
const (
	OccurrencePending OccurrenceStatus = "pending"
	OccurrenceRunning OccurrenceStatus = "running"
	OccurrenceDone    OccurrenceStatus = "done"
	OccurrenceFailed  OccurrenceStatus = "failed"
)

// MaxRoutineRetries bounds max_retries so a broken routine cannot retry for long.
const MaxRoutineRetries = 5

// minIntervalRoutine is the shortest interval a routine may use.
const minIntervalRoutine = time.Minute

// Routine is scheduled work an agent performs on its own. Quiet window bounds are minutes
// after midnight in the node's time zone; both are nil or both are set.
type Routine struct {
	ID          string
	OwnerID     string
	AgentID     string
	Name        string
	Prompt      string
	TriggerKind TriggerKind
	// Schedule is a 5-field cron expression for cron triggers or a Go duration for intervals.
	Schedule   string
	RunOn      RunOn
	QuietStart *int
	QuietEnd   *int
	MaxRetries int
	NotifyOn   NotifyOn
	Paused     bool
	NextRunAt  time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time

	// ConversationID is the conversation the routine's runs happen in. The first occurrence creates it.
	ConversationID string
	// EndedAt is set when the agent ends the routine with routine_finish. An ended routine never fires again.
	EndedAt *time.Time
}

// Normalize fills defaults. It does not check values; Validate does.
func (r *Routine) Normalize() {
	if r.RunOn == "" {
		r.RunOn = RunOnPC
	}
	if r.NotifyOn == "" {
		r.NotifyOn = NotifyFailure
	}
}

// Validate checks the definition. Errors wrap ErrInvalidInput and name the field.
func (r Routine) Validate(loc *time.Location) error {
	if r.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidInput)
	}
	if strings.TrimSpace(r.Prompt) == "" {
		return fmt.Errorf("%w: prompt is required", ErrInvalidInput)
	}
	if err := ValidateMessageBody(RoutineMessage(r.Prompt)); err != nil {
		return fmt.Errorf("prompt: %w", err)
	}
	if _, err := r.NextAfter(time.Time{}, loc); err != nil {
		return err
	}
	switch r.RunOn {
	case RunOnPC, RunOnVPS, RunOnAuto:
	default:
		return fmt.Errorf("%w: run_on must be pc, vps or auto", ErrInvalidInput)
	}
	switch r.NotifyOn {
	case NotifyAlways, NotifyNews, NotifyFailure:
	default:
		return fmt.Errorf("%w: notify_on must be always, news or failure", ErrInvalidInput)
	}
	if (r.QuietStart == nil) != (r.QuietEnd == nil) {
		return fmt.Errorf("%w: quiet window needs both start and end", ErrInvalidInput)
	}
	if r.QuietStart != nil {
		s, e := *r.QuietStart, *r.QuietEnd
		if s < 0 || s > 1439 || e < 0 || e > 1439 || s == e {
			return fmt.Errorf("%w: quiet window must be two different minutes of the day (0-1439)", ErrInvalidInput)
		}
	}
	if r.MaxRetries < 0 || r.MaxRetries > MaxRoutineRetries {
		return fmt.Errorf("%w: max_retries must be between 0 and %d", ErrInvalidInput, MaxRoutineRetries)
	}
	return nil
}

// NextAfter returns the next scheduled time strictly after after, in loc.
func (r Routine) NextAfter(after time.Time, loc *time.Location) (time.Time, error) {
	switch r.TriggerKind {
	case TriggerCron:
		spec, err := ParseCron(r.Schedule)
		if err != nil {
			return time.Time{}, err
		}
		return spec.Next(after, loc)
	case TriggerInterval:
		d, err := time.ParseDuration(r.Schedule)
		if err != nil || d < minIntervalRoutine {
			return time.Time{}, fmt.Errorf("%w: interval must be a duration of at least 1m", ErrInvalidInput)
		}
		return after.Add(d), nil
	default:
		return time.Time{}, fmt.Errorf("%w: trigger must be cron or interval", ErrInvalidInput)
	}
}

// OccurrenceID is the deterministic identity of one scheduled time of a routine. The same
// routine and scheduled time always produce the same ID, so retries and restarts reuse it.
func OccurrenceID(routineID string, scheduledFor time.Time) string {
	return routineID + "@" + scheduledFor.UTC().Format(time.RFC3339)
}

// RetryBackoff is the delay before retry number attempt (1 for the first retry).
// The schedule is 1, 5, then 30 minutes, and stays at 30 minutes.
func RetryBackoff(attempt int) time.Duration {
	switch {
	case attempt <= 1:
		return time.Minute
	case attempt == 2:
		return 5 * time.Minute
	default:
		return 30 * time.Minute
	}
}

// PostponeOutsideQuietWindow returns t unchanged when it is outside the window. Inside
// the window it returns the window's end, so the occurrence is delayed, not discarded.
// Windows may cross midnight (start later than end).
func PostponeOutsideQuietWindow(t time.Time, loc *time.Location, start, end *int) time.Time {
	if start == nil || end == nil {
		return t
	}
	local := t.In(loc)
	minute := local.Hour()*60 + local.Minute()
	s, e := *start, *end
	var inside bool
	if s < e {
		inside = minute >= s && minute < e
	} else {
		inside = minute >= s || minute < e
	}
	if !inside {
		return t
	}
	day := local.Day()
	if s > e && minute >= s {
		day++
	}
	return time.Date(local.Year(), local.Month(), day, e/60, e%60, 0, 0, loc)
}

// Occurrence is one scheduled time of a routine and its delivery state.
type Occurrence struct {
	ID            string
	RoutineID     string
	OccurrenceID  string
	ScheduledFor  time.Time
	RunID         string
	Attempt       int
	Status        OccurrenceStatus
	HadNews       *bool
	LastError     string
	NextAttemptAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
