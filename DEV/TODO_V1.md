# TODO — Calango V1 pronta e funcional

Gerado a partir de toda a documentação `DEV/` em 2026-11-23. Escopo da V1: **fases 0 a 6** do [ROADMAP](SPECS/ROADMAP.md), mais a integração do [Orquestrador Maestro](ORQUESTRADOR_MAESTRO.md). Fase 7 (relay próprio, canais externos, voz, marketplace) fica **depois da V1**.

Legenda: `[ ]` pendente · `[x]` feito · `⛔` bloqueado por decisão do dono · `→` fonte na documentação.

Ordem de execução: cada bloco só começa quando o anterior cumpre o critério de saída do ROADMAP.

---

## 0. Decisões do dono (bloqueiam partes da V1)

- [x] D1 Código do Calango será **aberto** (decidido pelo dono em 2026-11-23).
- [x] D1a Licença: **AGPL-3.0** + licença comercial + CLA + política de marca (ADR 0010). Arquivos criados.
- [ ] D1b Revisão jurídica de `CLA.md`, `COMMERCIAL.md` e `TRADEMARKS.md`; preencher o nome legal do mantenedor e os contatos (`licenca@`, `seguranca@`, `conduta@`).
- [ ] ⛔ D2 Controle de tela remoto: Pro ou só Team. → PLANO §14, PRODUCT_HYBRID
- [ ] ⛔ D3 Primeiro harness do MVP: Claude Code, Codex ou API direta. → `SPECS/modules/harness.md`
- [ ] ⛔ D4 Domínio e nome do módulo Go (`calango` é placeholder). → HANDOFF
- [ ] ⛔ D5 Conferir licença do `tsnet` e limites do plano gratuito do Tailscale. → ADR 0005
- [ ] ⛔ D6 Auto-atualização do Maestro: confirmar padrão (sugerido: desligada). → ADR 0009

---

## Fase 0 — Fundação

- [x] Go 1.27.4 instalado em `~/.local/opt/go` (SHA-256 verificado contra go.dev).
- [x] `go vet ./...` limpo.
- [x] `go test ./...` passa (4 testes em `internal/core`).
- [x] `gofmt -l` sem saída.
- [x] `calango` sobe e `GET /healthz` responde 200; POST responde 405.
- [ ] F0-T04 CI no GitHub Actions: vet, test, build linux/windows/darwin em amd64 e arm64.
- [ ] F0-T05 `internal/config`: lê `CALANGO_*` em um lugar e valida na subida.
- [ ] Validador OpenAPI completo sobre `api/openapi.yaml` no CI.
- [ ] Validação de contrato: `tokens.json` gera CSS e ThemeData sem perda.

---

## Fase 1 — MVP desktop local (Windows primeiro)

### Núcleo
- [ ] F1-T01 `internal/store`: SQLite (`modernc.org/sqlite`), WAL, executor de migrações imutáveis, testes em memória. → DATA_MODEL, ADR 0002
- [ ] F1-T02 Migrações iniciais: agents, conversations, messages, runs, run_steps, usage, audit_log.
- [ ] F1-T03 `internal/gateway`: interface `Provider`, orçamento antes da chamada, uso depois, provider fake. → `modules/gateway.md`, ADR 0007
- [ ] F1-T04 Provider Anthropic direto (HTTP com streaming); chave vinda do cofre.
- [ ] F1-T05 `internal/harness` com adaptador do harness escolhido em D3. → `modules/harness.md`
- [ ] F1-T06 Agentes: CRUD, pausa global, limite mensal. → `modules/agents.md`
- [ ] F1-T07 Conversas e runs: DM, passos em streaming, parar, orientar, tentar com instruções. → `modules/conversations.md`
- [ ] F1-T08 Aprovações: cartão com ação exata, decisão, recibo, "aprovar sempre", expiração em 24 h. → `modules/approvals.md`
- [ ] F1-T09 Rotinas: cron e intervalo, retry com backoff, notify_on, janela de silêncio, perda de execução sem repetição. → `modules/routines.md`
- [ ] F1-T10 Cofre: Windows Credential Manager, macOS Keychain, Linux Secret Service; recusa texto puro. → `modules/vault.md`
- [ ] F1-T11 Memória: salvar, listar, editar, apagar; injeção com orçamento de tokens. → `modules/memory.md`
- [ ] F1-T12 MCP: conectar servidor, gateway MCP local por turno, escopos none/read/write. → `modules/tools-mcp.md`
- [ ] F1-T13 API v1 conforme `api/openapi.yaml` e WebSocket `/v1/events`.
- [ ] F1-T14 Token local do desktop, guardado no cofre.
- [ ] F1-T15 Pareamento e autenticação de dispositivo (versão local, sem celular ainda). → `modules/pairing-auth.md`
- [ ] F1-T16 Integração do Maestro: instalar linha V1, `verify`, vincular a agentes, cartão de atualização. → `modules/maestro.md`, ADR 0009
- [ ] F1-T17 Painel de uso e custo por agente, rotina, projeto e nó. → ADR 0007, DATA_MODEL `usage`
- [ ] F1-T18 Auditoria: escrita append-only de aprovações, ações e mudanças de permissão. → SECURITY

