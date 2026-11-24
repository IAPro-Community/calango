# Implantação da governança — issue #1

Este checklist não comprova que as configurações administrativas já foram aplicadas.

## Administração

- [ ] Confirmar mantenedores, revisores humanos independentes e permissões; adicionar CODEOWNERS somente com usuários/equipes reais confirmados.
- [ ] Settings > General: somente squash merge; título do PR como commit squash; exclusão automática de branches; auto-merge opcional condicionado aos gates; verificar Issues e Discussions.
- [ ] Settings > Rules > Rulesets: protected-main ativo; impedir exclusão e force push; exigir PR, uma aprovação independente, invalidar aprovações obsoletas, resolver conversas, histórico linear e squash; sem bypass permanente.
- [ ] Confirmar execução real de CI e nome do quality-gate antes de exigir o check e branch atualizada.
- [ ] Confirmar canal privado de segurança e processo CLA; não presumir bot.
- [ ] Configurar projeto, labels e milestones; evidenciar na issue #1.
- [ ] Sincronizar branch de governança com main atual e revisar conflitos, preservando Go 1.23 e alterações do core/desktop.
- [ ] Registrar CI, aprovação humana, squash merge e validação posterior. Não encerrar #1 antes disso.

## Projeto

Calango Product: projeto único da organização, inclusão automática de issues quando disponível. Status: Backlog, Refinamento, Pronta, Em andamento, Em revisão, Validação, Concluída. Bloqueio por dependência e label; preservar status. Prioridade P0 crítica, P1 bloqueia próximo marco, P2 importante, P3 futura. Tamanho XS até 2h, S meio dia, M 2 dias, L 5 dias, XL subdividir (referências humanas). Assignee, milestone e labels nativas, sem duplicar campos. Visões: Desenvolvimento, Roadmap, Bugs, Decisões, Releases, Minhas tarefas.

Labels propostas: type:epic, type:feature, type:bug, type:task, type:decision, type:spike, type:docs; area:core, area:desktop, area:mobile, area:cloud, area:ade, area:browser, area:security, area:teams, area:devops, area:website, area:product; good first issue, help wanted, needs triage, blocked, breaking change, retrospective. Nomes técnicos estáveis, descrições em português. Prioridade/tamanho/status apenas no Project.

Milestones propostos: v0.1.0 Fundação; v0.2.0 Assistente local; v0.3.0 Assistente conectado; v0.4.0 Assistente híbrido; v0.5.0 ADE; v0.6.0 Browser e automação; v0.7.0 Colaboração; v1.0.0 Disponibilidade geral. Confirmar escopo/tags antes de criar.

## CI

Preservar checks existentes. quality-gate exige sucesso de todos os jobs dependentes, inclusive sem skipped/cancelled. Evoluir com Staticcheck, race, govulncheck, CodeQL, detecção de segredos/dependências, desktop/mobile, assinatura e smoke. Privilégio mínimo; forks não recebem segredos.

## Transição e evidências

TODO e roadmap legados preservados. Issues retroativas registram recortes históricos e validações restantes, sem inventar PRs/aprovações. A migração de tarefas futuras ainda não foi executada. Ver DEV/AUDITORIA_RETROATIVA.md e issue #1. As traduções de documentos legados serão graduais. A base original do PR era dee8efa; a main auditada depois avançou para 1e8868b. A CI e configurações administrativas ainda precisam de confirmação.
