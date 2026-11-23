# Produto híbrido — PC, VPS e celular

Status: proposta de produto. Nada disto está implementado.

## Ideia central

Calango é **um agente com vários nós**. Cada instalação é um nó, e a pessoa escolhe onde cada coisa roda:

- **Nó PC** (grátis): instala no computador. Tem acesso aos arquivos, apps e navegador locais.
- **Nó VPS** (pago): a mesma instalação, na VPS da pessoa, via Docker. Roda 24/7, executa rotinas e projetos sem depender do PC ligado.
- **Clientes** (celular, desktop, web): não executam agente. Falam com os nós da conta e mostram o que está acontecendo em qualquer um deles.

A conta é a identidade. Os nós são da pessoa. O que sai dos nós para a nuvem da Calango é só o mínimo necessário para conectar e cobrar.

## Os três modos de conexão

| Caminho | Quando | Quem vê o conteúdo |
|---|---|---|
| **LAN direta** | celular e PC na mesma rede (mDNS) | só os dispositivos |
| **Túnel via relay** | fora de casa; o PC abre conexão de saída para o relay, sem abrir porta no roteador | ninguém além das pontas (criptografia ponta a ponta) |
| **VPS direta** | VPS com domínio e TLS | só os dispositivos |

O túnel é a peça-chave. O PC (ou a VPS) abre uma conexão **de saída** para o relay da Calango, e o celular se conecta ao relay. O relay só encaminha bytes criptografados entre os dois. Assim funciona atrás de qualquer roteador, sem configurar nada.

## Trabalho híbrido

- **Onde roda**: cada tarefa ou rotina tem um destino: PC, VPS ou automático. Automático = PC se estiver online, senão VPS (se a pessoa tiver VPS). Sem VPS, a rotina espera o PC voltar.
- **Projetos (ADE)**: o código fica em git. PC e VPS trabalham em worktrees do mesmo repositório e sincronizam por `push`/`pull`. Não inventar sync de código próprio.
- **Memória e conversas**: registro de eventos append-only por nó, replicado entre os nós da conta. Cada registro tem o nó de origem, então não há conflito de autoria.
- **Controle remoto do PC pelo celular**: o celular manda comandos ao PC pelo túnel, e o PC executa. Ação sensível continua exigindo aprovação no celular.

## Controle remoto do PC (o que a pessoa paga para ter)

Pelo celular, com o PC ligado:
- ver o estado dos agentes, rotinas e tarefas;
- aprovar ou recusar ações sensíveis;
- enviar tarefas e mensagens;
- ver a tela do PC e controlar mouse e teclado (opt-in explícito, com aviso na tela do PC e registro de auditoria).

Controle remoto de tela é o recurso de maior risco. Ele fica desligado por padrão e precisa ser ligado no PC.

## Segurança (não negociável)

- Criptografia ponta a ponta entre celular e nó. A chave nasce no pareamento e fica no keychain do dispositivo. O relay não consegue ler conteúdo.
- Pareamento por código de uso único, com revogação por dispositivo.
- Nó PC e VPS nunca aceitam conexão sem token de dispositivo válido, mesmo na LAN.
- Auditoria de toda ação remota, visível no celular e no PC.
- Ação de escrita sensível sempre pede aprovação, venha de onde vier.

## Monetização

A regra é: **o que roda local é grátis; o que exige infraestrutura nossa ou tempo de servidor é pago.** Assim o grátis é útil de verdade e o pago tem motivo claro.

### Planos

| | Free | Pro | Team |
|---|---|---|---|
| Nó PC | 1 | 1 | ilimitados por membro |
| Nó VPS (extensão) | não | 1 | 1 por membro |
| Celular na LAN | sim | sim | sim |
| Túnel via relay: controle e status | cota diária pequena | ilimitado | ilimitado |
| Controle de tela e arquivos pelo túnel | não | sim | sim |
| Execução híbrida (PC ↔ VPS) | não | sim | sim |
| Rotinas na VPS 24/7 | não | sim | sim |
| Push no celular | não | sim | sim |
| Canais compartilhados | não | não | sim |

