# Módulo: channels-teams

Fase 6 · Pacotes `internal/conversations`, `internal/router`, `internal/members`

## Objetivo
Canais com pessoas e agentes; plano Team.

## Regras
- Canais com threads e menções. Mensagem sem menção vai para o **roteador**: modelo barato escolhe 0 a 2 agentes. O roteador nunca concede permissão.
- A escolha do roteador aparece no topo da resposta que ela iniciou, não como linha solta.
- "Ninguém precisa responder" é uma decisão válida e aparece discreta.
- Membros com papéis (dono, admin, membro, convidado); permissões por canal.
- Aprovação delegada: o dono de um agente pode nomear substituto.
- Um agente pode servir mais de uma pessoa (especialista compartilhado).

## Aceite
- Mensagem sem menção num canal com três agentes gera no máximo dois runs.
- Convidado não vê canais em que não foi incluído.
