# Revisão de originalidade e licenças

Issue #50 (X-License, épico #37). Revisão feita em 25/11/2026 sobre o `develop` em `5a11c81`. Não é parecer jurídico: a revisão jurídica de `CLA.md`, `COMMERCIAL.md` e dos arquivos derivados de Apache-2.0 continua na #42.

## Escopo e método

1. **Arquivos derivados do AgenticOS (ADR 0011).** Listar todo arquivo com o cabeçalho `Derived from vstorm-co/agenticos`, conferir que o caminho citado existe no commit fixado `a4b0b3c` e que o arquivo é Go reescrito, não cópia do Python.
2. **Código portado sem cabeçalho.** Comparar o upstream em `a4b0b3c` (lido, não copiado, só para a revisão) com o código do Calango nas áreas da tabela da ADR 0011: auditoria, cofre, aprovações, rotinas, memória, consumo e MCP.
   - Literais de texto com 18 ou mais caracteres: 331 do upstream, procurados em todo `.go`, `.ts` e `.tsx`.
   - Nomes de função do upstream convertidos para camelCase, procurados no pacote Go correspondente.
3. **Referências proibidas (ADR 0006).** Procurar menções a agentes.chat e a outros projetos de referência no código.
4. **Atribuições.** Cruzar o `NOTICE` com o que de fato vai para o usuário:
   - módulos Go linkados em `cmd/calango` para linux, windows e darwin (`go list -deps`);
   - pacotes npm embutidos no bundle do renderer;
   - arquivos do instalador Windows (PR #149).

## Resultados

### Arquivos derivados

| Arquivo | Upstream (a4b0b3c) | Situação |
|---|---|---|
| `internal/audit/chain.go` | `backend/app/core/audit.py`, `backend/app/services/audit.py` | cabeçalho ok, caminhos existem |
| `internal/audit/chain_test.go` | `backend/app/services/audit.py` | cabeçalho ok; cenários reescritos |
| `internal/store/audit.go` | `backend/app/core/audit.py`, `backend/app/services/audit.py` | cabeçalho ok |
| `internal/store/audit_test.go` | idem | cabeçalho ok; cenários reescritos para SQLite |
| `internal/vault/envelope.go`, `keys.go`, `service.go` | `backend/app/core/vault.py` | cabeçalho ok |

`third_party/agenticos/LICENSE` e `NOTICE` são idênticos aos do upstream em `a4b0b3c`.

### Código portado sem cabeçalho

Nenhum indício.
- **Literais:** só duas mensagens de erro da verificação da cadeia coincidem, e elas estão em `internal/audit/chain.go` e `chain_test.go`, que já têm cabeçalho. A outra coincidência é o cabeçalho HTTP `application/json, text/event-stream`, que é do protocolo MCP e não é texto autoral.
- **Nomes de função:** as coincidências são genéricas (`request`, `call_tool`, `list_approvals`, `master_key`). As de cofre e auditoria caem em arquivos já marcados.
- **Áreas sem cabeçalho:** aprovações, rotinas, memória, consumo e MCP não mostram estrutura nem texto do upstream.

### Referências proibidas

Nenhuma menção a agentes.chat ou a outros projetos de referência em `.go`, `.ts`, `.tsx`, `.mjs`, `.css` ou `.html`, fora dos cabeçalhos exigidos pela ADR 0011.

### Atribuições: o que estava faltando

1. **Textos de licença dos módulos Go.** O `NOTICE` listava os módulos, mas o repositório não trazia os textos. MIT, BSD-2 e BSD-3 exigem reproduzir o aviso e a licença na distribuição binária. Agora os textos ficam em `third_party/go/<módulo>/`, incluindo licenças de código vendorizado dentro dos módulos:
   - musl em `modernc.org/libc`;
   - `honnef.co/go/netdb` em `modernc.org/libc`;
   - `internal/shellescape` em `go-keyring`.
2. **Go toolchain.** O runtime do Go entra no binário e não aparecia no `NOTICE`. Licença e concessão de patentes em `third_party/go/go-toolchain/`.
3. **Módulos que faltavam no `NOTICE`.** Os dois entram só nos builds windows e darwin, via `modernc.org/libc`:
   - `github.com/mattn/go-isatty` v0.0.20 (MIT);
   - `github.com/ncruces/go-strftime` v0.1.9 (MIT).
4. **React no renderer.** O Vite embute `react`, `react-dom` e `scheduler` (MIT) em `dist/renderer` e o minificador remove os comentários de licença. Textos agora em `third_party/npm/`, com a lista no `NOTICE`.

O instalador Windows do PR #149 já copia `LICENSE`, `NOTICE` e `third_party/` para `resources/legal/`. Então, quando os dois PRs entrarem, os textos novos vão junto com o app. O electron-builder também inclui `LICENSE.electron.txt` e `LICENSES.chromium.html` (conferido no `win-unpacked`).

### Fora do pacote

- O site carrega Bricolage Grotesque, Instrument Sans e JetBrains Mono do Google Fonts (SIL OFL 1.1). As fontes não são redistribuídas pelo repositório nem pelo app.
- As dependências npm de build e teste (Vite, electron-builder, Playwright, Vitest) não vão no app. Suas licenças foram avaliadas nos PRs que as adicionaram.

## Como manter

`npm run licenses:check` (no `npm run quality` e no job `desktop` da CI) roda `scripts/third-party-licenses.mjs`. Ele falha quando:
- um módulo Go linkado ou um pacote do renderer não tem texto em `third_party/`;
- um texto mudou em relação ao módulo ou pacote;
- sobrou pasta de algo que saiu;
- o `NOTICE` não cita módulo e versão.

Para corrigir, rode `node scripts/third-party-licenses.mjs --write` e atualize o `NOTICE` à mão (copyright e licença). A lista de pacotes do renderer fica no script e precisa ser atualizada quando o renderer importar um pacote novo em tempo de execução.

## Limitações

- A comparação com o upstream é heurística (literais e nomes) e cobre o backend Python das áreas da ADR 0011. Não substitui a revisão humana de PRs de porte, que a ADR 0011 já exige.
- O pacote do Orquestrador Maestro é instalado pelo npm em tempo de uso, fora do binário, e continua coberto pela ADR 0009 e pelo `NOTICE`.
- Builds com CGO e outras arquiteturas não foram listados; o script usa `amd64` e `CGO_ENABLED=0`, como o instalador e a imagem.
