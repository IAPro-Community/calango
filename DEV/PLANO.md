# Calango — plano completo do software

Status: planejamento. Versão 1, 2026-11-23. Nada além da fundação (fase 0) está implementado.
Documentos de apoio: [ARCHITECTURE.md](ARCHITECTURE.md), [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md), [MONETIZATION.md](MONETIZATION.md), [DEPLOYMENT.md](DEPLOYMENT.md).

---

## 1. Visão

**Calango é um agente pessoal que mora no seu computador, se estende para a sua VPS e responde no seu bolso.**

Ele conversa, lembra, executa tarefas, roda rotinas, navega na web, programa projetos e trabalha em canais com pessoas e outros agentes. Tudo roda em máquinas da própria pessoa. A Calango (empresa) fornece só identidade, licença, push e atualizações.

Posicionamento frente às referências:

| | Roda onde | Ponto forte | O que falta |
|---|---|---|---|
| Dot (encerrado) | nuvem deles | memória, proatividade | não executava nada |
| Grok Bot | nuvem deles | sempre ligado | sem controle local, dados com terceiros |
| Tucano | só o PC | agentes, canais, aprovações | sem nuvem, sem celular de verdade |
| agentes.chat | servidor de time | chat de equipe com agentes | não é assistente pessoal, AGPL |
| **Calango** | **PC + VPS da pessoa** | **pessoal + 24/7 + ADE + celular** | — |

## 2. Princípios

1. **Os nós são da pessoa.** Dados, chaves e execução ficam no PC e na VPS dela.
2. **Um núcleo só.** O mesmo binário Go (`calango`) roda no PC e na VPS. Clientes não executam agente.
3. **Nada sensível sem aprovação.** Enviar, apagar, publicar, pagar, fazer deploy: sempre um cartão de aprovação.
4. **Tudo tem custo visível.** Cada chamada de modelo é medida, tem dono e tem limite.
5. **Silêncio é uma resposta válida.** Rotina sem novidade não notifica ninguém.
6. **Código original.** Tucano e agentes.chat são só referência de leitura.

---

## 3. Topologia

```
                        ┌──────────────── Calango Cloud (nosso) ───────────────┐
                        │ conta · licenças · billing · push relay · updates     │
                        └───────────────▲──────────────────▲────────────────────┘
                                        │ só metadados     │
          Tailscale (tailnet da pessoa) │                  │
   ┌────────────────────────────────────┼──────────────────┼───────────────┐
   │                                    │                  │               │
┌──┴────────────────┐          ┌────────┴─────────┐   ┌────┴─────┐   ┌─────┴────┐
│ Nó PC (grátis)    │◄────────►│ Nó VPS (pago)    │   │ iOS      │   │ Android  │
│ Electron + calango│  eventos │ Docker calango  │   └──────────┘   └──────────┘
│ arquivos, apps,    │  + git   │ 24/7, rotinas,   │        ▲ controle remoto ▲
│ navegador local    │          │ projetos, browser│        └──── de PC e VPS ┘
└────────────────────┘          └──────────────────┘
```

- **Nó PC**: app Electron que sobe o `calango` como processo filho.
- **Nó VPS**: `calango` em Docker, sem interface própria; acessado pelos clientes e pela web.
- **Clientes**: desktop (Electron), iOS, Android, web. Todos falam a mesma API.
- **Transporte**: Tailscale na fase inicial (decidido). Relay próprio depois.
- **Calango Cloud**: nunca vê conversas, arquivos ou chaves de modelo.

---

## 4. Stack por componente

### 4.1 `calango` — núcleo (Go)

