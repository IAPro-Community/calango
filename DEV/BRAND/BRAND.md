# Marca — Calango

## Logo

Arquivo original: [logo-original.png](logo-original.png) (1254×1254). Versões reduzidas em `site/assets/`.

Leitura do estilo:
- **Colagem de papel recortado.** O calango é feito de pedaços de papel com bordas retas e cantos irregulares, levemente sobrepostos, com textura de fibra.
- **Fundo de post-its.** Quadrados amarelos sobrepostos, em tons um pouco diferentes, como uma parede de ideias.
- **Tile de vidro líquido.** Tudo dentro de um quadrado de cantos bem arredondados, com borda translúcida, reflexo branco no topo e brilho amarelo nas bordas.
- **Paleta quente e solar:** amarelo post-it, verdes limão/folha/musgo, faixa laranja, barriga areia, rocha argila, olho marrom-escuro.
- **Postura:** o calango está de cabeça erguida, atento, em cima da pedra. Representa um assistente alerta, sempre de guarda.

## Princípios visuais

1. **Feito à mão, acabamento premium.** Formas de papel recortado com sombras macias; vidro só em destaques (tile do ícone, cartões principais).
2. **Pedaços que se juntam.** A metáfora central: pedaços de papel (tarefas, nós, rotinas) se juntando para formar o calango. Usar isso em animações e diagramas.
3. **Calor, não neon.** Sem roxo, sem gradiente de IA genérico. Amarelo, verde e terra.
4. **Leve inclinação.** Elementos de papel com rotação pequena (−3° a 3°), nunca alinhados demais.

## Paleta

Ver `packages/tokens/tokens.json`. Principais: `postit #FDD838`, `lime #B6E021`, `leaf #6CC23A`, `moss #42912E`, `tangerine #FE9904`, `sand #F3CC85`, `clay #B47133`, `bark #4A2A15`, `paper #FFFBEF`, `ink #22180F`.

Contraste: texto sempre `ink` sobre fundos claros e `paper` sobre `moss`/`bark`/`night`. Nunca texto branco sobre amarelo.

## Tipografia

- Títulos: **Bricolage Grotesque** 700–800, espaçamento apertado, tamanho grande.
- Texto: **Instrument Sans** 400–600.
- Código: **JetBrains Mono**.

## Movimento

- **Assentar papel:** elemento entra com pequena rotação e escala 0,96 → 1, curva `ease-paper` (passa levemente do ponto e volta).
- **Escalonamento:** grupos entram em sequência de 45 ms.
- **Relógio único** nas demos de produto: um só tempo dirige todas as cenas, permitindo pausar, avançar e repetir.
- `prefers-reduced-motion`: sem deslocamento nem rotação; só opacidade.

## Ícone em tamanhos pequenos

A colagem perde leitura abaixo de 48 px. Pendente: versão simplificada (silhueta do calango em `moss` sobre `postit`, sem textura) para favicon, bandeja e notificações.
