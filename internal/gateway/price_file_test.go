package gateway

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPriceTable(t *testing.T) {
	table, err := LoadPriceTable("")
	if err != nil || len(table.Models) != 0 {
		t.Fatalf("unset tariff = %+v, %v", table, err)
	}
	path := filepath.Join(t.TempDir(), "prices.json")
	if err := os.WriteFile(path, []byte(`{"version":"test-v1","currency":"USD","models":{"anthropic/default/m":{"input_per_million":1000000,"output_per_million":0}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	table, err = LoadPriceTable(path)
	if err != nil {
		t.Fatal(err)
	}
	p, ok := table.lookup("anthropic/default", "m")
	if !ok || p.InputPerMillion != 1000000 || p.OutputPerMillion != 0 || table.Version != "test-v1" {
		t.Fatalf("unexpected tariff: %+v", table)
	}
	if _, ok := table.lookup("anthropic/other", "m"); ok {
		t.Fatal("tariff must not leak across accounts")
	}
}

func TestLoadPriceTableRejectsIncompleteOrInvalidTariff(t *testing.T) {
	for _, data := range []string{
		`{}`,
		`{"version":"v1","currency":"BRL","models":{}}`,
		`{"version":"v1","currency":"USD","models":{"p/m":{"input_per_million":1}}}`,
		`{"version":"v1","currency":"USD","models":{"p/m":{"input_per_million":-1,"output_per_million":1}}}`,
		`{"version":"v1","currency":"USD","models":{}} {}`,
	} {
		path := filepath.Join(t.TempDir(), "prices.json")
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadPriceTable(path); err == nil {
			t.Fatalf("accepted invalid tariff %s", data)
		}
	}
}
