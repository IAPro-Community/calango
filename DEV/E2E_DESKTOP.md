# E2E do desktop (Playwright Electron)

Issue #119. Os testes abrem o app desktop compilado (`apps/desktop/dist`) com o daemon real (`bin/calango`) e um provider fake determinístico. Nenhum teste usa o perfil, o keyring ou uma conta real.

## Como rodar

```bash
npm ci
npm run desktop:build
xvfb-run -a npm run desktop:e2e
```

Sem `xvfb-run`, o Electron abre janelas na sessão gráfica atual. A CI roda o job `desktop-e2e` em `ubuntu-latest` com Xvfb.

## Isolamento

Cada teste recebe um ambiente novo, criado pela fixture `apps/desktop/e2e/app.ts`:

- `HOME`, `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `XDG_CACHE_HOME` e `CODEX_HOME` em um diretório temporário, apagado no fim;
- `CALANGO_DATA_DIR` próprio, com banco SQLite novo;
- `CALANGO_VAULT_KEY` aleatória, então o daemon não toca o keyring do sistema;
- `CALANGO_ADDR` em uma porta livre de `127.0.0.1`;
- locale `en-US`; os textos esperados vêm de `packages/i18n/messages.json`.

## Provider e conector fake

- `e2e/fixtures/bin/codex` coloca `e2e/fixtures/fake-codex.mjs` no `PATH` no lugar da CLI do Codex. `codex login status` sempre responde logado. `codex exec` lê o prompt e responde um texto fixo. A última linha do prompt com marcador escolhe o comportamento: `[e2e:slow]` fica rodando até ser encerrado (cancelamento) e `[e2e:tool]` chama a ferramenta `create_note` pelo endpoint MCP da execução e espera a aprovação.
- `e2e/fixtures/fake-mcp.mjs` é um servidor MCP stdio mínimo com uma ferramenta de escrita (`create_note`). O teste de aprovação registra o conector e a permissão `write` pela API local, porque ainda não há tela para conectores.

## Fluxos cobertos

| Teste | O que verifica |
|---|---|
| onboarding | conta Codex e agente criados pela interface; ponte `window.calango` só com os seis métodos esperados; sem `require`, `process` ou `ipcRenderer` no renderer; IPC rejeita argumento inválido; `window.open` e navegação para fora de `app://calango` são bloqueados |
| chat e uso | mensagem enviada, resposta do provider fake exibida, execução `done` e linha de uso com tokens |
| cancelamento | execução lenta parada pelo botão; status `stopped` e sem resposta |
| aprovação | escrita pelo conector deixa a execução em `waiting_approval`; o item Aprovações do menu passa a descrever 1 pendente sem mudar o nome acessível; aprovação pela conversa conclui a execução com o resultado da ferramenta; fila e contador vazios depois |
| tema | tema claro escolhido em Configurações aplica na hora e continua depois de recarregar; "seguir o sistema" acompanha o `prefers-color-scheme` |
| terminal | sessão PTY aberta pela tela, comando executado no shell com a saída na tela, sessão encerrada |
| rotinas | rotina por intervalo criada, pausada e retomada |
| queda do daemon | `SIGKILL` no daemon; app mostra "tentando de novo", sobe outro processo, mantém os dados e a conversa funciona com o token novo |
| segredos | chave Anthropic fake não aparece no DOM, no `localStorage`, no `sessionStorage`, na listagem do cofre nem nos logs; a chave do cofre e o token da API também não aparecem nos logs |

## Falhas

Só falhas guardam evidência em `apps/desktop/e2e-results/`: trace do Playwright (`trace.zip`), screenshot e o log do processo Electron (inclui a saída do daemon). Na CI esses arquivos sobem como artefato `desktop-e2e-falhas` por 7 dias. Todos os dados desses arquivos são fake. Para abrir um trace: `npx playwright show-trace <arquivo>`.

## Limitações

- Só Linux. A fixture acha o processo do daemon com `pgrep`, e o wrapper `codex` é um script shell. A matriz nativa completa fica para a #120, e o instalador Windows tem smoke próprio na #116.
- O provider é fake. A homologação com provider real é autorizada à parte, na #41.
- O daemon tenta a atualização automática do Orquestrador Maestro ao iniciar. No perfil descartável ela falha e só deixa a linha `maestro automatic update failed` no log, sem afetar os testes.
- O job `desktop-e2e` faz parte do `quality-gate` desde que a suíte ficou estável na CI.
