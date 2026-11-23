package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type pairedDevice struct {
	DeviceID string `json:"device_id"`
	Token    string `json:"token"`
	Scope    string `json:"scope"`
}

// issuePairingCode asks the desktop for a code. An empty scope sends no body.
func issuePairingCode(t *testing.T, h http.Handler, scope string) string {
	t.Helper()
	body := ""
	if scope != "" {
		body = fmt.Sprintf(`{"scope":%q}`, scope)
	}
	rec := call(t, h, http.MethodPost, "/v1/pairing-codes", testToken, body)
	expectStatus(t, rec, http.StatusCreated)
	var out struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode pairing code: %v", err)
	}
	return out.Code
}

// pairDevice redeems a code without any credential, as a new client does.
func pairDevice(t *testing.T, h http.Handler, code string) pairedDevice {
	t.Helper()
	body := fmt.Sprintf(`{"code":%q,"device_name":"Phone","platform":"ios"}`, code)
	rec := call(t, h, http.MethodPost, "/v1/pair", "", body)
	expectStatus(t, rec, http.StatusOK)
	var out pairedDevice
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode pair response: %v", err)
	}
	return out
}

func TestPairingFlowIssuesWorkingToken(t *testing.T) {
	h := newTestHandler(t)
	code := issuePairingCode(t, h, "")
	paired := pairDevice(t, h, code)

	if paired.Scope != "full" || paired.Token == "" || paired.DeviceID == "" {
		t.Fatalf("pair response = %+v, want full scope with token and device id", paired)
	}
	expectStatus(t, call(t, h, http.MethodGet, "/v1/agents", paired.Token, ""), http.StatusOK)

	list := call(t, h, http.MethodGet, "/v1/devices", testToken, "")
	expectStatus(t, list, http.StatusOK)
	if strings.Contains(list.Body.String(), "token") {
		t.Fatalf("device list leaks token material: %s", list.Body.String())
	}
	if !strings.Contains(list.Body.String(), paired.DeviceID) {
		t.Fatalf("device list %s does not show paired device %s", list.Body.String(), paired.DeviceID)
	}
}

func TestReadScopeReadsButDoesNotWrite(t *testing.T) {
	h := newTestHandler(t)
	paired := pairDevice(t, h, issuePairingCode(t, h, "read"))

	expectStatus(t, call(t, h, http.MethodGet, "/v1/agents", paired.Token, ""), http.StatusOK)
	rec := call(t, h, http.MethodPost, "/v1/agents", paired.Token, `{}`)
	expectStatus(t, rec, http.StatusForbidden)
	if got := errorCode(t, rec); got != "forbidden" {
		t.Fatalf("error code = %q, want forbidden", got)
	}
}

func TestApproveOnlyScopeOnlyTouchesApprovals(t *testing.T) {
	h := newTestHandler(t)
	paired := pairDevice(t, h, issuePairingCode(t, h, "approve_only"))

	expectStatus(t, call(t, h, http.MethodGet, "/v1/approvals", paired.Token, ""), http.StatusOK)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/agents", paired.Token, ""), http.StatusForbidden)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/memories", paired.Token, ""), http.StatusForbidden)
}

func TestRevokedDeviceIsUnauthorized(t *testing.T) {
	h := newTestHandler(t)
	paired := pairDevice(t, h, issuePairingCode(t, h, ""))
	expectStatus(t, call(t, h, http.MethodGet, "/v1/agents", paired.Token, ""), http.StatusOK)

	expectStatus(t, call(t, h, http.MethodDelete, "/v1/devices/"+paired.DeviceID, testToken, ""), http.StatusNoContent)

	rec := call(t, h, http.MethodGet, "/v1/agents", paired.Token, "")
	expectStatus(t, rec, http.StatusUnauthorized)
	if got := errorCode(t, rec); got != "unauthorized" {
		t.Fatalf("error code = %q, want unauthorized", got)
	}
}

func TestPairingCodeCannotBeReused(t *testing.T) {
	h := newTestHandler(t)
	code := issuePairingCode(t, h, "")
	pairDevice(t, h, code)

	body := fmt.Sprintf(`{"code":%q,"device_name":"Tablet","platform":"android"}`, code)
	rec := call(t, h, http.MethodPost, "/v1/pair", "", body)
	expectStatus(t, rec, http.StatusUnauthorized)
	if got := errorCode(t, rec); got != "unauthorized" {
		t.Fatalf("error code = %q, want unauthorized", got)
	}
}

func TestPairingIsRateLimitedPerOrigin(t *testing.T) {
	h := newTestHandler(t)
	body := `{"code":"AAAAAAAA","device_name":"Phone","platform":"ios"}`
	for range pairAttemptsPerWindow {
		expectStatus(t, call(t, h, http.MethodPost, "/v1/pair", "", body), http.StatusUnauthorized)
	}
	rec := call(t, h, http.MethodPost, "/v1/pair", "", body)
	expectStatus(t, rec, http.StatusTooManyRequests)
	if got := errorCode(t, rec); got != "rate_limited" {
		t.Fatalf("error code = %q, want rate_limited", got)
	}
}

func TestPairingCodeRoutesNeedWriteScope(t *testing.T) {
	h := newTestHandler(t)
	expectStatus(t, call(t, h, http.MethodPost, "/v1/pairing-codes", "", ""), http.StatusUnauthorized)

	reader := pairDevice(t, h, issuePairingCode(t, h, "read"))
	expectStatus(t, call(t, h, http.MethodPost, "/v1/pairing-codes", reader.Token, ""), http.StatusForbidden)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/devices", reader.Token, ""), http.StatusForbidden)
}

func TestUnknownDeviceTokenIsUnauthorized(t *testing.T) {
	h := newTestHandler(t)
	rec := call(t, h, http.MethodGet, "/v1/agents", "not-a-device-token", "")
	expectStatus(t, rec, http.StatusUnauthorized)
}
