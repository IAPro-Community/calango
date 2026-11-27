package vault

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

// osKeyringTestEnv liga os testes que gravam no keyring real do sistema.
const osKeyringTestEnv = "CALANGO_TEST_OS_KEYRING"

// requireOSKeyring pula o teste fora de uma máquina de homologação. Na CI os runners não
// têm sessão de usuário com keyring desbloqueado (no Linux nem há Secret Service), então
// o resultado não provaria o comportamento real. Ver DEV/CI_NATIVA.md.
func requireOSKeyring(t *testing.T) {
	t.Helper()
	if os.Getenv(osKeyringTestEnv) != "1" {
		t.Skip("keyring real do sistema não exercitado: defina " + osKeyringTestEnv +
			"=1 numa sessão de usuário com keyring desbloqueado (gate de homologação, DEV/CI_NATIVA.md)")
	}
}

func TestOSKeyringRoundTrip(t *testing.T) {
	requireOSKeyring(t)
	const service = "calango-test"
	account := fmt.Sprintf("vault-roundtrip-%d", time.Now().UnixNano())
	t.Cleanup(func() { _ = keyring.Delete(service, account) })

	kr := OSKeyring{}
	if _, err := kr.Get(service, account); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("Get de entrada inexistente = %v, want ErrKeyNotFound", err)
	}
	if err := kr.Set(service, account, "valor-de-teste"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := kr.Get(service, account)
	if err != nil || got != "valor-de-teste" {
		t.Fatalf("Get = %q, %v", got, err)
	}
}
