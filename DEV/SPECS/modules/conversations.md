# Módulo: conversations

Fase 1 · Pacote `internal/conversations` · Tabelas `conversations`, `conversation_members`, `messages`, `runs`, `run_steps`

## Objetivo
DMs com agentes e, depois, canais. Cada mensagem pode disparar um run; o run publica passos em tempo real.

## Regras
- Um run por mensagem por agente. Vários runs em paralelo na mesma conversa até o limite configurável (padrão 3); acima disso, fila.
- Opção por agente: "uma resposta por vez".
- **Parar** encerra só aquele run. **Orientar** injeta uma instrução no próximo passo do run em curso, sem virar mensagem.
- **Tentar com instruções**: refaz um run parado com uma direção extra, no mesmo lugar.
- Passos (`run_steps`) são gravados e transmitidos: pensamento (opcional), chamada de ferramenta, resultado, texto.
- Próximos passos sugeridos pelo agente viram cartão de caixas de marcar, nada marcado por padrão; não bloqueia.
- Pedidos entre agentes aparecem dobrados, fora da contagem de não lidas.

## API
`GET/POST /v1/conversations`, `GET /v1/conversations/{id}/messages`, `POST /v1/conversations/{id}/messages`, `POST /v1/runs/{id}/stop`, `POST /v1/runs/{id}/steer`, `POST /v1/runs/{id}/retry`.

## Eventos
`message.created`, `run.started`, `run.step`, `run.finished`.

## Aceite
- Duas mensagens seguidas geram dois runs paralelos; parar um não afeta o outro.
- Run interrompido por queda do `calango` aparece como `failed` com `error_code=interrupted` na volta.
