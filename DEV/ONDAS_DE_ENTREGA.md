# Ondas de entrega do Calango

A sequência operacional da v0.2.0 é **styleguide e núcleo → MVP Web → Desktop Windows**. Mobile vem depois de Windows. Uma onda pode ocupar várias sprints semanais. O PO registra o aceite na issue do gate com versão, ambiente, demonstração, evidências e resultado do CI.

| Onda | Entrada | Saída verificável |
| --- | --- | --- |
| 0 · Fundação | Refinar o [styleguide #175](https://github.com/IAPro-Community/calango/issues/175) e o [núcleo #31](https://github.com/IAPro-Community/calango/issues/31) em paralelo | Catálogo local acessível pelo índice com temas, estados e acessibilidade básica. Pela API/CLI, demonstrar conversa autorizada, rotina, aprovação, persistência, auditoria e custo. |
| 1 · Web | Aceite de #175 e #31 | [Gate #188](https://github.com/IAPro-Community/calango/issues/188): navegador conectado a um nó de referência, conversa, rotina, aprovação, resultado, custo, reconexão, erros e proteção das credenciais. |
| 2 · Windows | Aceite de #188 | [Gate #41](https://github.com/IAPro-Community/calango/issues/41): instalação assinada e fluxo principal em Windows. #116 valida pacote interno, #39 assinatura e distribuição, #118/#119 cobertura de UI e E2E. |
| 3 · Mobile | Aceite de #41 | [Gate #71](https://github.com/IAPro-Community/calango/issues/71): aprovação remota pelo celular. Preparar distribuição e TestFlight após validar o fluxo compartilhado. |
| 4 em diante · Expansão | Refinar após as ondas anteriores | Gates #81 (híbrido), #90 (ADE), #98 (browser) e #103 (equipes). Manter pós-V1 no backlog próprio. |

## Backlog e sprints

Use milestone para versão, campo de onda para ordem de liberação e o campo **Iteração** do GitHub Project para o compromisso semanal. Mantenha visões **Ondas e gates**, **Sprint atual** e **Bloqueadas**. A issue indica objetivo, evidência de aceite, responsável e apoio quando necessário. Use dependências nativas entre issues. P1 bloqueia o próximo aceite, P2 contribui para a onda, P3 está no horizonte futuro e P0 representa incidente crítico. Uma issue de onda futura só entra em **Pronta** após o gate anterior.

Na segunda-feira, planeje pelas horas disponíveis de cada pessoa, contando revisão, pareamento e homologação. Priorize recortes de uma ou duas janelas de trabalho e uma tarefa principal por pessoa. No check-in diário assíncrono, registre o próximo passo e a ajuda necessária. Combine apoio no mesmo dia útil. Se o bloqueio persistir no dia seguinte, PO e liderança técnica decidem entre parear, redistribuir, reduzir escopo ou replanejar. Na revisão semanal, demonstre o resultado e considere também a carga de revisão e apoio ao distribuir a semana seguinte.

## Primeira semana

- Styleguide: confirmar o catálogo com o roteiro em [STYLEGUIDE.md](STYLEGUIDE.md) e revisar o [PR #176](https://github.com/IAPro-Community/calango/pull/176).
- Núcleo: revisar #53 (provedor), #57 (MCP) e #117 (conformidade) como candidatas paralelas. A aprovação ponta a ponta #54 aguarda #57. A homologação de segurança #113 aguarda #53, #54 e #57. Triar as evidências já integradas dos bugs #156, #160 e #162 antes de classificá-los como bloqueadores.
- Nas semanas seguintes do núcleo, puxar rotinas, persistência, auditoria e custos conforme as dependências e o que ainda faltar para a demonstração de #31.
- Fechar o compromisso semanal com a capacidade declarada pela equipe. Registrar em #1 se um dev consegue localizar o catálogo, puxar uma tarefa pronta, encontrar quem destrava e explicar como demonstrar a entrega.

## Demonstração do núcleo

Em um nó de teste, registrar a versão, o ambiente e o orçamento aprovado. Pela API/CLI, enviar uma mensagem ao agente, acompanhar a resposta e conferir o consumo. Criar uma rotina, confirmar uma execução no horário configurado e consultar o resultado após reiniciar o nó. Em uma execução separada, solicitar uma ação externa em destino controlado, confirmar a espera antes da aprovação e conferir resultado e trilha de auditoria depois da decisão. Registrar as evidências e limitações em #31, com indicação dos testes que usaram fixtures. As sessões reais de #53 e #57 exigem autorização de execução e, quando houver consumo de modelo, orçamento aprovado.

## MVP Web · Planejar meu dia

A primeira experiência atende pessoas sem perfil técnico. A promessa é receber um plano com prioridades e próximos passos a partir das pendências informadas, na hora ou no horário escolhido. A equipe prepara o provedor e entrega um link individual para um ambiente isolado por participante no nó de referência. A experiência em português brasileiro inclui estados de execução, falha e custo.

Após o aceite de #31 e #175, detalhar três sub-issues no [épico #187](https://github.com/IAPro-Community/calango/issues/187), nesta ordem:

| Entrega | Demonstração |
| --- | --- |
| Acesso individual e interface Web | Abrir o link no navegador em uma sessão nova e chegar à tela inicial sem terminal ou configuração de API. Conferir isolamento entre participantes e proteção de credenciais. Reusar o SDK TypeScript, tokens e componentes compatíveis do renderer. |
| Criação e primeira execução | Digitar pendências, gerar um plano na hora e consultar prioridades, próximos passos, estado e custo. Conferir a apresentação de uma falha de execução. |
| Agendamento, histórico e recuperação | Escolher o horário, acompanhar uma execução agendada e reabrir o último plano após reconexão. Simular uma falha, consultar o diagnóstico e recuperar a rotina. |

O [gate #188](https://github.com/IAPro-Community/calango/issues/188) reúne as evidências de versão, ambiente, CI e aceite humano. A equipe demonstra aprovação externa em destino de teste controlado, em uma execução separada do planejamento diário. Calendário, e-mail, ADE, colaboração, billing e apps nativos seguem no backlog das ondas correspondentes.

### Piloto de sete dias

Convidar cinco pessoas do público escolhido. Registrar por participante, com identificador de piloto, o tempo até o primeiro plano, a necessidade de ajuda ao vivo e os dias em que consultou ou gerou planos. Entrevistar as cinco pessoas para documentar decisões apoiadas pelo plano e situações em que ele foi ignorado ou atrapalhou.

- **Ativação:** pelo menos quatro pessoas geram o primeiro plano sem ajuda ao vivo em até cinco minutos após receberem acesso.
- **Uso repetido:** pelo menos três pessoas voltam em três dias distintos para consultar ou gerar planos.
- **Confiabilidade:** demonstrar execução agendada, resultado persistido, diagnóstico, recuperação e custo visível. Interromper o piloto se uma ação externa ocorrer sem aprovação ou se uma credencial for exposta.

Se a ativação ficar abaixo da meta, corrigir o início da jornada na próxima sprint. Se a ativação atingir a meta e o uso repetido não, revisar a utilidade do plano e o horário da rotina. A próxima onda só entra em execução após o aceite do gate Web. Após o piloto, o PO decide se prioriza a próxima capacidade ou melhora a primeira experiência.
