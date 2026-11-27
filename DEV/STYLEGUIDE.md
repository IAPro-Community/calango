# Styleguide do Calango

A fonte de cores, tipografia, escalas e espaçamento é [packages/tokens/tokens.json](../packages/tokens/tokens.json). Consulte a [identidade visual](BRAND/BRAND.md), o [Figma](https://www.figma.com/design/vnna5rthWR8aXWBRYo85Tb) e a [issue #175](https://github.com/IAPro-Community/calango/issues/175) para rastrear as decisões. A modelagem e o manifesto do Figma estão no [PR #176](https://github.com/IAPro-Community/calango/pull/176).

## Abrir o catálogo

Na raiz da worktree baseada em develop:

```bash
npm ci
npm exec --workspace apps/desktop -- vite --host 127.0.0.1 --port 5173 --strictPort
```

Abra [o catálogo local](http://127.0.0.1:5173/catalog.html). Ele usa os componentes de [apps/desktop/src/lib](../apps/desktop/src/lib) e tem entrada em [catalog.html](../apps/desktop/catalog.html).

## Critérios de aceite e adoção

- Conferir temas escuro e claro solar e os estados aplicáveis dos componentes essenciais, incluindo foco, desabilitado e erro.
- Verificar navegação por teclado, contraste e legibilidade. Registrar no #175 variantes ainda necessárias para o MVP Web.
- Manter os tokens compartilhados como fonte única e documentar no catálogo os novos estados e variantes antes de aplicá-los em telas.
- Executar `npm run contracts:check` e `npm run test:renderer -w apps/desktop`. Registrar a versão, a evidência e o aceite humano em #175.