| Função | Escolha |
|---|---|
| Linguagem | Go 1.22+ |
| API | HTTP + WebSocket, contrato OpenAPI (gera o SDK TypeScript) |
| Banco | SQLite puro Go (`modernc.org/sqlite`), migrações versionadas e imutáveis |
| Agendador | interno (já existe o esqueleto), com cron, intervalos e eventos |
| Gateway de modelo | todo token passa por ele: orçamento antes, custo depois |
| Gateway de ferramentas | MCP; cada agente vê só as ferramentas e escopos dele |
| Harnesses | adaptadores para CLIs de agente (Claude Code, Codex, Gemini, Grok, OpenCode) e API direta |
| Navegador do agente | Chromium via DevTools Protocol (`chromedp`) |
| Terminal | PTY no daemon (`creack/pty`; ConPTY no Windows), transmitido por WebSocket |
| Git / ADE | git do sistema; um worktree por tarefa |
| Isolamento | Docker na VPS; no PC, pasta de trabalho restrita + aprovação por comando |
| Rede | Tailscale (avaliar `tsnet` embutido) |
| Segredos | cofre do sistema: Windows Credential Manager/DPAPI, macOS Keychain, Linux Secret Service |

### 4.2 Desktop — Electron (Windows, macOS, Linux)

- **Processo principal (Node)**: janela, bandeja, atalhos globais, auto-update, inicia e vigia o `calango`.
- **Renderer (React + TypeScript + Vite)**: toda a interface. Fala com o `calango` pela API, igual ao celular.
- **Navegador interno**: abas reais com `WebContentsView`. O agente dirige a mesma aba pelo protocolo de depuração do Electron, e a pessoa pode **assumir** a aba a qualquer momento (e devolver).
- **Editor da ADE**: Monaco. **Terminal**: xterm.js ligado ao PTY do daemon.
- **Windows**: instalador NSIS assinado (certificado de code signing desde o primeiro release público; o Tucano sofre com aviso do SmartScreen por não assinar), inicia com o sistema, ícone na bandeja.
- **macOS**: assinado e notarizado. **Linux**: `.deb` e AppImage (AppImage é restrito no Ubuntu 24.04+, então `.deb` é o caminho principal lá).

### 4.3 Mobile — iOS e Android

- **Flutter** (decidido em 2026-11-23). Um código Dart para iOS e Android.
- Como o desktop é React/TypeScript, o compartilhamento passa a ser por **contrato**, não por código:
  - **API**: o OpenAPI do `calango` gera dois clientes, TypeScript (desktop/web) e Dart (mobile). Nenhum cliente é escrito à mão.
  - **Design**: tokens (cores, tipografia, espaçamento) num JSON único, que gera CSS para o desktop e `ThemeData` para o Flutter.
  - **Traduções**: catálogo único que gera os arquivos ARB do Flutter e o JSON do desktop.
- Custo dessa escolha: telas duplicadas (React e Flutter). Mitigação: o mobile tem um escopo menor e focado (controle remoto, aprovações, conversas), não replica a ADE.
- Funções: conversas, aprovações, rotinas, tarefas, notificações, status dos nós, controle remoto do PC e da VPS.
- Push via relay nosso (APNs/FCM), só com "tem novidade"; o app busca o conteúdo no nó.
- Compra: sistema das lojas (via RevenueCat ou integração direta), com assinatura reconhecida também no desktop.

### 4.4 Web

O mesmo app React, servido pelo próprio `calango`. Serve para acessar a VPS de um navegador qualquer.

### 4.5 Calango Cloud (Go, nosso)

| Serviço | Função |
|---|---|
| Conta | login, dispositivos, lista de nós da pessoa |
| Licenças | emite chave assinada (Ed25519) que o nó VPS valida offline |
| Billing | Stripe (web/desktop) + compras das lojas; uma fonte de verdade de plano |
| Push relay | encaminha avisos para APNs/FCM, sem conteúdo |
| Updates | canais estável/beta para desktop e imagem Docker |
| Telemetria | só com opt-in, sem conteúdo |

---

## 5. Módulos funcionais

### 5.1 Assistente pessoal
- Chat por texto e voz (ditado e resposta falada numa fase posterior).
- **Briefing diário** configurável: agenda, e-mails importantes, tarefas, o que as rotinas acharam.
- **Proatividade**: o agente sugere próximos passos num cartão de caixas de marcar, sem bloquear.
- "Para você": caixa única com perguntas, aprovações e entregas pendentes.

