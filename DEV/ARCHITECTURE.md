# Arquitetura — Calango

## Modos de implantação

- **VPS da pessoa (Docker)**: calango roda na VPS; desktop, iOS e Android se conectam por TLS. Ver [DEPLOYMENT.md](DEPLOYMENT.md).
- **Computador local**: mesmo binário, endereço local.
- **Hospedado por nós** (pago): mesma API, sem o usuário administrar servidor.

Os clientes nunca executam agente. Só mostram o estado e enviam comandos.

## Princípio

Um núcleo só, em Go. Local, nuvem e clientes são cascas em volta dele. Nada de lógica de agente no cliente.

## Camadas

```
clientes (desktop Electron, iOS, Android, web)
        │  HTTP + WebSocket
calango  (internal/api)
        │
núcleo    (internal/core: agentes, canais, rotinas, orçamento, aprovações)
        │
armazenamento (SQLite, interface Store)   ·   adaptadores (modelos, navegador, MCP)
```

## Pacotes

- `cmd/calango`: entrada do daemon. Sobe o servidor e o agendador; encerra com sinal.
- `internal/core`: domínio puro, sem I/O de rede.
  - `agent.go`: agente com orçamento de tokens (`Charge` recusa quando estoura).
  - `scheduler.go`: rotinas por intervalo, relógio injetável.
- `internal/api`: rotas HTTP.

## Regras

- Chamadas a modelos passam por um único gateway que cobra o orçamento antes e registra o custo depois.
- Segredos nunca voltam para o cliente.
- Ação de escrita sensível (enviar, apagar, publicar, pagar) exige aprovação explícita do dono.
- Toda entidade tem dono (`owner`) para suportar múltiplas contas na nuvem depois.
