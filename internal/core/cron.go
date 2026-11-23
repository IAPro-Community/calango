package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronSpec is a parsed 5-field cron expression: minute hour day-of-month month day-of-week.
// Each field is a bitmask of allowed values. Day-of-week 0 and 7 both mean Sunday.
type CronSpec struct {
	minute, hour, dom, month, dow uint64
	domStar, dowStar              bool
}

// cronSearchLimit bounds the search for the next match. Five years covers every valid
// expression that fires at least once, including 29 February.
const cronSearchLimit = 5 * 366 * 24 * 60

// ParseCron parses a 5-field cron expression. Errors wrap ErrInvalidInput.
func ParseCron(expr string) (CronSpec, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return CronSpec{}, fmt.Errorf("%w: cron must have 5 fields, got %d", ErrInvalidInput, len(fields))
	}
	var spec CronSpec
	var err error
	if spec.minute, err = parseCronField(fields[0], 0, 59); err != nil {
		return CronSpec{}, fmt.Errorf("%w: cron minute: %v", ErrInvalidInput, err)
	}
	if spec.hour, err = parseCronField(fields[1], 0, 23); err != nil {
		return CronSpec{}, fmt.Errorf("%w: cron hour: %v", ErrInvalidInput, err)
	}
	if spec.dom, err = parseCronField(fields[2], 1, 31); err != nil {
		return CronSpec{}, fmt.Errorf("%w: cron day of month: %v", ErrInvalidInput, err)
	}
	if spec.month, err = parseCronField(fields[3], 1, 12); err != nil {
		return CronSpec{}, fmt.Errorf("%w: cron month: %v", ErrInvalidInput, err)
	}
	if spec.dow, err = parseCronField(fields[4], 0, 7); err != nil {
		return CronSpec{}, fmt.Errorf("%w: cron day of week: %v", ErrInvalidInput, err)
	}
	if spec.dow&(1<<7) != 0 {
		spec.dow = spec.dow&^(1<<7) | 1
	}
	spec.domStar = strings.HasPrefix(fields[2], "*")
	spec.dowStar = strings.HasPrefix(fields[4], "*")
	return spec, nil
}

func parseCronField(field string, min, max int) (uint64, error) {
	var mask uint64
	for _, part := range strings.Split(field, ",") {
		step := 1
		base := part
		if i := strings.Index(part, "/"); i >= 0 {
			base = part[:i]
			n, err := strconv.Atoi(part[i+1:])
			if err != nil || n <= 0 {
				return 0, fmt.Errorf("invalid step in %q", part)
			}
			step = n
		}
		lo, hi := min, max
		switch {
		case base == "*":
		case strings.Contains(base, "-"):
			bounds := strings.SplitN(base, "-", 2)
			a, errA := strconv.Atoi(bounds[0])
			b, errB := strconv.Atoi(bounds[1])
			if errA != nil || errB != nil {
				return 0, fmt.Errorf("invalid range %q", part)
			}
			lo, hi = a, b
		default:
			n, err := strconv.Atoi(base)
			if err != nil {
				return 0, fmt.Errorf("invalid value %q", part)
			}
			lo = n
			hi = n
			if step != 1 {
				hi = max
			}
		}
		if lo < min || hi > max || lo > hi {
			return 0, fmt.Errorf("%q is outside %d-%d", part, min, max)
		}
		for v := lo; v <= hi; v += step {
			mask |= 1 << uint(v)
		}
	}
	return mask, nil
}

// Next returns the first minute strictly after after, in loc, that matches the spec.
func (c CronSpec) Next(after time.Time, loc *time.Location) (time.Time, error) {
	t := after.In(loc).Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < cronSearchLimit; i++ {
		if c.matches(t) {
			return t, nil
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("%w: cron has no match in the next five years", ErrInvalidInput)
}

func (c CronSpec) matches(t time.Time) bool {
	if c.minute&(1<<uint(t.Minute())) == 0 || c.hour&(1<<uint(t.Hour())) == 0 {
		return false
	}
	if c.month&(1<<uint(t.Month())) == 0 {
		return false
	}
	domOK := c.dom&(1<<uint(t.Day())) != 0
	dowOK := c.dow&(1<<uint(t.Weekday())) != 0
	switch {
	case c.domStar && c.dowStar:
		return true
	case c.domStar:
		return dowOK
	case c.dowStar:
		return domOK
	default:
		// Standard cron: when both fields are restricted, either one matching is enough.
		return domOK || dowOK
	}
}
