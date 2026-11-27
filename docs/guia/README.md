# Guia do Calango

Guia em pt-BR para quem vai usar o Calango. Cada página separa o que **já funciona** do que **vem em fases futuras**.

> [!IMPORTANT]
> O Calango está na fase 1 (MVP desktop local), ainda sem versão publicada. Hoje ele roda a partir do código-fonte. O instalador e o app de celular ainda não existem para o público.

| Página | Assunto |
|---|---|
| [Primeiros passos](primeiros-passos.md) | instalar a partir do código, conectar um modelo, criar o primeiro agente |
| [Pareamento de dispositivos](pareamento.md) | códigos de pareamento, escopos e revogação |
| [Calango no VPS](vps.md) | imagem Docker, chave do cofre, volume de dados e TLS |
| [Orquestrador Maestro](maestro.md) | instalar, vincular a agentes e atualizar |
| [Privacidade](privacidade.md) | o que fica guardado, onde fica e o que sai da máquina |

## O que existe em cada fase

| Recurso | Situação |
|---|---|
| App desktop (agentes, conversas, aprovações, rotinas, consumo, configurações, auditoria) | disponível, a partir do código |
| Contas Codex (login do Codex CLI) e Anthropic (chave de API) | disponível |
| Conectores MCP com aprovação para ações de escrita | disponível por API, sem tela no app |
| Cofre de segredos criptografado | disponível |
| Pareamento de dispositivos por API | disponível, sem tela no app |
| Imagem Docker para VPS | exemplo, parcialmente validado |
| Instalador Windows | em validação interna (fase 1) |
| Pacotes Linux e macOS | fase futura (v1.0.0) |
| Pareamento por QR Code e app de celular | fase 2 (v0.3.0) |
| Desktop conectado a um nó remoto, execução PC/VPS | fase 3 (v0.4.0) |
| ADE, browser, computer use, canais com várias pessoas | fases 4 a 6 |

## Como este guia é mantido

- Os comandos foram executados no Linux em 25/11/2026, sobre o `develop` em `5a11c81`. Comandos de Windows e macOS que não foram executados estão marcados.
- A cada milestone, quem fechar a milestone revisa a tabela acima e as seções "Fases futuras" de cada página.
- Detalhes de engenharia ficam em [`DEV/`](../../DEV/INDEX.md). Este guia aponta para lá em vez de repetir.
