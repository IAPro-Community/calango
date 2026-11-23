# Roadmap com tarefas

Cada tarefa tem ID (`F<fase>-T<nn>`), módulo, dependências e critério de pronto. Marque `[x]` ao terminar e registre no `WORKLOG.md`. Não comece uma fase antes de a anterior cumprir o critério de saída.

---

## Fase 0 — Fundação

Saída: `go vet ./...` e `go test ./...` passam; `calango` responde `/healthz`.

- [x] F0-T01 Esqueleto do repo, `go.mod`, `cmd/calango`, `internal/core`, `internal/api`.
- [x] F0-T02 Docs `DEV/`, ADRs, OpenAPI rascunho, tokens.
- [x] F0-T03 Instalar Go 1.22+ e rodar `go vet` / `go test`. Feito com Go 1.27.4 em `~/.local/opt/go`; vet, testes e healthz passam.
- [ ] F0-T04 CI (GitHub Actions): vet, test, build para linux/windows/darwin amd64+arm64.
- [ ] F0-T05 `internal/config`: lê variáveis `CALANGO_*` num único lugar, valida na subida.

## Fase 1 — MVP desktop local

Saída: no Windows, instalar o app, conversar com um agente, criar uma rotina diária, receber e aprovar um cartão de aprovação, ver o custo no painel de uso.

### Núcleo
- [ ] F1-T01 `internal/store`: SQLite (modernc), WAL, executor de migrações. Testes em memória. → `DATA_MODEL.md`
- [ ] F1-T02 Migrações iniciais: agents, conversations, messages, runs, run_steps, usage, audit_log. (dep: T01)
- [ ] F1-T03 `internal/gateway`: interface `Provider`, orçamento antes, uso depois, provider fake para testes. → `modules/gateway.md` (dep: T02)
- [ ] F1-T04 Primeiro provider: API Anthropic direta (HTTP, streaming). Chave vinda do cofre. (dep: T03, T10)
- [ ] F1-T05 `internal/harness`: interface de harness + adaptador para Claude Code CLI. → `modules/harness.md` (dep: T03)
- [ ] F1-T06 Agentes: CRUD, pausa global, orçamento mensal. → `modules/agents.md` (dep: T02)
- [ ] F1-T07 Conversas e runs: DM com agente, streaming de passos, parar uma resposta. → `modules/conversations.md` (dep: T03, T06)
- [ ] F1-T08 Aprovações: cartão, decisão, recibo, regras "aprovar sempre". → `modules/approvals.md` (dep: T07)
- [ ] F1-T09 Rotinas persistidas: cron + intervalo, retries, notify_on, janela de silêncio. → `modules/routines.md` (dep: T07)
- [ ] F1-T10 Cofre: Windows Credential Manager, macOS Keychain, Linux Secret Service. → `modules/vault.md`
- [ ] F1-T11 Memória: salvar, listar, editar, apagar; injeção no contexto do run. → `modules/memory.md` (dep: T07)
- [ ] F1-T12 Ferramentas via MCP: conectar servidor MCP, escopos por agente. → `modules/tools-mcp.md` (dep: T07, T08)
- [ ] F1-T13 API v1 conforme `api/openapi.yaml` + WebSocket `/v1/events`. (dep: T06–T09)
- [ ] F1-T14 Auth local: token do app desktop gerado na primeira subida, guardado no cofre. (dep: T10)
- [ ] F1-T16 Integração do Orquestrador Maestro (linha V1, npm). → `modules/maestro.md`, ADR 0009 (dep: T05, T13)

### Desktop
- [ ] F1-T20 `apps/desktop`: Electron + Vite + React + TS strict. → `modules/desktop-electron.md`
- [ ] F1-T21 Processo principal inicia e vigia o `calango`; reinicia se cair; mostra estado. (dep: T20)
- [ ] F1-T22 `packages/sdk-ts` gerado do OpenAPI; `packages/tokens` gerando CSS. (dep: T13)
- [ ] F1-T23 Telas: DMs, conversa com passos, cartão de aprovação, rotinas, agentes, uso, configurações. (dep: T21, T22)
- [ ] F1-T24 Bandeja, iniciar com o sistema, atalho global.
- [ ] F1-T25 Instalador Windows NSIS assinado + auto-update; `.deb` Linux; DMG macOS notarizado.
- [ ] F1-T26 i18n pt-BR, en, es desde a primeira tela.

