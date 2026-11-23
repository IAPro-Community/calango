# Módulo: memory

Fase 1 · Pacote `internal/memory` · Tabela `memories`, `skills`

## Objetivo
O agente lembra fatos, preferências e decisões, e a pessoa controla tudo.

## Regras
- Escopos: conta, agente, projeto.
- O agente salva memória por ferramenta (`memory_save`); a pessoa vê, edita e apaga.
- Injeção no contexto: memórias do escopo, mais recentes e mais relevantes primeiro, até um orçamento de tokens fixo.
- Apagar é lógico (`deleted_at`) e some do contexto na hora; limpeza física depois de 30 dias.
- O agente pode **propor** mudança nas próprias instruções; vira cartão que a pessoa aceita ou recusa.
- Skills: o agente propõe virar skill quando repete um fluxo; a pessoa aprova antes de valer.

## API
`GET/POST /v1/memories`, `PATCH/DELETE /v1/memories/{id}`, `GET/POST /v1/skills`, `PATCH /v1/skills/{id}`.

## Aceite
- Memória apagada não aparece no próximo run.
- Proposta de instrução recusada não altera o agente.