### 5.2 Agentes
- Vários agentes por conta, cada um com papel, instruções, modelo/harness, ferramentas, orçamento mensal e onde roda (PC, VPS, automático).
- Assistente de criação: descreva o agente, ele monta o rascunho.
- Respostas em paralelo com limite configurável; orientar uma resposta em curso; parar só uma.
- Pausa global de todos os agentes.

### 5.3 Canais e DMs
- DM com cada agente; canais com vários agentes (e pessoas, no plano Team).
- **Roteador**: um modelo barato escolhe quem responde (0 a 2 agentes). Nunca concede permissão.
- Threads, menções, citação de trecho.
- A escolha do roteador aparece no topo da resposta, não como linha solta no canal.

### 5.4 Rotinas
- Gatilhos: horário (cron), intervalo, evento de conector (e-mail novo, PR aberto, webhook).
- Destino: PC, VPS ou automático.
- **Nova tentativa e aviso** quando falha (o agentes.chat perde entregas semanais por não fazer isso).
- Janela de silêncio; rotina sem novidade não notifica.
- Calendário de rotinas e histórico de execuções com custo.

### 5.5 Tarefas (Quadro)
- Kanban de tarefas: minhas, sugeridas por agentes, em andamento por agentes.
- Uma tarefa pode atravessar várias execuções até terminar.
- Rotinas ficam fora do Quadro (são agenda, não tarefa).

### 5.6 Aprovações
- Cartão com ação, alvo e argumentos exatos.
- Recibo depois: concluído, falhou (com motivo) ou não executado.
- Níveis por agente: sempre perguntar, perguntar só no sensível, aprovar sempre esse tipo.
- Aprovação pelo celular e, no Team, por substituto.
- Aprovações respeitam a ordem em que foram pedidas.

### 5.7 Navegador interno
- Abas, perfis persistentes por agente, bloqueador de anúncios e de avisos de cookie.
- O agente identifica **barreiras** (verificação de robô, login, página vazia) e avisa em vez de insistir.
- Pessoa assume e devolve a aba sem recarregar.
- Conteúdo de página é sempre tratado como dado, nunca como instrução (defesa contra prompt injection).

### 5.8 ADE — ambiente de desenvolvimento com agentes
- **Projetos** separados, cada um com seu repositório, agentes, conectores e memória (`DEV/` como memória do projeto).
- Cada tarefa abre um **worktree** próprio: vários agentes trabalham em paralelo sem pisar um no outro.
- Editor, terminal, diff, rodar testes, servidor de preview aberto no navegador interno.
- Integração GitHub/GitLab: branch, PR, revisão, CI.
- Projeto pode rodar no PC ou na VPS; sincronia sempre por git.
- Importar skills de Claude Code e Codex.

### 5.9 Uso do computador (PC)
- Agente opera apps de desktop (cliques, teclado, capturas de tela). Desligado por padrão, liga por app.
- Controle da tela do PC pelo celular (plano pago), com aviso visível no PC e auditoria.

### 5.10 Conectores
- Catálogo de servidores MCP (Gmail, Calendar, Drive, GitHub, Slack, Notion, Jira etc.) e MCP próprio.
- Conecta uma vez por conta; cada agente recebe só os escopos de que precisa (nenhum, leitura, leitura e escrita).
- O nó também expõe um servidor MCP, para Claude Code ou Codex conversarem com o Calango.

### 5.11 Skills
- Receitas em Markdown reutilizáveis.
- Quando o agente repete um fluxo, ele propõe virar skill; a pessoa aprova.

### 5.12 Memória
- Fatos, preferências e decisões, por conta, por agente e por projeto.
- Tudo revisável, editável e apagável. O agente propõe mudanças nas próprias instruções; a pessoa aceita ou recusa.

### 5.13 Arquivos e documentos (fase posterior)
- Pasta de entregas dos agentes, com quem vê cada pasta.
- Prévia de PDF, imagem, vídeo, planilha e texto, com botão de copiar.

