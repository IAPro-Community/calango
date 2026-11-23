# ADR 0008 — Replicação entre nós por eventos; código por git

Status: aceita · 2026-11-23

## Contexto
PC e VPS trabalham juntos (execução híbrida). Conversas, memória, rotinas e tarefas precisam aparecer nos dois. Código de projetos também.

## Decisão
- Cada nó escreve um log append-only (`events`) com `(node_id, seq)` único.
- Nós trocam eventos pelo transporte; cada um aplica os do outro e guarda `last_applied_seq` em `peers`.
- Cada entidade tem um **nó dono**; só o dono altera. O outro nó pede alteração por comando, que o dono aplica e publica. Não há edição concorrente, então não há merge.
- Código sincroniza só por git (push/pull entre worktrees e remoto). Nada de sync de arquivos próprio.
- Arquivos grandes ficam no nó de origem; o outro busca sob demanda.

## Consequências
- Simples de raciocinar; sem CRDT.
- Se o dono estiver offline, a alteração espera (a interface mostra isso).
- Rotina com `run_on=auto` muda de dono quando o PC fica offline: a transferência de dono é um evento explícito.
