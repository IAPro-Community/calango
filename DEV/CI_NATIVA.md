# CI nativa — Linux, Windows e macOS

Issue #120. Workflow: `.github/workflows/ci-nativa.yml`. Complementa o workflow `ci` (quality-gate, lint, contratos, imagens e builds cruzados), que continua igual.

**Passar nesta CI não homologa Linux nem macOS.** A CI prova que o código compila, que os testes automatizados passam e que o daemon sobe e encerra em runners limpos de cada sistema. A homologação exige os gates manuais descritos no fim deste documento.

## Matriz

| Runner | Obrigatório | Go | Node |
|---|---|---|---|
| `ubuntu-latest` | sim | versão do `go.mod` | 22.12.0 (piso de `engines.node`) |
| `windows-latest` | sim | versão do `go.mod` | 22.12.0 |
| `macos-latest` | sim | versão do `go.mod` | 22.12.0 |

Cada plataforma é um job `Nativo <runner>` com `fail-fast: false`: uma falha não cancela as outras, e o resultado de cada sistema fica visível. O campo `obrigatoria` da matriz permite rebaixar uma plataforma para informativa (`continue-on-error`) sem mexer no gate; hoje as três são obrigatórias.

## O que roda em cada sistema

1. `npm ci` com `ELECTRON_SKIP_BINARY_DOWNLOAD=1`.
2. `go test ./... -race -count=1 -json`, com `CGO_ENABLED=1` (o detector de corrida exige cgo; os três runners têm compilador C).
3. `npm run test:main -w apps/desktop`: compila o processo principal do Electron e roda os testes de `apps/desktop/test`.
4. `node scripts/daemon-smoke.mjs`: compila `cmd/calango` e verifica o ciclo de vida do daemon (detalhes abaixo).
5. `node --test scripts/signature-check.test.mjs`: verificação de assinatura, pulada sem artefato assinado.

O checkout desativa `core.autocrlf` para que o Windows receba os arquivos com LF, como nos outros sistemas.

### Smoke do daemon

`scripts/daemon-smoke.mjs` é Node puro, sem shell específico de plataforma. Roda localmente com `node scripts/daemon-smoke.mjs` (compila com o Go do PATH) ou `node scripts/daemon-smoke.mjs --binary <caminho>`. Ele:

- cria um data dir temporário com espaço e acento no caminho e gera uma `CALANGO_VAULT_KEY` descartável, sem tocar no keyring do sistema;
- executa `calango print-token` e confere que o token não está vazio e que o banco foi criado no data dir;
- no Linux e no macOS, confere permissão 0600 no banco e 0700 no data dir; no Windows registra que a verificação não se aplica (ACL herdada do perfil);
- sobe o daemon numa porta livre de loopback e espera `/healthz`;
- confere que o token autentica `GET /v1/agents` e que a chamada sem token recebe 401;
- encerra o processo e confere que ele saiu dentro do prazo e liberou a porta. No Unix envia SIGTERM e exige saída com código 0 (shutdown pelo contexto cancelado). No Windows, `child.kill()` usa TerminateProcess: o teste prova que o processo morre e solta a porta, não que o shutdown é gracioso;
- confere que o token não aparece na saída do daemon.

## Testes isolados (skips com motivo)

