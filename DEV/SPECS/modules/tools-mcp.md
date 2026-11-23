# Módulo: tools-mcp

Fase 1 · Pacotes `internal/tools`, `internal/mcpgw` · Tabelas `connectors`, `agent_tools`

## Objetivo
Ferramentas dos agentes via MCP, com escopo por agente e aprovação no que escreve.

## Regras
- Conectar um servidor MCP uma vez (stdio ou HTTP); cada agente recebe um subconjunto de ferramentas com acesso `none`, `read` ou `write`.
- **Gateway MCP local**: a cada turno, uma URL secreta que lista só as ferramentas daquele agente. É a única fonte de ferramentas do harness.
- Chamada a ferramenta `write` passa por `approvals`.
- Instruções do servidor MCP (`instructions`) são repassadas ao agente quando a conexão permitir.
- Ferramentas próprias do Calango também saem pelo gateway: `memory_save`, `next_steps`, `ask_person`, `routine_finish`, `task_update`.
- O nó também expõe um **servidor MCP** para Claude Code/Codex externos falarem com o Calango (fase 4).
- Recusa por regra devolve **o que fazer em vez disso**, para o agente não repetir a mesma chamada.

## Aceite
- Agente sem acesso a uma ferramenta não a vê na lista.
- Ferramenta `write` sem aprovação não executa.
