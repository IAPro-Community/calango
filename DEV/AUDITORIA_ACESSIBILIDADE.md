# Auditoria de acessibilidade — desktop e site

Issue: #48 (X-A11y). Data: 25/11/2026. Base: `develop` no momento da auditoria.

Escopo: app desktop (`apps/desktop/src/renderer`) e site (`site/index.html`). O app mobile ainda não existe e fica para depois da F2-T04.

Referência: WCAG 2.2, nível AA.

## Método e ferramentas

| Verificação | Como |
|---|---|
| Regras automáticas | axe-core 4.10.3 com as tags `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`, `wcag22aa` e `best-practice`, em todas as telas do desktop e no site (claro e escuro, pt-BR, en e es) |
| Árvore de acessibilidade | Playwright 1.59.1 (`_electron` no desktop, Chromium no site): `ariaSnapshot()` de cada tela e um fluxo completo feito só com `getByRole` e nomes acessíveis (criar conta, criar agente, abrir conversa, enviar mensagem) |
| Teclado | Tab a partir do menu em cada tela, registrando elemento focado, nome acessível e estilo de foco; Enter nos itens do menu |
| Contraste | razão WCAG calculada sobre os pares de `packages/tokens/tokens.json` usados em `styles.css` (texto 4,5:1; bordas de campo e foco 3:1) e, no site, pelo axe nos dois temas |
| Idioma | `document.documentElement.lang` com o sistema em pt-BR e en no desktop e com `?lang=pt|en|es` no site |

O desktop rodou em Linux com Xvfb, daemon real e diretórios temporários (`HOME`, `XDG_CONFIG_HOME`, `CALANGO_DATA_DIR`, `CALANGO_VAULT_KEY` aleatória e `CALANGO_ADDR` em porta livre). Não houve teste com leitor de tela real.

## Severidade

- **Bloqueador**: impede uma tarefa para quem usa teclado ou leitor de tela.
- **Alta**: a tarefa é possível, mas com informação errada ou ausente.
- **Média**: atrapalha ou viola um critério AA sem impedir a tarefa.
- **Baixa**: melhoria ou boa prática.

## Achados no desktop

| # | Tela | Achado | Critério | Severidade | Situação |
|---|---|---|---|---|---|
| D1 | Conversas | Mensagens da conversa sem autor para leitor de tela (o autor só aparece pela cor e posição) e sem região ao vivo: a resposta do agente chega sem ser anunciada | 1.3.1, 4.1.3 | Bloqueador | Corrigido: `role="log"` com nome, prefixo oculto "Você/Agente/Sistema" em cada mensagem |
| D2 | Agentes, Aprovações, Conversas, Rotinas, Uso | Mensagens de erro sem `role="alert"`; o erro aparece, mas não é anunciado | 4.1.3 | Alta | Corrigido nas cinco telas |
| D3 | Conversas | Campo de título da nova conversa rotulado com o texto do campo de mensagem ("Escreva uma mensagem...") | 2.4.6, 3.3.2 | Alta | Corrigido: rótulo "Título (opcional)" |
| D4 | Todas | Borda de `input`, `select` e `textarea` em `ink-soft`: 1,98:1 sobre `night-card` e 2,19:1 sobre `night` | 1.4.11 | Alta | Corrigido: borda em `clay` (4,24:1 e 4,71:1) |
| D5 | Conversas | Campo de mensagem sem rótulo, só placeholder | 3.3.2 | Média | Corrigido: `aria-label` "Mensagem" |
| D6 | Conversas | Estado da execução (na fila, rodando, falhou...) muda sem anúncio | 4.1.3 | Média | Corrigido: elemento `<output>` (papel `status`) |
| D7 | Inicialização | Telas "Carregando", "Tentando de novo" e "Indisponível" do daemon sem anúncio | 4.1.3 | Média | Corrigido: `aria-live="polite"` |
| D8 | Todas | `<html lang="pt-BR">` fixo, mesmo com a interface em en ou es | 3.1.1 | Média | Corrigido: `lang` acompanha o idioma detectado |
| D9 | Todas | Ao escolher uma seção no menu, o foco fica no menu; quem usa teclado atravessa o menu de novo a cada troca | 2.4.3 | Média | Corrigido: foco vai para `<main>` |
| D10 | Menu e lista de conversas | Item ativo diferenciado só por fundo `night-card` sobre `night` (cerca de 1,1:1) | 1.4.11 | Média | Corrigido: marca amarela à esquerda; `aria-current` na lista de conversas |
| D11 | Conversas | Sem título de nível 1 e com `aside` dentro de `main` | 1.3.1, boa prática | Média | Corrigido: `h1` visualmente oculto e `div` no lugar do `aside` |
| D12 | Uso e custos | Opções de agrupamento com identificadores da API em inglês (`agent`, `routine`...) | 3.1.2 | Baixa | Corrigido: textos traduzidos |
| D13 | Agentes | Lista de contas renderizada vazia (`<ul>` sem itens) | boa prática | Baixa | Pendente |
| D14 | Várias | Botão de perigo (`clay-deep`) com 2,71:1 contra o card; o texto do botão tem 5,94:1 e identifica o controle | 1.4.11 | Baixa | Pendente, aceito por ora |
| D15 | Uso e custos | Custos em micros sem unidade e tabela sem legenda | 1.3.1 | Baixa | Pendente |