### Desktop (Electron)
- [ ] F1-T20 `apps/desktop`: Electron + Vite + React + TS strict; `contextIsolation`, `sandbox`, CSP estrita. → `modules/desktop-electron.md`
- [ ] F1-T21 Processo principal inicia e vigia o `calango`; reinicia com backoff; estado na bandeja.
- [ ] F1-T22 `packages/sdk-ts` gerado do OpenAPI; `packages/tokens` gerando CSS.
- [ ] F1-T23 Telas: barra lateral, conversas, cartão de aprovação, rotinas, agentes, uso, configurações (você, agentes, modelos, conexões, cofre), "Sobre" com crédito ao Maestro.
- [ ] F1-T24 Bandeja, iniciar com o sistema, atalho global para conversa rápida.
- [ ] F1-T25 Instalador Windows NSIS **assinado** + auto-update assinado; `.deb` Linux; DMG macOS notarizado.
- [ ] F1-T26 i18n: pt-BR, en e es desde a primeira tela; catálogo único em `packages/i18n`.
- [ ] Entregar o arquivo `NOTICE` dentro do instalador.

### Critério de saída da fase 1
- [ ] Instalar no Windows, conversar com um agente, criar rotina diária, aprovar uma ação, ver custo no painel.

---

## Fase 2 — Celular

- [ ] F2-T01 Pareamento por QR e código de uso único (5 min), token guardado no keychain. → `modules/pairing-auth.md`
- [ ] F2-T02 Transporte Tailscale atrás de `internal/transport`; avaliar `tsnet` embutido (depende de D5). → ADR 0005
- [ ] F2-T03 `packages/sdk-dart` gerado do OpenAPI; `ThemeData` gerado de `tokens.json`.
- [ ] F2-T04 `apps/mobile` Flutter: parear, nós, "Para você", conversas, aprovações, rotinas, tarefas. → `modules/mobile-flutter.md`
- [ ] F2-T05 Calango Cloud mínima: contas e dispositivos, Postgres, migrações. → `modules/cloud.md`
- [ ] F2-T06 Push relay APNs/FCM só com "tem novidade" e id opaco; teste que prova ausência de texto no payload. → `modules/push.md`
- [ ] F2-T07 TestFlight (iOS) e teste interno (Android).
- [ ] Mobile fica disponível na loja somente após revisão das diretrizes de controle remoto.

### Critério de saída da fase 2
- [ ] Aprovar pelo celular, fora de casa, uma ação pedida por agente no PC, em até 3 toques a partir do push.

---

## Fase 3 — VPS e primeira receita

- [ ] F3-T01 Imagem Docker multi-arquitetura + instalador de uma linha; compose e Caddy atualizados. → `DEPLOYMENT.md`
- [ ] F3-T02 Log de eventos e replicação entre nós (`events`, `peers`, comandos ao dono). → `modules/nodes-sync.md`, ADR 0008
- [ ] F3-T03 Execução híbrida: `run_on` pc, vps e auto; troca de dono quando o PC cai; volta ao PC no fim do run.
- [ ] F3-T04 Licença Ed25519 validada offline, tolerância de 7 dias, rebaixamento para Free sem parar nem apagar. → `modules/licensing-billing.md`
- [ ] F3-T05 Billing: Stripe (web e desktop), compras das lojas (RevenueCat ou direto), fonte única de plano na Cloud.
- [ ] F3-T06 `Entitlements` no núcleo: VPS, rotinas, projetos, controle de tela, push.
- [ ] F3-T07 Interface mostra nó offline: qual, desde quando, o que espera por ele.
- [ ] F3-T08 Atualização do Maestro automática (opcional, desligada por padrão) com rollback e `verify`. → `modules/maestro.md`
- [ ] Termos de uso e política de privacidade publicados antes da cobrança.

### Critério de saída da fase 3
- [ ] Com o PC desligado, rotina `auto` roda na VPS no horário; assinatura Pro cobrada e nó VPS liberado.

---

## Fase 4 — ADE

