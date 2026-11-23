# Módulo: nodes-sync

Fase 3 · Pacotes `internal/sync`, `internal/transport` · Tabelas `nodes`, `events`, `peers` · ADR 0008

## Objetivo
PC e VPS da mesma conta trabalham como um só assistente.

## Regras
- Cada escrita de domínio gera um evento em `events` com `(node_id, seq)` crescente.
- Troca: ao conectar, cada nó pede ao outro eventos com `seq > last_applied_seq`; depois, stream contínuo.
- Cada entidade tem nó dono. O outro nó pede alteração por comando (`POST /v1/sync/commands`), que o dono aplica.
- `run_on=auto`: quando o PC fica offline por mais de 2 min, a VPS assume as rotinas auto (evento `ownership.transferred`). Quando o PC volta, as rotinas voltam para ele no fim do run em curso.
- Heartbeat a cada 15 s; offline após 45 s sem resposta. Evento `node.offline` com `since`.
- Código: worktrees sincronizam por git; o Calango só orquestra `push`/`pull`.

## Aceite
- Mensagem criada no PC aparece na VPS em menos de 2 s com ambos online.
- PC desligado: rotina auto roda na VPS no próximo horário; ao religar, o PC vê o resultado.
