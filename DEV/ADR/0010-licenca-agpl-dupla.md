# ADR 0010 — Código aberto sob AGPL-3.0, com licença comercial, CLA e marca

Status: aceita · 2026-11-23 · publicação em `IAPro-Community`

## Contexto
O projeto será construído com a comunidade, mas é um negócio: o dono lidera e a receita vem de serviços e de licenças comerciais. Um código permissivo permitiria que terceiros hospedassem uma cópia concorrente sem contribuir de volta. O projeto também depende de contribuições de terceiros, o que exige cessão de direitos para manter a licença comercial.

## Decisão
1. **Licença do código:** GNU AGPL-3.0 (`LICENSE`).
2. **Licença comercial** para quem não pode ou não quer cumprir a AGPL (`COMMERCIAL.md`).
3. **CLA** obrigatório para contribuições (`CLA.md`): licença de direitos e de patentes, com direito de relicenciar.
4. **Marca** protegida separadamente (`TRADEMARKS.md`).
5. **Serviços pagos fora do repositório:** relay, push, VPS gerenciada, créditos de modelo, billing e licenciamento ficam em serviços operados pela Calango. O binário aberto não contém a lógica de cobrança como barreira de acesso.
6. **Dependências:** o Orquestrador Maestro (Apache-2.0) é consumido via npm. Apache-2.0 é compatível com AGPL-3.0 nesse uso.

## Alternativas descartadas
- **Apache-2.0 / MIT:** permitiria fork hospedado concorrente sem obrigação de publicar.
- **GPL-3.0:** não cobre uso como serviço em rede.
- **Fonte disponível (BSL, SSPL, FSL):** não são licenças open source aprovadas pela OSI; o dono pediu código aberto.

## Riscos aceitos
- Empresas que evitam AGPL podem preferir a licença comercial. Isso é receita, não só custo.
- A verificação de licença da VPS pode ser removida por quem compila o código. Por isso o que é cobrado fica fora do binário (item 5).
- O CLA exige revisão jurídica antes da publicação.

## Pendências
- Revisão jurídica de `CLA.md` e `COMMERCIAL.md`.
- Endereço de contato, termos de serviço e política de privacidade reais.
- Mecanismo de assinatura do CLA no PR (bot ou checkbox).
