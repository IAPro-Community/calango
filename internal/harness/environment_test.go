package harness

import (
	"reflect"
	"testing"
)

func TestSafeCLIEnvDoesNotInheritDaemonOrProviderSecrets(t *testing.T) {
	in := []string{
		"PATH=/bin", "HOME=/home/example", "SystemRoot=C:\\Windows", "CODEX_HOME=/account",
		"CALANGO_VAULT_KEY=secret", "CALANGO_VAULT_KEY_FILE=/secret",
		"calango_vault_key_previous=secret", "CALANGO_MCP_TOKEN=previous-turn",
		"OPENAI_API_KEY=secret", "ANTHROPIC_API_KEY=secret", "GITHUB_TOKEN=secret",
		"NODE_OPTIONS=--require=unsafe", "LD_PRELOAD=unsafe", "HTTPS_PROXY=http://secret",
		"INVALID",
	}
	want := []string{"PATH=/bin", "HOME=/home/example", "SystemRoot=C:\\Windows", "CODEX_HOME=/account"}
	if got := safeCLIEnv(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected inherited environment: %v", got)
	}
	if got := safeCLIEnv(nil); got == nil {
		t.Fatal("nil environment would cause exec to inherit the parent")
	}
}
