package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"calango/internal/audit"
)

type fakeAudit struct{ v audit.Verification }

func (f fakeAudit) VerifyAudit(context.Context) (audit.Verification, error) { return f.v, nil }

func TestAuditVerifyReportsTheFirstBreak(t *testing.T) {
	h := &handler{
		audit: fakeAudit{v: audit.Verification{
			EntriesChecked: 3,
			FirstBreak:     &audit.Break{Seq: 2, ID: "entry-2", Reason: "entry hash mismatch"},
		}},
		localToken: newTestLocalToken(t),
		ownerID:    testOwner,
		deviceID:   testDeviceID,
	}
	mux := http.NewServeMux()
	h.registerAudit(mux)

	rec := call(t, h.authenticate(mux), http.MethodGet, "/v1/audit/verify", testToken, "")
	expectStatus(t, rec, http.StatusOK)
	var got auditVerification
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode verification: %v", err)
	}
	if got.OK || got.EntriesChecked != 3 || got.FirstBreak == nil ||
		got.FirstBreak.Seq != 2 || got.FirstBreak.Reason != "entry hash mismatch" {
		t.Fatalf("verification = %+v, want first break at seq 2", got)
	}
}

func TestAuditVerifyIsReadScopedAndEmptyChainIsOK(t *testing.T) {
	h := newTestHandler(t)
	read := pairDevice(t, h, issuePairingCode(t, h, "read"))
	approveOnly := pairDevice(t, h, issuePairingCode(t, h, "approve_only"))

	rec := call(t, h, http.MethodGet, "/v1/audit/verify", read.Token, "")
	expectStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"first_break":null`) {
		t.Fatalf("empty chain body = %s, want ok true and no break", body)
	}
	expectStatus(t, call(t, h, http.MethodGet, "/v1/audit/verify", approveOnly.Token, ""), http.StatusForbidden)
}
