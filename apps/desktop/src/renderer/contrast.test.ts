// WCAG 2.2 contrast of the color pairs styles.css uses. A token change that drops a pair
// below the threshold fails here. Body text needs 4.5:1; the focus outline needs 3:1.
import { describe, expect, it } from 'vitest';
import tokens from '../../../../packages/tokens/tokens.json' with { type: 'json' };

type ColorToken = keyof typeof tokens.color;

function luminance(hex: string): number {
  const channels = [1, 3, 5].map((i) => Number.parseInt(hex.slice(i, i + 2), 16) / 255);
  const [r, g, b] = channels.map((c) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(fg: ColorToken, bg: ColorToken): number {
  const [a, b] = [luminance(tokens.color[fg].value), luminance(tokens.color[bg].value)].sort((x, y) => y - x);
  return (a + 0.05) / (b + 0.05);
}

const pairs: [string, ColorToken, ColorToken, number][] = [
  ['texto na página', 'paper', 'night', 4.5],
  ['texto em cartão e campo', 'paper', 'night-card', 4.5],
  ['texto secundário (.muted) em cartão', 'sand', 'night-card', 4.5],
  ['texto secundário (.muted) na página', 'sand', 'night', 4.5],
  ['erro (.error) em cartão', 'tangerine', 'night-card', 4.5],
  ['erro (.error) na página', 'tangerine', 'night', 4.5],
  ['botão', 'ink', 'postit', 4.5],
  ['botão de perigo', 'paper', 'clay-deep', 4.5],
  ['mensagem da pessoa', 'ink', 'postit-light', 4.5],
  ['contorno de foco na página', 'sand', 'night', 3],
  ['contorno de foco em cartão', 'sand', 'night-card', 3],
];

describe('contraste dos tokens', () => {
  it.each(pairs)('%s (%s sobre %s) atinge %s:1', (_name, fg, bg, min) => {
    expect(contrast(fg, bg)).toBeGreaterThanOrEqual(min);
  });
});