### 5.14 Cofre
- Segredos criptografados pelo sistema operacional; o agente usa sem ver o valor.
- Importação de senhas exportadas do navegador numa fase posterior.

### 5.15 Uso e custos
- Painel por agente, rotina, projeto e nó: tokens, custo, tempo.
- Limites por agente e alerta antes de estourar.

### 5.16 Canais externos (fase posterior)
- Telegram para aprovações e conversa.
- WhatsApp, Slack e e-mail como entrada e saída dos agentes, com limites de envio e aprovação.

### 5.17 Times (plano Team)
- Membros, papéis, permissões por canal, agentes compartilhados, aprovação delegada.

---

## 6. Segurança e privacidade

| Ameaça | Defesa |
|---|---|
| Acesso indevido ao nó | pareamento por código único, token por dispositivo, revogação, Tailscale |
| Agente faz algo destrutivo | aprovação obrigatória no sensível, isolamento, pasta de trabalho restrita |
| Prompt injection por web, e-mail ou arquivo | conteúdo externo marcado como dado; ferramentas de escrita sempre aprovadas |
| Vazamento de chaves | cofre do sistema, segredos nunca vão para o cliente, máscara em logs |
| Gasto descontrolado | orçamento por agente checado antes de cada chamada |
| Controle remoto abusado | opt-in no PC, aviso visível, auditoria, encerra com um clique |
| Saída de dados | lista de destinos permitidos por agente (fase posterior) |

Auditoria: toda ação remota e toda aprovação ficam registradas e visíveis no PC e no celular.

---

## 7. Dados e sincronização

- Cada nó tem seu SQLite.
- **Registro de eventos** (append-only) por nó, replicado entre os nós da conta. Cada registro tem um nó dono; não há edição concorrente do mesmo registro.
- **Código** sincroniza só por git.
- **Arquivos grandes** ficam no nó onde foram criados; o outro nó busca sob demanda.
- **Backup**: comando documentado para o volume do Docker e para a pasta de dados do PC.
- Quando um nó cai, os clientes mostram qual, desde quando, e o que está esperando por ele (lição do agentes.chat, onde só aparece "esperando computador").

---

## 8. Monetização (resumo)

| | Free | Pro | Team |
|---|---|---|---|
| Nó PC | 1 | 1 | por membro |
| Nó VPS | — | 1 | por membro |
| Celular (via Tailscale) | sim | sim | sim |
| Execução híbrida e rotinas 24/7 | — | sim | sim |
| Push no celular | limitado | sim | sim |
| Controle de tela remoto | — | a decidir | sim |
| Canais com pessoas | — | — | sim |

Extras pagos: créditos de modelo (o grátis usa chave ou assinatura própria), VPS gerenciada por nós, armazenamento, conectores premium. Licença do nó VPS assinada e validada offline; sem licença, rebaixa para o modo grátis em vez de parar. Detalhes em [MONETIZATION.md](MONETIZATION.md) e [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md).

---

## 9. Distribuição

| Plataforma | Formato | Cuidado |
|---|---|---|
| Windows | NSIS assinado, auto-update | certificado de code signing, Git for Windows para a ADE |
| macOS | DMG assinado e notarizado | permissões de acessibilidade e gravação de tela para uso do computador |
| Linux | `.deb`, AppImage | keyring destravado para o cofre |
| VPS | imagem Docker + instalador de uma linha | volume de dados, backup, Tailscale |
| iOS | App Store (TestFlight antes) | compra pelo sistema da Apple |
| Android | Google Play | compra pelo sistema do Google |

Idiomas desde o início: português, inglês e espanhol, com catálogo de traduções versionado.

---

## 10. Monorepo