### Add-ons (cobrados por uso)

1. **Créditos de modelo** (BYOK no grátis; comprado no Pro).
2. **Relay além da cota**: tráfego de arquivos e tela, por GB.
3. **VPS gerenciada por nós**: a mesma VPS, sem a pessoa administrar.
4. **Armazenamento** de arquivos e histórico acima da cota.
5. **Conectores premium** com custo por chamada.

### Como cobrar a extensão VPS

- Quem paga recebe uma **chave de licença**. A imagem Docker do nó VPS só aceita a chave válida para os recursos pagos.
- A validação é **assinada offline**: o nó confere a assinatura localmente e consulta o servidor de licenças de tempos em tempos. Se a internet cair, há um período de tolerância. Nada do que é grátis para de funcionar.
- Sem chave, a VPS roda só o que é grátis, ou nem sobe. Decidir isso com cuidado: derrubar um servidor de produção da pessoa por falta de pagamento é ruim para a confiança do produto. Preferível: rebaixar para o modo grátis e avisar.

### Por que o usuário paga

- Quer o agente rodando quando o PC está desligado.
- Quer controlar o PC de fora sem configurar roteador.
- Quer trabalhar em projetos na VPS e no PC ao mesmo tempo.
- Quer notificação no celular mesmo com tudo fechado.

### Custos que a Calango tem (e cobra por eles)

- Relay (tráfego de túnel): cobrado por uso acima da cota.
- Push (APNs/FCM via relay): incluso no Pro, limite no Free.
- Servidor de licenças: barato, incluso.
- Modelos hospedados: créditos com markup.

## O que construir primeiro (ordem)

1. **Nó PC com pareamento e LAN** (grátis; prova o produto).
2. **Cliente celular** conectado ao nó PC na LAN.
3. **Relay com túnel de saída** e cota grátis de controle.
4. **Chave de licença e nó VPS** (primeiro item pago).
5. **Execução híbrida** (escolha de destino e replicação de eventos).
6. **Controle de tela** pelo túnel (pago).

## Riscos

- **Relay custa dinheiro e vira gargalo**: limitar no grátis desde o início.
- **Controle remoto de PC é superfície de ataque**: auditoria e opt-in são obrigatórios antes de liberar.
- **Código do nó VPS** é o que a pessoa roda; decidir se é código aberto (com licença por chave) ou fonte disponível. Não copiar código dos repos de referência.
- **Lojas de apps**: confirmar que o app de controle remoto aprova nas duas lojas, e que a assinatura passa pelo sistema de compra delas (ver [MONETIZATION.md](MONETIZATION.md)).
- **Sincronização de memória entre nós**: é a parte mais difícil de acertar. Começar com replicação simples e um nó "dono" por registro.

## Decisão: transporte por enquanto é Tailscale

Decidido em 2026-11-23: a fase inicial usa **Tailscale** (rede privada WireGuard) como túnel, em vez de um relay próprio.

- Celular, nó PC e nó VPS entram na mesma tailnet da pessoa. Conexão ponta a ponta, sem porta aberta e sem relay nosso.
- Alternativa de implementação: **tsnet** (biblioteca Go do Tailscale) embutida no `calango`, para o nó entrar na tailnet sem a pessoa instalar o cliente separado. Confirmar licença e termos antes de embutir.
- Consequência para o modelo de negócio: o túnel deixa de ser cobrado por tráfego nosso nesta fase. A monetização passa a ser a licença da VPS, os créditos de modelo, a VPS gerenciada e os conectores. A cota de relay fica para quando existir relay próprio.
- Custo do usuário: precisa de conta Tailscale. Confirmar limites do plano gratuito deles para uso pessoal antes de anunciar "grátis".
- Relay próprio continua no roteiro, para quem não quer depender de terceiros e para o controle de tela pago.

Perguntas que continuam em aberto:
- Nó VPS aberto ou fonte disponível?
- Controle de tela entra no Pro ou só no Team?
