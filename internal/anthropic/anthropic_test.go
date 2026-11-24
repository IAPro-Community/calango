package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"calango/internal/gateway"
)

const testKey = "sk-ant-test-key-1234"

type fakeSecrets struct {
	value string
	err   error
}

func (f fakeSecrets) Reveal(_ context.Context, name string) (string, error) {
	if name != KeyName {
		return "", errors.New("unexpected vault item " + name)
	}
	return f.value, f.err
}

// sse writes one streaming event the way the Messages API frames it.
func sse(w io.Writer, typ, data string) {
	io.WriteString(w, "event: "+typ+"\ndata: "+data+"\n\n")
}

func fullStream(w io.Writer) {
	sse(w, "message_start", `{"type":"message_start","message":{"usage":{"input_tokens":12,"output_tokens":1}}}`)
	sse(w, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
	sse(w, "ping", `{"type":"ping"}`)
	sse(w, "content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Olá, "}}`)
	sse(w, "content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"mundo"}}`)
	sse(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
	sse(w, "message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":7}}`)
	sse(w, "message_stop", `{"type":"message_stop"}`)
}

func newServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestCompleteStreamsTextAndReportsTokens(t *testing.T) {
	var gotBody messagesRequest
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			t.Errorf("request = %s %s, want POST /v1/messages", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != testKey {
			t.Errorf("x-api-key = %q, want the vault key", got)
		}
		if got := r.Header.Get("anthropic-version"); got != apiVersion {
			t.Errorf("anthropic-version = %q, want %q", got, apiVersion)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("content-type", "text/event-stream")
		fullStream(w)
	})

	p, err := New(fakeSecrets{value: testKey}, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	usage, err := p.Complete(context.Background(), gateway.Request{
		Provider: Name, Model: "claude-test", Prompt: "diga oi", MaxOutputTokens: 64,
	}, func(c gateway.Chunk) { text.WriteString(c.Text) })
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	if text.String() != "Olá, mundo" {
		t.Errorf("streamed text = %q, want %q", text.String(), "Olá, mundo")
	}
	if usage != (gateway.Usage{TokensIn: 12, TokensOut: 7, Estimated: false}) {
		t.Errorf("usage = %+v, want exact 12 in / 7 out", usage)
	}
	if !gotBody.Stream || gotBody.Model != "claude-test" || gotBody.MaxTokens != 64 {
		t.Errorf("body = %+v, want stream=true model=claude-test max_tokens=64", gotBody)
	}
	if len(gotBody.Messages) != 1 || gotBody.Messages[0].Role != "user" || gotBody.Messages[0].Content != "diga oi" {
		t.Errorf("messages = %+v, want one user message with the prompt", gotBody.Messages)
	}
}

func TestCompleteHTTPErrorDoesNotLeakKey(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	})
	p, err := New(fakeSecrets{value: testKey}, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Complete(context.Background(), gateway.Request{Provider: Name, Model: "m", Prompt: "x", MaxOutputTokens: 1}, nil)
	if err == nil {
		t.Fatal("Complete succeeded on HTTP 401")
	}
	msg := err.Error()
	if !strings.Contains(msg, "401") || !strings.Contains(msg, "authentication_error") {
		t.Errorf("error = %q, want status and error type", msg)
	}
	if strings.Contains(msg, testKey) {
		t.Errorf("error %q contains the api key", msg)
	}
}

func TestCompleteStreamErrorKeepsPartialText(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		sse(w, "message_start", `{"type":"message_start","message":{"usage":{"input_tokens":5,"output_tokens":1}}}`)
		sse(w, "content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"parcial"}}`)
		sse(w, "error", `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`)
	})
	p, _ := New(fakeSecrets{value: testKey}, srv.URL)
	var text strings.Builder
	usage, err := p.Complete(context.Background(), gateway.Request{Provider: Name, Model: "m", Prompt: "x", MaxOutputTokens: 9}, func(c gateway.Chunk) {
		text.WriteString(c.Text)
	})
	if err == nil || !strings.Contains(err.Error(), "overloaded_error") {
		t.Fatalf("err = %v, want overloaded_error", err)
	}
	if text.String() != "parcial" {
		t.Errorf("text = %q, want the chunks sent before the error", text.String())
	}
	if usage.TokensIn != 5 {
		t.Errorf("usage.TokensIn = %d, want 5 reported before the error", usage.TokensIn)
	}
}

func TestCompleteFailsWhenStreamEndsWithoutMessageStop(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		sse(w, "content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"cortado"}}`)
	})
	p, _ := New(fakeSecrets{value: testKey}, srv.URL)
	_, err := p.Complete(context.Background(), gateway.Request{Provider: Name, Model: "m", Prompt: "x", MaxOutputTokens: 9}, nil)
	if err == nil || !strings.Contains(err.Error(), "message_stop") {
		t.Fatalf("err = %v, want a truncated-stream error", err)
	}
}

func TestCompleteMissingKeyMakesNoRequest(t *testing.T) {
	var hits atomic.Int32
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	})
	p, _ := New(fakeSecrets{err: errors.New("vault item not found")}, srv.URL)
	_, err := p.Complete(context.Background(), gateway.Request{Provider: Name, Model: "m", Prompt: "x", MaxOutputTokens: 9}, nil)
	if err == nil || !strings.Contains(err.Error(), "read api key") {
		t.Fatalf("err = %v, want a key-read error", err)
	}
	if hits.Load() != 0 {
		t.Errorf("server received %d requests without a key", hits.Load())
	}
}

func TestNewRejectsBadBaseURL(t *testing.T) {
	for _, base := range []string{"api.anthropic.com", "ftp://x", "https://"} {
		if _, err := New(fakeSecrets{}, base); err == nil {
			t.Errorf("New(%q) accepted a bad base URL", base)
		}
	}
}

// memLedger is the smallest gateway.Ledger the gateway needs.
type memLedger struct {
	limit   int64
	records []gateway.Record
}

func (m *memLedger) AgentTokenLimit(context.Context, string) (int64, error) { return m.limit, nil }
func (m *memLedger) MonthlyTokens(context.Context, string, time.Time, time.Time) (int64, error) {
	return 0, nil
}
func (m *memLedger) RecordUsage(_ context.Context, rec gateway.Record) error {
	m.records = append(m.records, rec)
	return nil
}

func TestCallThroughGatewayRecordsUsage(t *testing.T) {
	srv := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		fullStream(w)
	})
	p, err := New(fakeSecrets{value: testKey}, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ledger := &memLedger{limit: 1_000_000}
	gw, err := gateway.New(gateway.Config{
		Ledger:    ledger,
		Providers: []gateway.Provider{p},
		Now:       time.Now,
		NewID:     func(time.Time) (string, error) { return "usage-1", nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	usage, err := gw.Call(context.Background(), gateway.Caller{AgentID: "agent-1"}, gateway.Request{
		Provider: Name, Model: "claude-test", Prompt: "oi", MaxOutputTokens: 64,
	}, nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if usage.TokensIn != 12 || usage.TokensOut != 7 {
		t.Errorf("usage = %+v, want 12 in / 7 out", usage)
	}
	if len(ledger.records) != 1 {
		t.Fatalf("ledger has %d records, want 1", len(ledger.records))
	}
	rec := ledger.records[0]
	if rec.Provider != Name || rec.Model != "claude-test" || rec.TokensIn != 12 || rec.TokensOut != 7 {
		t.Errorf("record = %+v, want anthropic/claude-test 12/7", rec)
	}
	if rec.CostStatus != gateway.CostUnknown {
		t.Errorf("cost status = %q, want unknown with an empty price table", rec.CostStatus)
	}
}
