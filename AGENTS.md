# AGENTS.md — Calango

Ponto de entrada para qualquer agente de código (Claude Code, Codex, Cursor, etc.) que vá trabalhar neste repositório.

## O que é

Calango é um agente pessoal que roda no computador da pessoa (nó PC, grátis), se estende para a VPS dela (nó VPS, pago) e é controlado de qualquer lugar pelo celular (iOS/Android) e pelo desktop. Núcleo em Go, desktop em Electron, mobile em Flutter.

## Ordem de leitura (obrigatória antes de codar)

1. `DEV/INDEX.md` — mapa de toda a documentação.
2. `DEV/HANDOFF.md` — onde o trabalho parou e o próximo passo.
3. `DEV/CONTEXT.md` — estado atual do projeto.
4. `DEV/SPECS/ACTIVE.md` — contrato da tarefa em andamento.
5. `DEV/SPECS/ROADMAP.md` — fases e tarefas numeradas (ex.: `F1-T03`).
6. Só então a spec do módulo que você vai tocar, em `DEV/SPECS/modules/`.

Não carregue o `DEV/` inteiro. Leia o que a tarefa pede.

## Regras inegociáveis

- **Código original.** Tucano (`eduwxyz/tucano`) e agentes.chat (`agentes-chat/agentes-chat`) foram lidos só como referência de produto. Não copie código, texto, testes ou assets deles. O agentes.chat é AGPL; qualquer trecho copiado contaminaria a licença.
- **Exceção: Orquestrador Maestro** (`IAPro-Community/Orquestrador-Maestro`, Apache-2.0). Não copiar o repositório: consumir pelo pacote npm oficial (ADR 0009). Manter `NOTICE` atualizado.
- **O binário se chama `calango`** (não `calangod`). Entrada em `cmd/calango`.
- **Clientes não executam agente.** Desktop, mobile e web só falam com a API do `calango`.
- **Toda chamada de modelo passa pelo gateway** (`internal/gateway`): orçamento checado antes, custo registrado depois.
- **Ação sensível exige aprovação** (enviar, apagar, publicar, pagar, deploy, comando destrutivo).
- **Segredos nunca vão para o cliente** nem para logs.
- **Migrações são imutáveis** depois de commitadas: mudança = nova migração.
- **Decisão de arquitetura nova = ADR nova** em `DEV/ADR/`.
- Não faça commit nem push sem pedido explícito do dono do projeto.
- Licença: AGPL-3.0 (`LICENSE`). Contribuições exigem aceitar o `CLA.md`. Não adicione código de terceiro sem checar a licença.

## Comandos

```bash
go vet ./...
go test ./...
go run ./cmd/calango          # sobe em 127.0.0.1:7317
curl 127.0.0.1:7317/healthz
```

Desktop, mobile e cloud ainda não existem; os comandos deles entram aqui quando forem criados.

## Depois de trabalhar

Atualize `DEV/WORKLOG.md` (o que mudou e por quê), `DEV/VERIFY.md` (o que foi verificado e como) e `DEV/HANDOFF.md` (próximo passo). Marque a tarefa no `DEV/SPECS/ROADMAP.md`.
