package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"calango/internal/core"
	"calango/internal/events"
	"calango/internal/maestro"
)

type maestroTestRunner struct {
	calls      int
	failVerify bool
}

func (r *maestroTestRunner) Run(_ context.Context, _ string, _ []string, executable string, args ...string) ([]byte, error) {
	r.calls++
	if r.failVerify && strings.Contains(executable, filepath.Join(".bin", "orquestrador-maestro")) && len(args) > 0 && args[0] == "verify" {
		return nil, errors.New("verification failed")
	}
	return []byte("ok"), nil
}

type maestroTestRegistry []string

func (r maestroTestRegistry) Versions(context.Context) ([]string, error) { return r, nil }

func TestMaestroRoutesPersistSettingsAndSwitchVerifiedVersion(t *testing.T) {
	runner := &maestroTestRunner{}
	manager, err := maestro.New(maestro.Config{
		DataDir: t.TempDir(), Runner: runner, Registry: maestroTestRegistry{"1.1.1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	hub := events.NewHub()
	sub := hub.Subscribe("owner")
	defer sub.Close()
	h := &handler{maestro: manager, events: hub, ownerID: "owner"}
	mux := http.NewServeMux()
	h.registerMaestro(mux)
	if _, err := manager.CheckUpdate(context.Background()); err != nil {
		t.Fatal(err)
	}
	serve := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request = request.WithContext(withPrincipal(request.Context(), principal{scope: core.ScopeFull}))
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		return recorder
	}

	patchRec := serve(http.MethodPatch, "/v1/maestro/settings", `{"auto_update":false}`)
	if patchRec.Code != http.StatusOK || !strings.Contains(patchRec.Body.String(), `"auto_update":false`) {
		t.Fatalf("settings response %d: %s", patchRec.Code, patchRec.Body.String())
	}
	getRec := serve(http.MethodGet, "/v1/maestro", "")
	if getRec.Code != http.StatusOK ||
		!strings.Contains(getRec.Body.String(), `"update_available":{"version":"1.1.1"}`) ||
		!strings.Contains(getRec.Body.String(), `"rollback_available":false`) {
		t.Fatalf("initial status response %d: %s", getRec.Code, getRec.Body.String())
	}

	installRec := serve(http.MethodPost, "/v1/maestro/install", `{"version":"1.1.1"}`)
	if installRec.Code != http.StatusOK || !strings.Contains(installRec.Body.String(), `"version":"1.1.1"`) {
		t.Fatalf("install response %d: %s", installRec.Code, installRec.Body.String())
	}
	select {
	case event := <-sub.C:
		if event.Type != "maestro.updated" {
			t.Fatalf("event type = %s", event.Type)
		}
	default:
		t.Fatal("expected maestro.updated event")
	}
	noRollback := serve(http.MethodPost, "/v1/maestro/rollback", "")
	if noRollback.Code != http.StatusConflict || !strings.Contains(noRollback.Body.String(), `"no_previous_version"`) {
		t.Fatalf("rollback without previous release = %d: %s", noRollback.Code, noRollback.Body.String())
	}

	if err := manager.Install(context.Background(), "1.1.2"); err != nil {
		t.Fatal(err)
	}
	getRec = serve(http.MethodGet, "/v1/maestro", "")
	if getRec.Code != http.StatusOK || !strings.Contains(getRec.Body.String(), `"rollback_available":true`) {
		t.Fatalf("rollback status response %d: %s", getRec.Code, getRec.Body.String())
	}
	rollbackRec := serve(http.MethodPost, "/v1/maestro/rollback", "")
	if rollbackRec.Code != http.StatusOK || !strings.Contains(rollbackRec.Body.String(), `"version":"1.1.1"`) {
		t.Fatalf("rollback response %d: %s", rollbackRec.Code, rollbackRec.Body.String())
	}
}
