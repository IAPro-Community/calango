package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"calango/internal/skills"
)

type discoveryBody struct {
	Skills []struct {
		SourceID     string `json:"source_id"`
		Name         string `json:"name"`
		Origin       string `json:"origin"`
		Hash         string `json:"hash"`
		ImportStatus string `json:"import_status"`
		Confirm      bool   `json:"requires_license_confirmation"`
	} `json:"skills"`
	Roots []struct {
		Status string `json:"status"`
	} `json:"roots"`
}

func newSkillsTestHandler(t *testing.T) (http.Handler, string) {
	t.Helper()
	root := t.TempDir()
	for name, license := range map[string]string{"pdf": "license: MIT\n", "bare": ""} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		md := "---\nname: " + name + "\ndescription: test skill\n" + license + "---\nBody\n"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	lib, err := skills.New(skills.Config{
		Dir:   filepath.Join(t.TempDir(), "skills"),
		Roots: []skills.Root{{Origin: skills.OriginClaude, Path: root}},
		Now:   time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	gw := &blockingGateway{done: make(chan struct{})}
	h, _, _ := newTestEnvWithGateway(t, nil, gw, func() { close(gw.done) }, func(c *Config) { c.Skills = lib })
	return h, root
}

func discover(t *testing.T, h http.Handler, token string) discoveryBody {
	t.Helper()
	rec := call(t, h, http.MethodGet, "/v1/skill-imports/discover", token, "")
	expectStatus(t, rec, http.StatusOK)
	var d discoveryBody
	if err := json.Unmarshal(rec.Body.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestSkillImportRoutesLifecycle(t *testing.T) {
	h, _ := newSkillsTestHandler(t)
	d := discover(t, h, testToken)
	if len(d.Skills) != 2 || len(d.Roots) != 1 || d.Roots[0].Status != "ok" {
		t.Fatalf("discovery = %+v", d)
	}
	byName := map[string]int{}
	for i, s := range d.Skills {
		byName[s.Name] = i
		if s.Origin != "claude" || s.ImportStatus != "new" {
			t.Fatalf("skill = %+v", s)
		}
	}
	pdf, bare := d.Skills[byName["pdf"]], d.Skills[byName["bare"]]

	body := fmt.Sprintf(`{"source_id":%q,"hash":%q}`, pdf.SourceID, pdf.Hash)
	expectStatus(t, call(t, h, http.MethodPost, "/v1/skill-imports", testToken, body), http.StatusCreated)
	rec := call(t, h, http.MethodPost, "/v1/skill-imports", testToken, body)
	expectStatus(t, rec, http.StatusOK)
	var out struct {
		Result string `json:"result"`
		Skill  struct {
			ID string `json:"id"`
		} `json:"skill"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Result != "unchanged" || out.Skill.ID != pdf.SourceID {
		t.Fatalf("reimport = %s (%v)", rec.Body.String(), err)
	}

	bareBody := fmt.Sprintf(`{"source_id":%q,"hash":%q}`, bare.SourceID, bare.Hash)
	rec = call(t, h, http.MethodPost, "/v1/skill-imports", testToken, bareBody)
	expectStatus(t, rec, http.StatusConflict)
	if got := errorCode(t, rec); got != "license_confirmation_required" {
		t.Fatalf("code = %q", got)
	}
	confirmed := fmt.Sprintf(`{"source_id":%q,"hash":%q,"confirm_license":true}`, bare.SourceID, bare.Hash)
	expectStatus(t, call(t, h, http.MethodPost, "/v1/skill-imports", testToken, confirmed), http.StatusCreated)

	stale := fmt.Sprintf(`{"source_id":%q,"hash":"sha256:old"}`, pdf.SourceID)
	rec = call(t, h, http.MethodPost, "/v1/skill-imports", testToken, stale)
	expectStatus(t, rec, http.StatusConflict)
	if got := errorCode(t, rec); got != "skill_changed" {
		t.Fatalf("code = %q", got)
	}
	expectStatus(t, call(t, h, http.MethodPost, "/v1/skill-imports", testToken, `{"source_id":"x","hash":"y","path":"/etc"}`), http.StatusBadRequest)
	expectStatus(t, call(t, h, http.MethodPost, "/v1/skill-imports", testToken, `{"source_id":"0123456789abcdef0123456789abcdef","hash":"y"}`), http.StatusNotFound)

	rec = call(t, h, http.MethodGet, "/v1/skill-imports", testToken, "")
	expectStatus(t, rec, http.StatusOK)
	var list []struct {
		ID               string `json:"id"`
		LicenseConfirmed bool   `json:"license_confirmed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 2 {
		t.Fatalf("list = %s (%v)", rec.Body.String(), err)
	}
	for _, s := range discover(t, h, testToken).Skills {
		if s.ImportStatus != "imported" {
			t.Fatalf("after import %s status = %q", s.Name, s.ImportStatus)
		}
	}

	expectStatus(t, call(t, h, http.MethodDelete, "/v1/skill-imports/"+pdf.SourceID, testToken, ""), http.StatusNoContent)
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/skill-imports/"+pdf.SourceID, testToken, ""), http.StatusNotFound)
}

func TestSkillImportRoutesNeedTokenAndWriteScope(t *testing.T) {
	h, _ := newSkillsTestHandler(t)
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/v1/skill-imports/discover"},
		{http.MethodGet, "/v1/skill-imports"},
		{http.MethodPost, "/v1/skill-imports"},
		{http.MethodDelete, "/v1/skill-imports/abc"},
	} {
		expectStatus(t, call(t, h, route.method, route.path, "", ""), http.StatusUnauthorized)
		expectStatus(t, call(t, h, route.method, route.path, "wrong-token", ""), http.StatusUnauthorized)
	}

	reader := pairDevice(t, h, issuePairingCode(t, h, "read"))
	d := discover(t, h, reader.Token)
	expectStatus(t, call(t, h, http.MethodGet, "/v1/skill-imports", reader.Token, ""), http.StatusOK)
	body := fmt.Sprintf(`{"source_id":%q,"hash":%q}`, d.Skills[0].SourceID, d.Skills[0].Hash)
	rec := call(t, h, http.MethodPost, "/v1/skill-imports", reader.Token, body)
	expectStatus(t, rec, http.StatusForbidden)
	if got := errorCode(t, rec); got != "forbidden" {
		t.Fatalf("code = %q", got)
	}
	expectStatus(t, call(t, h, http.MethodDelete, "/v1/skill-imports/"+d.Skills[0].SourceID, reader.Token, ""), http.StatusForbidden)

	approver := pairDevice(t, h, issuePairingCode(t, h, "approve_only"))
	expectStatus(t, call(t, h, http.MethodGet, "/v1/skill-imports/discover", approver.Token, ""), http.StatusForbidden)
}
