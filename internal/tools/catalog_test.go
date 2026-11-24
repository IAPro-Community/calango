package tools_test

import (
	"context"
	"errors"
	"testing"

	"calango/internal/core"
	"calango/internal/tools"
)

func TestCatalogReportsReadOnlyHints(t *testing.T) {
	ctx := context.Background()
	ts := remoteServer(t)
	f := newFixture(t)

	c, err := f.svc.Register(ctx, owner, tools.NewConnector{Name: "notes", Transport: core.TransportHTTP, URL: ts.URL})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	list, err := f.svc.Catalog(ctx, owner, c.ID)
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	readOnly := map[string]bool{}
	for _, info := range list {
		readOnly[info.Name] = info.ReadOnly
	}
	if len(readOnly) != 2 || !readOnly["read_note"] || readOnly["create_note"] {
		t.Fatalf("catalog read-only flags = %v, want read_note true and create_note false", readOnly)
	}
}

func TestCatalogReportsUnreachableConnectorAsUnavailable(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	c, err := f.svc.Register(ctx, owner, tools.NewConnector{Name: "down", Transport: core.TransportHTTP, URL: "http://127.0.0.1:9/mcp"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := f.svc.Catalog(ctx, owner, c.ID); !errors.Is(err, tools.ErrUnavailable) {
		t.Fatalf("catalog of unreachable connector = %v, want ErrUnavailable", err)
	}
}

func TestCatalogIsOwnerScoped(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)

	c, err := f.svc.Register(ctx, owner, tools.NewConnector{Name: "down", Transport: core.TransportHTTP, URL: "http://127.0.0.1:9/mcp"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := f.svc.Catalog(ctx, "someone-else", c.ID); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("catalog for another owner = %v, want ErrNotFound", err)
	}
}
