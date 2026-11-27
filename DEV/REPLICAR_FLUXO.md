# Replicar o fluxo de entrega em outro projeto

Use o [template IAPro · Entrega por ondas](https://github.com/orgs/IAPro-Community/projects/2) para copiar o quadro. Cada produto define suas ondas, gates, prioridades e responsáveis. No Calango, a sequência de aceite está em [Ondas de entrega](ONDAS_DE_ENTREGA.md).

## Configurar o destino

1. Crie o Project a partir do template. Vincule o repositório e o time responsáveis e confira permissões. Ajuste as datas das iterações semanais e renomeie as ondas para as entregas do produto.
2. Ative **Auto-add to project** no quadro de destino, selecione o repositório e configure o filtro `is:issue is:open`. Importe separadamente as issues abertas que já existiam. Confira quais automações alteram status e quando elas agem.
3. Adapte os formulários e o template de PR da pasta `.github`, os checks de CI, as labels e os milestones. Configure as branches e os rulesets do repositório de destino, com revisão independente e nomes de checks iguais aos produzidos pelo CI. Mantenha as credenciais no destino definido pela organização.
4. Crie um gate por onda e vincule os pré-requisitos por dependências nativas. Defina o aceite com resultado demonstrável e responsável humano. Classifique as issues abertas. Use **Pronta** após refinamento e liberação da onda. Na Sprint atual, distribua trabalho conforme a disponibilidade semanal, incluindo revisão e pareamento.

## Testar a replicação

Em um repositório piloto autorizado, abra uma issue de teste e observe se ela entra no quadro automaticamente. Crie uma dependência entre duas issues e confira se o responsável consegue identificar o bloqueio e quem pode ajudar. Coloque uma tarefa pronta na iteração vigente e confira a visão Sprint atual. Abra um PR de teste e verifique se os gates exigem os checks corretos e revisão independente antes de permitir integração. Registre o resultado no repositório piloto.
