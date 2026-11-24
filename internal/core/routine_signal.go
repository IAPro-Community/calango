package core

import (
	"encoding/json"
	"strings"
)

const routineSignalPrefix = "ROUTINE:"

const routineContract = `This message is automatic, not from a person. End your reply with one line, exactly in this form:
ROUTINE: {"had_news": true or false, "finish": true or false}
had_news: true when the owner has something new to read from this run.
finish: true only when this routine has nothing left to do and must never run again.`

// RoutineSignal is the control line an agent ends a routine reply with.
type RoutineSignal struct {
	HadNews bool
	Finish  bool
}

// RoutineMessage is the message a routine sends: its prompt followed by the control contract.
func RoutineMessage(prompt string) string {
	return prompt + "\n\n" + routineContract
}

// ParseRoutineSignal reads the last ROUTINE line of a reply. A missing or malformed line
// gives HadNews=true and Finish=false, so the owner hears about the run instead of missing
// it, and the routine keeps running. A missing had_news key counts as true for the same reason.
// ok reports whether a well-formed line was found.
func ParseRoutineSignal(reply string) (RoutineSignal, bool) {
	fallback := RoutineSignal{HadNews: true}
	lines := strings.Split(reply, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		rest, found := strings.CutPrefix(strings.TrimSpace(lines[i]), routineSignalPrefix)
		if !found {
			continue
		}
		var raw struct {
			HadNews *bool `json:"had_news"`
			Finish  bool  `json:"finish"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(rest)), &raw); err != nil {
			return fallback, false
		}
		sig := RoutineSignal{HadNews: true, Finish: raw.Finish}
		if raw.HadNews != nil {
			sig.HadNews = *raw.HadNews
		}
		return sig, true
	}
	return fallback, false
}
