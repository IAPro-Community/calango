# Módulo: agents

Fase 1 · Pacote `internal/core` (domínio) + `internal/agents` (serviço) · Tabelas `agents`, `agent_tools`

## Objetivo
Criar e configurar agentes: papel, instruções, harness/modelo, ferramentas, orçamento, onde rodam, nível de aprovação.

## Regras
- `token_limit_month > 0` obrigatório. Sem limite, sem chamada (já implementado em `core.Agent.Charge`).
- `run_on`: `pc`, `vps` ou `auto`. `vps` exige plano com nó VPS.
- `approval_level`: `always` (tudo que escreve pede), `sensitive` (padrão), `trusted` (só o que a lista fixa de ações perigosas define).
- Pausa global: um interruptor que impede qualquer run novo e para os em curso.
- Esforço de raciocínio tem os mesmos quatro nomes em todo lugar: `low`, `medium`, `high`, `max`.
- Criação assistida: a pessoa descreve, um agente interno gera um rascunho que ela revisa antes de salvar.

## API
`GET/POST /v1/agents`, `GET/PATCH/DELETE /v1/agents/{id}`, `POST /v1/agents/pause-all`, `POST /v1/agents/resume-all`.

## Eventos
`agent.updated`, `agents.paused`.

## Aceite
- Criar agente sem limite falha com `invalid_input`.
- Agente pausado não inicia run; a mensagem enviada a ele fica registrada com aviso.
- Mudança de instruções vale no próximo run, não no que está em curso.
