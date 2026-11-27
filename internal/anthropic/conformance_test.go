package anthropic

import (
	"strings"
	"testing"

	"calango/internal/gateway"
)

func TestUsageConformance(t *testing.T) {
	req := gateway.Request{Prompt: "abcd", MaxOutputTokens: 100}
	for _, tc := range []struct {
		name, body string
		in, out    int64
		estimated  bool
		failed     bool
	}{
		{"measured zero", "data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":0}}}\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":0}}\ndata: {\"type\":\"message_stop\"}\n", 0, 0, false, false},
		{"missing terminal usage", "data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":5,\"output_tokens\":1}}}\ndata: {\"type\":\"message_stop\"}\n", 5, 100, true, false},
		{"interrupted", "data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":5,\"output_tokens\":1}}}\n", 5, 100, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readStream(strings.NewReader(tc.body), req, nil)
			if (err != nil) != tc.failed || got.TokensIn != tc.in || got.TokensOut != tc.out || got.Estimated != tc.estimated {
				t.Fatalf("usage=%+v err=%v", got, err)
			}
		})
	}
}