```
calango/
├── cmd/
│   ├── calango/           núcleo (PC e VPS)
│   └── calango-cloud/     serviços nossos
├── internal/              Go: core, api, store, gateway, tools, browser, ade, sync
├── api/                   openapi.yaml (fonte dos dois SDKs)
├── apps/
│   ├── desktop/           Electron (main + renderer React)
│   ├── mobile/            Flutter (iOS, Android)
│   └── web/               build web do renderer
├── packages/
│   ├── sdk-ts/            cliente TypeScript gerado do OpenAPI
│   ├── sdk-dart/          cliente Dart gerado do OpenAPI
│   ├── tokens/            design tokens (JSON → CSS e ThemeData)
│   ├── ui/                componentes React do desktop/web
│   └── i18n/              catálogo único (→ JSON e ARB)
├── deploy/                Dockerfile, compose, instalador
└── DEV/                   documentação viva
```

---

## 11. Roadmap

| Fase | Entrega | Critério de pronto |
|---|---|---|
| **0. Fundação** | esqueleto Go, docs, Docker (feito, sem compilar) | `go test` e `/healthz` passando |
| **1. MVP desktop local** | Electron + `calango`; 1 agente via harness; chat; memória; rotinas; aprovações; painel de uso | instalar no Windows, conversar, criar rotina, aprovar ação |
| **2. Celular** | app Flutter; pareamento; Tailscale; push via relay | aprovar uma ação do PC pelo celular fora de casa |
| **3. VPS e primeira receita** | imagem Docker; licença; execução híbrida; billing | rotina migra para a VPS com o PC desligado; assinatura Pro cobrada |
| **4. ADE** | projetos, worktrees, editor, terminal, diff, PR | agente abre PR a partir de uma tarefa, no PC ou na VPS |
| **5. Navegador e uso do computador** | navegador interno com assumir; uso de apps; controle de tela | agente preenche um formulário e a pessoa assume no meio |
| **6. Canais e times** | canais com pessoas, roteador, plano Team | dois membros e três agentes num canal |
| **7. Expansão** | relay próprio, canais externos, voz, arquivos, marketplace | — |

---

## 12. Lições das referências (lidas em issues, PRs e changelog)

- **Assinar os instaladores desde o primeiro release.** Tucano e agentes.chat recebem reclamações por Windows sem assinatura e Mac sem notarização.
- **Login de CLI expira em segundo plano.** O agentes.chat tem vários bugs de conta Grok/Antigravity caindo; o adaptador de harness precisa detectar e pedir novo login de forma clara.
- **Rotina que falha precisa tentar de novo e avisar.** E rotina sem novidade não deve notificar.
- **Navegador de agente esbarra em verificação de robô.** Detectar a barreira e passar para a pessoa, em vez de clicar de novo.
- **Nó offline precisa dizer qual e desde quando.**
- **Ruído no canal**: a escolha do roteador e pedidos entre agentes devem aparecer dobrados, não como mensagens soltas.
- **Esforço de raciocínio** deve ter os mesmos nomes em todo lugar onde se escolhe modelo.

---

## 13. Métricas

Ativação (primeira rotina criada em 24h), retenção semanal, conversão grátis→Pro, nós VPS ativos, custo de modelo por usuário ativo, aprovações respondidas pelo celular, rotinas executadas com sucesso.

## 14. Decisões em aberto

1. ~~Mobile~~: decidido, Flutter.
2. **Licença do código do Calango**: aberto, fonte disponível ou fechado. Afeta a licença do nó VPS.
3. **Controle de tela**: Pro ou só Team.
4. **Primeiro harness do MVP**: Claude Code, Codex ou API direta.
5. **Nome do módulo Go e domínio.**

## 15. Riscos

| Risco | Mitigação |
|---|---|
| Escopo grande demais | fases com critério de pronto; não começar a fase seguinte antes |
| Custo de modelo no grátis | grátis usa chave ou assinatura própria |
| Dependência de CLIs de terceiros | camada de adaptador; testes de contrato por harness |
| Dependência do Tailscale | relay próprio no roteiro (fase 7) |
| Lojas recusarem controle remoto | revisar diretrizes antes da fase 2 |
| Segurança do uso do computador | desligado por padrão, aprovação e auditoria |