Verificado sem problema: foco visível (contorno `sand` de 3 px) em todos os controles; ordem de Tab segue a ordem visual; nenhum controle sem nome acessível; os únicos diálogos são o `window.confirm` nativo da exclusão de segredo e os `select` nativos, que fecham com Esc; texto principal, texto secundário, erros e botões passam de 4,5:1.

Depois das correções, o axe não aponta violações em nenhuma das sete telas, vazias ou com dados.

## Achados no site

| # | Achado | Critério | Severidade | Situação |
|---|---|---|---|---|
| S1 | Comentário de código no exemplo de IDE com 4,33:1 (`#8C7C62` sobre `#1E1810`), nos dois temas | 1.4.3 | Média | Corrigido: `#A0907A` (5,67:1) |
| S2 | `aside` dentro da janela de demonstração, que já é `role="img"` | boa prática | Baixa | Corrigido: `div` |
| S3 | Sem link "pular para o conteúdo"; são 7 links e 5 controles antes do conteúdo (landmarks e títulos cobrem o 2.4.1) | 2.4.1 | Baixa | Pendente |
| S4 | Botão de tema sem estado (`aria-pressed` ou rótulo com o tema atual) | 4.1.2 | Baixa | Pendente |

Verificado sem problema: `lang` muda para `pt-BR`, `en` e `es`; imagens decorativas com `alt=""` e logo final com texto alternativo traduzido; ilustrações complexas com `role="img"` e `aria-label`; foco visível em todos os elementos, sem foco dentro de `aria-hidden`; ordem de Tab lógica; demonstração com botão de pausa e respeito a `prefers-reduced-motion`; contraste AA nos temas claro e escuro, exceto S1.

Fora de acessibilidade: o botão de cena "3 · Pede sua ok" tem erro de concordância ("seu ok").

## Tokens fora do desktop

O desktop só tem tema escuro. No claro, `moss` (3,81:1) e `tangerine` (2,08:1) sobre `paper` não servem como texto. O site já usa variantes próprias (`--moss-text`, `--warn-text`); o futuro tema claro e o app mobile devem fazer o mesmo.

## Pendências

- Teste com leitor de tela real: NVDA e Narrador no Windows, VoiceOver no macOS e Orca no Linux. A árvore do Chromium mostra nomes, papéis e regiões ao vivo, mas não garante a leitura de cada leitor.
- Modo de alto contraste do Windows (`forced-colors`) e zoom de 200% no desktop.
- D13, D14, D15, S3 e S4.
- Mobile, depois da F2-T04.

## Testes automáticos

`apps/desktop/test/a11y.test.cjs` roda em `npm run test:main` (e, portanto, em `npm test` e `npm run quality`). Ele verifica:

- contraste dos pares de tokens do tema escuro (texto 4,5:1; bordas de campo e foco 3:1);
- `role="alert"` em todo elemento com `className="error"`;
- rótulo (`<label>` ou `aria-label`) em todo `input`, `select` e `textarea`;
- `lang` do documento ligado ao idioma da interface;
- um `h1` por tela.

São verificações estáticas e não substituem o teste no app.

## Como repetir

1. `npm ci` e `npm run desktop:build`.
2. Numa pasta temporária fora do repositório: `npm i playwright@1.59.1 axe-core@4.10.3` e `npx playwright install chromium`.
3. Desktop: num script Node, `_electron.launch({ args: ['<repo>/apps/desktop'], env })` com `HOME`, `XDG_CONFIG_HOME` e `CALANGO_DATA_DIR` temporários, `CALANGO_VAULT_KEY` de `head -c32 /dev/urandom | base64` e `CALANGO_ADDR=127.0.0.1:<porta livre>`; no Linux sem tela, rode com `xvfb-run -a`. A política de conteúdo do renderer bloqueia `addScriptTag`, então injete o axe com `page.evaluate` do conteúdo de `axe-core/axe.min.js`. Para cada botão do menu: clique, rode `axe.run`, guarde `page.locator('main').ariaSnapshot()` e percorra a tela com Tab.
4. Site: `python3 -m http.server 8765 --directory site` e, com Playwright Chromium, abra `/?lang=pt|en|es` com `colorScheme` claro e escuro, rode o axe e percorra a página com Tab.
5. Repita o fluxo de conversa só com `getByRole` (conta Codex, agente, nova conversa, mensagem). Sem login do Codex a execução falha com `harness_signed_out`, o que serve para conferir o anúncio de estado.