- [ ] F4-T01 Projetos: repositório, nó, agentes, conectores, `DEV/` do projeto. → `modules/ade.md`
- [ ] F4-T02 Worktree por tarefa, branch próprio, limpeza ao terminar (manter, PR ou descartar).
- [ ] F4-T03 Terminal: PTY no `calango` (`creack/pty`; ConPTY no Windows), WebSocket `/v1/pty/{id}`, xterm.js.
- [ ] F4-T04 Editor Monaco, árvore de arquivos, diff por worktree.
- [ ] F4-T05 GitHub: criar branch, abrir PR, ler status do CI. GitLab fica para depois da V1.
- [ ] F4-T06 Servidor de preview detectado e aberto no navegador interno (depende de F5-T01).
- [ ] F4-T07 Importar skills de `.claude/` e `.codex/`.
- [ ] Lista segura de comandos de shell; fora dela, aprovação.
- [ ] Projeto na VPS editável pelo desktop com o mesmo editor e terminal.

### Critério de saída da fase 4
- [ ] Duas tarefas no mesmo projeto rodam em paralelo em worktrees distintos e geram dois PRs.

---

## Fase 5 — Navegador e uso do computador

- [ ] F5-T01 Abas com `WebContentsView`; perfis persistentes por agente. → `modules/browser.md`
- [ ] F5-T02 Agente dirige a aba por CDP; assumir e devolver sem recarregar.
- [ ] F5-T03 Detecção de barreiras (verificação de robô, login, cookies, página vazia) com linha de aviso ao agente, sem citar o texto da página.
- [ ] F5-T04 Navegador headless na VPS (chromedp) com a mesma interface; imagem separada. → DEPLOYMENT, pendência
- [ ] F5-T05 Uso do computador (apps desktop), desligado por padrão, com lista de apps bloqueados. → `modules/computer-use.md`
- [ ] F5-T06 Controle de tela pelo celular (pago, depende de D2), com aviso no PC, auditoria e encerramento em menos de 1 s.
- [ ] Permissões de Acessibilidade e Gravação de Tela guiadas no macOS.

### Critério de saída da fase 5
- [ ] Agente preenche um formulário no navegador interno; a pessoa assume no meio e devolve.

---

## Fase 6 — Canais e times

- [ ] F6-T01 Canais, threads e menções. → `modules/channels-teams.md`
- [ ] F6-T02 Roteador: modelo barato escolhe 0 a 2 agentes; nunca concede permissão; escolha aparece no topo da resposta.
- [ ] F6-T03 Membros, papéis (dono, admin, membro, convidado), permissões por canal, aprovação delegada.
- [ ] F6-T04 Plano Team no billing; agente compartilhado entre pessoas.

### Critério de saída da fase 6
- [ ] Dois membros e três agentes conversam num canal; mensagem sem menção gera no máximo dois runs.

---

## Transversais (valem para todas as fases)

- [ ] Auditoria de segredos: teste de canário garante que nenhum segredo aparece em `run_steps`, logs ou API. → SECURITY
- [ ] Limites de API: corpo limitado, rate limit em `/v1/pair`.
- [ ] Backup e restauração do banco documentados e testados.
- [ ] Telemetria só com opt-in, sem conteúdo.
- [ ] Documentação de usuário: primeiro uso, pareamento, VPS, Maestro, privacidade.
- [ ] Revisão de acessibilidade (teclado, leitor de tela, contraste) no desktop, mobile e site.
- [x] Site: seção "Contribua com a V1" e rodapé com AGPL-3.0 (feito em 2026-11-23).
- [ ] Site: trocar "em breve" por preços reais quando definidos; hospedar; ícone simplificado para tamanhos pequenos (`DEV/BRAND/BRAND.md`).
- [ ] Publicar o repositório em `IAPro-Community/calango` (o repositório local existe, sem commit; push só com pedido explícito).
- [ ] Validar `docker build` e `docker run` do `Dockerfile` do site e do `deploy/Dockerfile.node` (bloqueado: daemon Docker indisponível neste ambiente; o usuário precisa liberar o acesso ao daemon).
- [ ] Validar o workflow `.github/workflows/ci.yml` após o primeiro push.
- [ ] Revisão visual completa do site em claro e escuro com o painel do navegador visível (pendente da sessão anterior).
- [ ] Remover qualquer resquício de código de referência: revisar PRs contra cópia do Tucano e do agentes.chat (ADR 0006).

---

## Pós-V1 (fora deste TODO)

- Relay próprio (F7-T01).
- Canais externos: Telegram, WhatsApp, Slack, e-mail (F7-T02).
- Voz (F7-T03), arquivos e documentos (F7-T04), marketplace (F7-T05).
- GitLab na ADE.

---

## Como usar este TODO

1. Confirme as decisões da seção 0 com o dono antes de começar o bloco que depende delas.
2. Siga a ordem das fases. Não pule o critério de saída.
3. Marque `[x]` aqui e em `SPECS/ROADMAP.md` ao terminar cada tarefa, com evidência em `VERIFY.md`.
4. Registre o que foi feito em `WORKLOG.md` e o próximo passo em `HANDOFF.md`.