## Fase 2 — Celular

Saída: fora de casa, aprovar pelo celular uma ação pedida por um agente no PC, e receber push.

- [ ] F2-T01 Pareamento: código de uso único, QR, troca por token de dispositivo, revogação. → `modules/pairing-auth.md`
- [ ] F2-T02 Transporte Tailscale (avaliar `tsnet` embutido) atrás de `internal/transport`. → ADR 0005
- [ ] F2-T03 `packages/sdk-dart` gerado; `ThemeData` gerado dos tokens.
- [ ] F2-T04 `apps/mobile` Flutter: parear, nós, conversas, aprovações, rotinas, tarefas. → `modules/mobile-flutter.md`
- [ ] F2-T05 Calango Cloud mínima: contas e dispositivos. → `modules/cloud.md`
- [ ] F2-T06 Push relay (APNs/FCM), só "tem novidade". → `modules/push.md`
- [ ] F2-T07 TestFlight e teste interno do Play.

## Fase 3 — VPS e primeira receita

Saída: com o PC desligado, uma rotina `run_on=auto` roda na VPS; a assinatura Pro é cobrada e libera o nó VPS.

- [ ] F3-T01 Imagem Docker oficial multi-arch + instalador de uma linha. → `DEPLOYMENT.md`
- [ ] F3-T02 Log de eventos e replicação entre nós. → `modules/nodes-sync.md`, ADR 0008
- [ ] F3-T03 Execução híbrida: `run_on` pc/vps/auto, troca de dono quando o PC cai.
- [ ] F3-T04 Licença assinada Ed25519, validação offline, tolerância, rebaixamento. → `modules/licensing-billing.md`
- [ ] F3-T05 Billing: Stripe (web/desktop) + compras das lojas; fonte única de plano na Cloud.
- [ ] F3-T06 Limites do plano aplicados no núcleo (`Entitlements`).
- [ ] F3-T07 Interface mostra nó offline: qual, desde quando, o que espera por ele.

## Fase 4 — ADE

Saída: a partir de uma tarefa, um agente cria worktree, altera código, roda testes e abre PR, no PC ou na VPS.

- [ ] F4-T01 Projetos: registrar repo, nó, agentes, memória `DEV/`. → `modules/ade.md`
- [ ] F4-T02 Worktree por tarefa, limpeza ao terminar.
- [ ] F4-T03 Terminal: PTY no `calango` (ConPTY no Windows) por WebSocket; xterm.js no desktop.
- [ ] F4-T04 Editor Monaco, diff, árvore de arquivos.
- [ ] F4-T05 GitHub: branch, PR, status do CI.
- [ ] F4-T06 Servidor de preview aberto no navegador interno.
- [ ] F4-T07 Importar skills de Claude Code e Codex.

## Fase 5 — Navegador e uso do computador

Saída: o agente preenche um formulário no navegador interno, a pessoa assume no meio e devolve.

- [ ] F5-T01 Abas com `WebContentsView`, perfis por agente. → `modules/browser.md`
- [ ] F5-T02 Agente dirige a aba por CDP; assumir/devolver sem recarregar.
- [ ] F5-T03 Detecção de barreiras (robô, login, vazio) com aviso ao agente.
- [ ] F5-T04 Navegador headless na VPS (chromedp) com a mesma interface.
- [ ] F5-T05 Uso do computador (apps desktop), desligado por padrão. → `modules/computer-use.md`
- [ ] F5-T06 Controle de tela pelo celular (pago), com aviso e auditoria.

## Fase 6 — Canais e times

Saída: dois membros e três agentes conversam num canal; o roteador escolhe quem responde.

- [ ] F6-T01 Canais, threads, menções. → `modules/channels-teams.md`
- [ ] F6-T02 Roteador (modelo barato, 0–2 agentes, nunca concede permissão).
- [ ] F6-T03 Membros, papéis, permissões por canal, aprovação delegada.
- [ ] F6-T04 Plano Team no billing.

## Fase 7 — Expansão

- [ ] F7-T01 Relay próprio (substitui Tailscale como opção).
- [ ] F7-T02 Canais externos: Telegram, WhatsApp, Slack, e-mail.
- [ ] F7-T03 Voz: ditado e resposta falada.
- [ ] F7-T04 Arquivos e documentos com permissões.
- [ ] F7-T05 Marketplace de rotinas e agentes.
