// Package anthropic is the gateway provider for the Anthropic Messages API. It
// streams a single-turn reply over HTTP. The API key comes from the vault for
// each call and is sent only in the x-api-key header of that request; it is
// never placed in an error, a log line or a run step.
package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"calango/internal/gateway"
)

const (
	// Name is the gateway provider name.
	Name = "anthropic"
	// KeyName is the vault item that holds the API key.
	KeyName = "anthropic-api-key"
	// DefaultBaseURL is the production API origin.
	DefaultBaseURL = "https://api.anthropic.com"

	apiVersion   = "2023-06-01"
	maxErrorBody = 4 << 10
	maxLine      = 1 << 20
)

// Secrets reads a vault item by name. The vault service implements it.
type Secrets interface {
	Reveal(ctx context.Context, name string) (string, error)
}

// Provider implements gateway.Provider over the Messages API.
type Provider struct {
	baseURL string
	keyName string
	secrets Secrets
	client  *http.Client
}

// New returns a provider that reads the default key item from secrets. baseURL
// is empty for the production API; tests point it at a local server.
func New(secrets Secrets, baseURL string) (*Provider, error) {
	return NewAccount(secrets, KeyName, baseURL)
}

// NewAccount returns a provider that reads the API key from the vault item
// keyName. Each provider account has its own item.
func NewAccount(secrets Secrets, keyName, baseURL string) (*Provider, error) {
	if secrets == nil {
		return nil, errors.New("anthropic: secrets source is required")
	}
	if keyName == "" {
		return nil, errors.New("anthropic: key item name is required")
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, fmt.Errorf("anthropic: invalid base URL %q", baseURL)
	}
	return &Provider{
		baseURL: strings.TrimRight(baseURL, "/"),
		keyName: keyName,
		secrets: secrets,
		// No overall timeout: streams can run long. The caller's context bounds them.
		client: &http.Client{},
	}, nil
}

// Name implements gateway.Provider.
func (p *Provider) Name() string { return Name }

type messagesRequest struct {
	Model     string    `json:"model"`
	MaxTokens int64     `json:"max_tokens"`
	Stream    bool      `json:"stream"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type usage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// event is the subset of Messages streaming events Calango reads.
type event struct {
	Type    string `json:"type"`
	Message *struct {
		Usage usage `json:"usage"`
	} `json:"message"`
	Delta *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Usage *usage    `json:"usage"`
	Error *apiError `json:"error"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Complete implements gateway.Provider. It streams text deltas to stream and
// returns the tokens reported by the API. Usage is marked estimated when the
// stream ends without both counts.
func (p *Provider) Complete(ctx context.Context, req gateway.Request, stream func(gateway.Chunk)) (gateway.Usage, error) {
	key, err := p.secrets.Reveal(ctx, p.keyName)
	if err != nil {
		return gateway.Usage{}, fmt.Errorf("anthropic: read api key: %w", err)
	}

	body, err := json.Marshal(messagesRequest{
		Model:     req.Model,
		MaxTokens: req.MaxOutputTokens,
		Stream:    true,
		Messages:  []message{{Role: "user", Content: req.Prompt}},
	})
	if err != nil {
		return gateway.Usage{}, fmt.Errorf("anthropic: encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return gateway.Usage{}, fmt.Errorf("anthropic: build request: %w", err)
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("accept", "text/event-stream")
	httpReq.Header.Set("anthropic-version", apiVersion)
	httpReq.Header.Set("x-api-key", key)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return gateway.Usage{}, fmt.Errorf("%w: anthropic: request failed: %w", gateway.ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return gateway.Usage{}, httpError(resp)
	}
	return readStream(resp.Body, stream)
}

// httpError describes a non-200 answer without echoing headers or the request.
// HTTP 429 and 5xx wrap gateway.ErrUnavailable: another provider may answer.
func httpError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	var env struct {
		Error *apiError `json:"error"`
	}
	err := fmt.Errorf("anthropic: HTTP %d", resp.StatusCode)
	if json.Unmarshal(raw, &env) == nil && env.Error != nil {
		err = fmt.Errorf("anthropic: HTTP %d %s: %s", resp.StatusCode, env.Error.Type, env.Error.Message)
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf("%w: %w", gateway.ErrUnavailable, err)
	}
	return err
}

// readStream parses the SSE body. Only `data:` lines carry JSON; the `event:`
// line repeats the type inside the JSON, so it is ignored.
func readStream(body io.Reader, stream func(gateway.Chunk)) (gateway.Usage, error) {
	var (
		in, out       int64
		sawStop       bool
		sawIn, sawOut bool
	)
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64<<10), maxLine)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var ev event
		if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &ev); err != nil {
			return gateway.Usage{TokensIn: in, TokensOut: out}, fmt.Errorf("anthropic: malformed stream event: %w", err)
		}
		switch ev.Type {
		case "message_start":
			if ev.Message != nil {
				if ev.Message.Usage.InputTokens > 0 {
					in, sawIn = ev.Message.Usage.InputTokens, true
				}
				if ev.Message.Usage.OutputTokens > 0 {
					out, sawOut = ev.Message.Usage.OutputTokens, true
				}
			}
		case "content_block_delta":
			if ev.Delta != nil && ev.Delta.Type == "text_delta" && ev.Delta.Text != "" && stream != nil {
				stream(gateway.Chunk{Text: ev.Delta.Text})
			}
		case "message_delta":
			// output_tokens here is cumulative for the whole message.
			if ev.Usage != nil {
				if ev.Usage.InputTokens > 0 {
					in, sawIn = ev.Usage.InputTokens, true
				}
				if ev.Usage.OutputTokens > 0 {
					out, sawOut = ev.Usage.OutputTokens, true
				}
			}
		case "message_stop":
			sawStop = true
		case "error":
			if ev.Error != nil {
				return gateway.Usage{TokensIn: in, TokensOut: out}, fmt.Errorf("%w: anthropic: stream error %s: %s", gateway.ErrUnavailable, ev.Error.Type, ev.Error.Message)
			}
			return gateway.Usage{TokensIn: in, TokensOut: out}, fmt.Errorf("%w: anthropic: stream error", gateway.ErrUnavailable)
		}
		if sawStop {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return gateway.Usage{TokensIn: in, TokensOut: out}, fmt.Errorf("%w: anthropic: read stream: %w", gateway.ErrUnavailable, err)
	}
	if !sawStop {
		return gateway.Usage{TokensIn: in, TokensOut: out}, fmt.Errorf("%w: anthropic: stream ended before message_stop", gateway.ErrUnavailable)
	}
	return gateway.Usage{TokensIn: in, TokensOut: out, Estimated: !(sawIn && sawOut)}, nil
}
