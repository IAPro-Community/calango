# ADR 0005 — Tailscale como transporte inicial

Status: aceita · 2026-11-23

## Contexto
Celular precisa alcançar o PC (atrás de NAT) e a VPS de qualquer lugar, com criptografia ponta a ponta, sem a pessoa configurar roteador.

## Decisão
Fases 2–6 usam Tailscale (tailnet da própria pessoa). Avaliar `tsnet` (biblioteca Go) para embutir no `calango` e dispensar o cliente separado no PC e na VPS. Relay próprio fica para a fase 7.

## Pendências antes de implementar
- Confirmar licença e termos do `tsnet` para uso embutido em produto comercial.
- Confirmar limites atuais do plano gratuito para uso pessoal.

## Consequências
- Sem custo de relay para nós nesta fase; monetização vem de licença VPS, créditos, VPS gerenciada.
- Dependência de terceiro: o transporte fica atrás de uma interface (`internal/transport`) para trocar pelo relay próprio.
