package core

import (
	"errors"
	"testing"
)

func TestAgentValidateAcceptsCompleteConfiguration(t *testing.T) {
	a := Agent{Name: "Assistant", TokenLimitMonth: 1000, RunOn: RunOnPC, ApprovalLevel: ApprovalSensitive, ReasoningEffort: EffortHigh}
	if err := a.Validate(); err != nil {
		t.Fatalf("valid agent rejected: %v", err)
	}
}

func TestAgentValidateRejectsInvalidConfiguration(t *testing.T) {
	valid := Agent{Name: "Assistant", TokenLimitMonth: 1000, RunOn: RunOnPC, ApprovalLevel: ApprovalSensitive}
	cases := map[string]func(*Agent){
		"empty name":        func(a *Agent) { a.Name = "" },
		"missing limit":     func(a *Agent) { a.TokenLimitMonth = 0 },
		"negative limit":    func(a *Agent) { a.TokenLimitMonth = -1 },
		"unknown run_on":    func(a *Agent) { a.RunOn = "cloud" },
		"unknown approval":  func(a *Agent) { a.ApprovalLevel = "never" },
		"unknown reasoning": func(a *Agent) { a.ReasoningEffort = "extreme" },
	}
	for name, mutate := range cases {
		a := valid
		mutate(&a)
		if err := a.Validate(); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}
