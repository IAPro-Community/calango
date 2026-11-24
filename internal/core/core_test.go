package core

import (
	"errors"
	"testing"
)

func chainOf(n int) []ProviderRef {
	refs := make([]ProviderRef, n)
	for i := range refs {
		refs[i] = ProviderRef{AccountID: "account-1", Kind: "codex", Label: "default", Model: "m-1"}
	}
	return refs
}

func TestAgentValidateAcceptsCompleteConfiguration(t *testing.T) {
	a := Agent{Name: "Assistant", TokenLimitMonth: 1000, RunOn: RunOnPC, ApprovalLevel: ApprovalSensitive, ReasoningEffort: EffortHigh, Providers: chainOf(1)}
	if err := a.Validate(); err != nil {
		t.Fatalf("valid agent rejected: %v", err)
	}
}

func TestAgentValidateRejectsInvalidConfiguration(t *testing.T) {
	valid := Agent{Name: "Assistant", TokenLimitMonth: 1000, RunOn: RunOnPC, ApprovalLevel: ApprovalSensitive, Providers: chainOf(1)}
	cases := map[string]func(*Agent){
		"empty name":        func(a *Agent) { a.Name = "" },
		"missing limit":     func(a *Agent) { a.TokenLimitMonth = 0 },
		"negative limit":    func(a *Agent) { a.TokenLimitMonth = -1 },
		"unknown run_on":    func(a *Agent) { a.RunOn = "cloud" },
		"unknown approval":  func(a *Agent) { a.ApprovalLevel = "never" },
		"unknown reasoning": func(a *Agent) { a.ReasoningEffort = "extreme" },
		"no providers":      func(a *Agent) { a.Providers = nil },
		"blank model":       func(a *Agent) { a.Providers = []ProviderRef{{AccountID: "account-1", Model: " "}} },
		"missing account":   func(a *Agent) { a.Providers = []ProviderRef{{Model: "m-1"}} },
		"too many accounts": func(a *Agent) { a.Providers = chainOf(MaxProvidersPerAgent + 1) },
	}
	for name, mutate := range cases {
		a := valid
		mutate(&a)
		if err := a.Validate(); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}
