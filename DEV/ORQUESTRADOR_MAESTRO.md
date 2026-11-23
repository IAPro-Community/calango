# Orquestrador Maestro dentro do Calango

Decisão: [ADR 0009](ADR/0009-orquestrador-maestro.md). Especificação do módulo: `SPECS/modules/maestro.md`.

## O que é

Projeto da IAPro Community, Apache-2.0: <https://github.com/IAPro-Community/Orquestrador-Maestro>.
Organiza agentes de código com regras, roteamento de skills, contexto compacto, verificação e handoff.
Versão estável da linha V1: **1.1.1** (release de 2026-10-18). Pacote npm: `@iapro/orquestrador-maestro-cli`. Requer Node.js 20.19 ou superior.

Comandos úteis da CLI oficial: `orquestrador-maestro install`, `verify`, `update`, `remove`, `sync`.

## Como o Calango usa

1. **Instala** a linha V1 numa pasta própria do Calango, nunca no diretório pessoal do usuário.
2. **Vincula** o Maestro ao workspace de cada agente que o dono escolher (`AGENTS.md`, `CLAUDE.md`, ou o equivalente do harness).
3. **Verifica** cada instalação ou atualização com `verify`. Falhou: reverte.
4. **Acompanha** o registro npm e avisa de novas versões da linha V1.
5. **Atualiza** conforme a preferência do dono (cartão ou automático). Versão maior: só com aviso.

## Fluxo de atualização

```
registro npm (1x/dia)
   └─ versão nova na linha V1?
        ├─ não → nada
        └─ sim → baixa para pasta nova (<calango-home>/maestro/versions/<v>)
              ├─ verify OK → troca o ponteiro "current" → guarda a anterior
              │      ├─ automático ligado → pronto, registra no WORKLOG do Calango
              │      └─ desligado → cartão "Maestro 1.x.y disponível"
              └─ verify falhou → mantém o ponteiro atual, avisa
```

Rollback: `current` volta para a versão anterior guardada. Versões antigas são limpas depois de 3 instalações bem-sucedidas.

## Regras

- Dono decide se o Maestro está ligado, por agente.
- Atualização automática **desligada** por padrão.
- Nunca atualizar durante um run em curso; espera o run terminar.
- Mudança nas regras do Maestro não altera o histórico de conversas nem a memória do agente.
- O Calango não edita arquivos do Maestro à mão. Só instala, verifica e troca versão.
- Toda atualização gera entrada na auditoria (`audit_log`) com versão anterior e nova.

## Atribuição

- `NOTICE` na raiz do repo e na tela "Sobre" do app.
- Link para o repositório e para a licença Apache-2.0.
- Não alterar o nome nem a marca do projeto.

## Pendências técnicas (verificar antes de implementar)

- Confirmar se `orquestrador-maestro install` aceita diretório de destino fora do home padrão. Se não aceitar, usar `HOME` isolado por processo.
- Confirmar saída máquina-legível (JSON) de `verify` e `update`.
- Confirmar que `update` não exige interação.
- Testar em Linux, Windows e macOS: o README oficial cita os três.
