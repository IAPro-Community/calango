# Módulo: routines

Fase 1 (cron/intervalo) · Fase 3 (eventos, híbrido) · Pacote `internal/core/scheduler.go` + `internal/routines` · Tabelas `routines`, `routine_runs`

## Objetivo
Trabalho agendado ou disparado por evento, executado por um agente.

## Regras
- Gatilhos: `cron` (expressão de 5 campos, fuso da conta), `interval`, `event` (conector: e-mail novo, PR aberto, webhook).
- `run_on`: pc, vps ou auto. Auto = PC se online, senão VPS; sem VPS, espera o PC.
- **Retry**: até `max_retries` com backoff exponencial (1 min, 5 min, 30 min). Esgotou: notifica o dono com o erro. Nunca perde a entrega em silêncio.
- **Silêncio**: o agente declara se teve novidade (`had_news`). `notify_on=news` não notifica sem novidade.
- **Janela de silêncio**: execuções dentro dela são adiadas para o fim da janela, não descartadas.
- Rotina perdida (PC desligado no horário): roda uma vez quando voltar, não N vezes.
- Uma rotina pode encerrar a si mesma (ferramenta `routine_finish`).
- Rotinas não aparecem no Quadro de tarefas.

## Estado atual
`core.Scheduler` com intervalo e relógio injetável já existe e tem testes (não executados).

## API
`GET/POST /v1/routines`, `PATCH/DELETE /v1/routines/{id}`, `POST /v1/routines/{id}/run`, `GET /v1/routines/{id}/runs`.

## Eventos
`routine.fired`, `routine.failed`.

## Aceite
- Falha transitória seguida de sucesso no retry: um `routine_run` com `attempt=2`, sem notificação de falha.
- PC desligado por 3 horários: ao voltar, uma execução só.
