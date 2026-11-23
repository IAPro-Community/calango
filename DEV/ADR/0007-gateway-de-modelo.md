# ADR 0007 — Gateway único de modelo com orçamento

Status: aceita · 2026-11-23

## Contexto
Freemium e limites de gasto dependem de medir cada chamada de modelo, venha de API direta ou de CLI de harness.

## Decisão
Toda chamada de modelo passa por `internal/gateway`:
1. Checa orçamento do agente (e da rotina, se houver) antes da chamada. Estourou → erro `budget_exceeded`, nenhuma chamada sai.
2. Executa a chamada pelo provider ou harness.
3. Registra tokens e custo em `usage` com agente, rotina, projeto e nó.
4. Emite `usage.threshold` ao passar de 80% e 100%.

Harness CLI que não informa tokens: o gateway estima e marca `estimated=true`.

## Consequências
- Nenhum outro pacote importa SDK de provider diretamente.
- O mesmo registro alimenta painel de uso, limites do plano e cobrança de créditos.
