package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthcheckURL_ProbesLoopbackForWildcardHosts(t *testing.T) {
	cases := map[string]string{
		"0.0.0.0:7317":   "http://127.0.0.1:7317/healthz",
		":7317":          "http://127.0.0.1:7317/healthz",
		"[::]:7317":      "http://[::1]:7317/healthz",
		"127.0.0.1:9000": "http://127.0.0.1:9000/healthz",
	}
	for addr, want := range cases {
		got, err := healthcheckURL(addr)
		if err != nil {
			t.Fatalf("healthcheckURL(%q): %v", addr, err)
		}
		if got != want {
			t.Errorf("healthcheckURL(%q) = %q, want %q", addr, got, want)
		}
	}
	if _, err := healthcheckURL("no-port"); err == nil {
		t.Error("healthcheckURL accepted an address without a port")
	}
}

func TestHealthcheck_SucceedsOnlyOn200(t *testing.T) {
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" || r.Header.Get("Authorization") != "" {
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.WriteHeader(status)
	}))
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")

	if err := healthcheck(context.Background(), addr, srv.Client()); err != nil {
		t.Fatalf("healthy daemon: %v", err)
	}
	status = http.StatusServiceUnavailable
	if err := healthcheck(context.Background(), addr, srv.Client()); err == nil {
		t.Fatal("healthcheck passed on 503")
	}
	srv.Close()
	if err := healthcheck(context.Background(), addr, srv.Client()); err == nil {
		t.Fatal("healthcheck passed with the daemon down")
	}
}