| Teste | Condição para rodar | Motivo do skip na CI | Gate de homologação |
|---|---|---|---|
| `internal/vault.TestOSKeyringRoundTrip` | `CALANGO_TEST_OS_KEYRING=1` | Runners não têm sessão de usuário com keyring desbloqueado; no Linux não há Secret Service. Um resultado ali não provaria o comportamento real. | Rodar `CALANGO_TEST_OS_KEYRING=1 go test ./internal/vault -run TestOSKeyring` numa sessão real de cada sistema (Keychain, Credential Manager, Secret Service). |
| `scripts/signature-check.test.mjs` | `CALANGO_SIGNED_ARTIFACT=<caminho>` | A cadeia de assinatura e notarização ainda não existe. | Apontar a variável para o binário ou app assinado: `codesign --verify --strict` no macOS, `Get-AuthenticodeSignature` = `Valid` no Windows. No Linux o teste é pulado: a verificação é por checksum e assinatura destacada. |
| `internal/harness` (testes com `fakeCodex`) | sistema diferente de Windows | O codex falso é um script `sh`. | Cobertura do harness no Windows depende de um fake multiplataforma (pendente). |
| `internal/auth.TestImportFileToken*` (importação do `api-token` legado: move, conflito, importação interrompida) | sistema diferente de Windows | Bug real, descrito abaixo. | Corrigir a checagem de permissão no Windows e remover o skip. |

## Bugs encontrados pela matriz

**Windows: o `api-token` legado nunca é importado.** `internal/auth/token.go` (`readToken`) recusa o arquivo quando `Mode().Perm()&0o077 != 0`. No Windows, `os.Stat` reporta 0666 para qualquer arquivo gravável, porque as ACLs não viram bits POSIX. Resultado: um nó Windows que ainda tenha `<data dir>/api-token` de uma versão anterior não sobe (`local api token: api token import: ... readable by other users; run chmod 600`), e a instrução de `chmod` não resolve nada no Windows. A CI nativa pulou os três testes afetados no Windows com esse motivo; o teste de recusa por permissão continua rodando. A correção mexe numa checagem de segurança e fica para revisão própria: no Windows, validar a ACL do arquivo ou confiar no perfil do usuário, com teste específico.

Os testes de vault com keyring falso (`internal/vault/keys_test.go`, `bootstrap_test.go`) continuam rodando em todos os sistemas: eles cobrem a lógica de fallback e a recusa sem keyring.

## Como ler os resultados

- **Resumo do job:** cada `Nativo <runner>` escreve no resumo da execução uma tabela com o resultado de cada etapa, a contagem de testes Go (passaram, falharam, pulados), a lista de falhas, cada teste pulado com o motivo e a tabela do smoke do daemon.
- **Artefatos:** `ci-nativa-<runner>` guarda `go-test.json` (saída completa de `go test -json`) e `smoke.md`, por 30 dias.
- **Gate:** o job `ci-nativa-gate` falha se qualquer plataforma obrigatória falhar. Para impedir a promoção de um build com regressão, os mantenedores precisam marcar `ci-nativa-gate` como check obrigatório no ruleset de `develop` e `main`, ao lado do `quality-gate`. Essa configuração do GitHub é feita e verificada à parte (GOVERNANCE.md).

Para reproduzir localmente: `go test ./... -race -count=1`, `npm run test:main -w apps/desktop` e `node scripts/daemon-smoke.mjs`. O resumo dos testes Go sai com `go test ./... -json > go-test.json; node scripts/go-test-summary.mjs go-test.json`.

## O que falta para homologar

- Assinatura e notarização dos instaladores e do binário (macOS e Windows), com a verificação de `scripts/signature-check.test.mjs` rodando no pipeline de release.
- Keyring real verificado em sessão de usuário de cada sistema (`CALANGO_TEST_OS_KEYRING=1`), incluindo Secret Service bloqueado no Linux.
- Shutdown gracioso do daemon no Windows: hoje o desktop e o smoke encerram com TerminateProcess; não há sinal de console equivalente ao SIGTERM.
- Correção da importação do `api-token` legado no Windows (seção de bugs acima).
- Fake do codex multiplataforma para cobrir o harness no Windows.
- Instalação limpa, atualização e desinstalação do app Electron empacotado em cada sistema; a CI testa só o processo principal compilado, sem abrir janela.
- Arquiteturas: `ubuntu-latest` e `windows-latest` rodam em amd64 e `macos-latest` em arm64. Linux e Windows em arm64 e macOS em amd64 continuam cobertos só pelo build cruzado do workflow `ci`.
