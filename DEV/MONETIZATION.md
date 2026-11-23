# Modelo de negócio — Calango (freemium)

Status: proposta inicial. Preços são hipóteses a validar, não decisões.

Atualização: o modelo híbrido (nó PC grátis, nó VPS pago, túnel via relay) está em [PRODUCT_HYBRID.md](PRODUCT_HYBRID.md). Esta tabela de planos é anterior e deve ser lida junto com aquele documento.

## Princípio

O custo que mais pesa é o de modelo (tokens) e o de execução na nuvem. O plano grátis precisa ser útil sem custar caro para nós: por isso o grátis roda local e usa a chave ou a assinatura do próprio usuário (BYOK).

## Planos (proposta)

| | Free | Pro | Team | Business |
|---|---|---|---|---|
| Quem | experimentar | pessoa que usa todo dia | pequeno time | empresa |
| Máquinas locais | 1 | ilimitadas | ilimitadas | ilimitadas |
| Nuvem (runner 24/7) | não | 1 runner | runners por pessoa | runners dedicados |
| Rotinas | 2 | ilimitadas | ilimitadas | ilimitadas |
| Canais | 1, só você | ilimitados | canais compartilhados | canais + permissões por papel |
| Projetos na ADE | 1 | ilimitados | ilimitados | ilimitados |
| Navegador do agente | básico, sem perfil persistente | perfil persistente | perfil por pessoa | perfil por pessoa + auditoria |
| Aprovações | só você | por dispositivo (push) | por pessoa | fluxo com dono e substituto |
| Créditos de modelo | BYOK | incluso + compra extra | incluso por assento | contrato |
| Suporte | comunidade | e-mail | prioritário | SLA |
| SSO, auditoria, retenção | não | não | opcional | sim |

## Monetização além da assinatura

1. **Créditos de uso**: tokens de modelos hospedados por nós, com markup sobre o custo. Compra avulsa quando o plano acaba.
2. **Minutos de runner**: execução 24/7 na nuvem cobrada por uso acima da cota.
3. **Armazenamento**: arquivos, memória e histórico acima da cota.
4. **Conectores premium**: integrações de terceiros com custo por chamada.
5. **Marketplace (futuro)**: rotinas e agentes prontos vendidos por criadores, com comissão.

## Como medir (base do plano)

Toda chamada passa por um gateway de modelo que registra tokens, custo e dono (ver [[ARCHITECTURE]]). O mesmo registro alimenta cobrança, limites do plano e painel de uso. Sem medição confiável, não há freemium viável.

## Limites e controles

- Cota por plano aplicada no núcleo (`Entitlements` por dono), não na interface.
- Quando estoura: o agente para e pede ação do dono; nunca cobra em silêncio.
- Limite de gasto por agente e por rotina, sempre visível.

## Pagamento

- **Web e desktop**: Stripe (assinatura e créditos).
- **iOS e Android**: a App Store e o Google Play exigem o próprio sistema de compra para assinatura digital consumida dentro do app, com comissão deles (faixa de 15 a 30%). Confirmar as regras vigentes antes de definir o preço no app. Alternativa: oferecer a compra só pela web e ler a assinatura no app (regra muda por loja e por país; checar).

## Riscos

- Plano grátis com modelo caro vira prejuízo: BYOK no grátis resolve isso.
- Comissão das lojas corrói margem no mobile: empurrar compra para a web onde a política permitir.
- Preço de referência (ChatGPT, Grok, Claude) cria expectativa de preço baixo; o diferencial precisa ser nuvem + ADE + controle, não "mais um chat".
- Dot foi encerrado (ver WORKLOG): assinatura de companion sem utilidade diária não se sustentou.

## Métricas iniciais

Conversão grátis→pago, retenção semanal, custo de modelo por usuário ativo, receita por usuário, churn, rotinas ativas por conta.

## Próximo passo

Validar com 5 a 10 pessoas do público-alvo: qual limite do grátis as faria pagar, e o que pagariam por nuvem contra o que já pagam em assinaturas de IA.
