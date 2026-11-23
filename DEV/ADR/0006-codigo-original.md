# ADR 0006 — Código original; referências só para leitura

Status: aceita · 2026-11-23

## Contexto
O produto se inspira em Tucano (Apache-2.0) e agentes.chat (AGPL-3.0, com marca registrada e CLA).

## Decisão
Nenhum código, texto, teste, prompt ou asset dos projetos de referência entra neste repositório. Eles servem só para entender produto e problemas (issues, changelog, ADRs). As specs descrevem comportamento com palavras próprias.

## Exceção
O Orquestrador Maestro (Apache-2.0) é consumido pelo pacote npm oficial, conforme [ADR 0009](0009-orquestrador-maestro.md). Não é cópia de código.

## Consequências
- Liberdade total de licença para o Calango.
- Mais trabalho de implementação; aceito.
- Revisão de PR deve recusar trechos com cara de cópia.
