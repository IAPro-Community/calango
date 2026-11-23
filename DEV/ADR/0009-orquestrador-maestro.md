# ADR 0009 — Orquestrador Maestro embutido, com atualização automática pelo canal oficial

Status: aceita · 2026-11-23 · **substitui parcialmente o ADR 0006** (só para o Maestro)

## Contexto
O Orquestrador Maestro (`IAPro-Community/Orquestrador-Maestro`, Apache-2.0) organiza o trabalho de agentes de código: regras, roteamento de skills, contexto compacto, verificação e handoff. O dono quer que o Calango embuta esse processo e acompanhe as atualizações dele. O ADR 0006 proíbe copiar código de referência; o Maestro é a exceção escolhida explicitamente pelo dono, com licença permissiva.

## Decisão
1. **Não copiar o repositório.** O Calango consome o Maestro pelo pacote npm oficial `@iapro/orquestrador-maestro-cli`, fixado na linha **V1** (`^1.1.1`). Assim o código pertence ao canal oficial e as atualizações chegam por versão.
2. **Atualização automática, com controle.** O `calango` consulta o registro npm periodicamente (padrão: 1 vez por dia). Quando sai versão nova dentro da linha V1:
   - aplica sozinho só se o dono ligou "atualizar automaticamente" (padrão: **desligado**, porque a atualização muda as regras de todos os agentes);
   - caso contrário, mostra um cartão: versão, resumo do CHANGELOG e botão **Atualizar** / **Agora não**.
   - Versão maior (V2 ou seguinte) **nunca** entra sozinha: sempre cartão com aviso de quebra.
3. **Instalação isolada.** O Maestro é instalado numa pasta do Calango (`<calango-home>/maestro/`), não na raiz do usuário. Só é ligado aos agentes que o dono escolher, por harness (Claude Code, Codex etc.), por link no `AGENTS.md`/`CLAUDE.md` do workspace.
4. **Verificação antes de ativar.** Após instalar ou atualizar, o Calango roda `orquestrador-maestro verify`. Se falhar, reverte para a versão anterior e avisa.
5. **Rollback.** Cada versão instalada fica guardada; reverter é trocar o ponteiro de versão.
6. **Atribuição.** Apache-2.0 exige manter a licença e os avisos. O Calango inclui `NOTICE` na raiz e uma tela "Sobre" com o crédito ao projeto e o link do repositório.

## Alternativas descartadas
- **Copiar o código para dentro do repo**: quebra a atualização automática e vira fork a manter.
- **Submódulo git**: acoplado a commits; atualização manual.
- **Instalar na pasta do usuário (`~/.orquestrador-maestro`, `~/.codex`)**: mexe em configuração pessoal fora do controle do Calango.

## Consequências
- Dependência de Node.js 20.19+ na máquina do dono (ou dentro do container, na VPS). Entra em `F1-T03`/`F3-T01`.
- A versão do Maestro vira parte do contrato: quando mudar, o Calango mostra a versão no painel de agentes.
- Quem não quer o Maestro roda sem ele; nada do núcleo depende dele.
