/// <reference types="node" />
// WCAG 2.2 contrast of the UI kit text pairs, in both themes. The values come from theme.css
// (and packages/tokens for the primitives), so a theme change that drops a pair below 4.5:1
// fails here. Soft backgrounds are the color-mix() of theme.css over the card.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import tokens from '../../../../../packages/tokens/tokens.json' with { type: 'json' };

// Read from disk (relative to apps/desktop, the Vitest root): Vitest does not load CSS, not even with ?raw.
const css = readFileSync('src/lib/styles/theme.css', 'utf8');

type Theme = 'dark' | 'light';
type RGB = [number, number, number];

function block(selector: string): string {
  const start = css.indexOf(`${selector} {`);
  return css.slice(start, css.indexOf('}', start));
}

function themeVars(theme: Theme): Record<string, string> {
  const source = theme === 'dark' ? block('[data-theme="dark"]') : block('[data-theme="light"]');
  const vars: Record<string, string> = {};
  for (const [, name, value] of source.matchAll(/--cl-([a-z-]+):\s*([^;]+);/g)) {
    vars[name] = value.trim();
  }
  return vars;
}

function resolve(value: string): RGB {
  const primitive = /^var\(--calango-color-([a-z-]+)\)$/.exec(value);
  const hex = primitive ? tokens.color[primitive[1] as keyof typeof tokens.color].value : value;
  return [1, 3, 5].map((i) => Number.parseInt(hex.slice(i, i + 2), 16) / 255) as RGB;
}

function mix(fg: RGB, bg: RGB, amount: number): RGB {
  return fg.map((c, i) => c * amount + bg[i] * (1 - amount)) as RGB;
}

function contrast(a: RGB, b: RGB): number {
  const lum = (rgb: RGB) => {
    const [r, g, b2] = rgb.map((c) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
    return 0.2126 * r + 0.7152 * g + 0.0722 * b2;
  };
  const [hi, lo] = [lum(a), lum(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

const cases = (['dark', 'light'] as Theme[]).flatMap((theme) => {
  const v = themeVars(theme);
  const c = (name: string) => resolve(v[name]);
  const card = c('card');
  const leaf = resolve('var(--calango-color-leaf)');
  const tangerine = resolve('var(--calango-color-tangerine)');
  const postit = resolve('var(--calango-color-postit)');
  const pairs: [string, RGB, RGB][] = [
    ['texto no cartão', c('ink'), card],
    ['texto secundário no cartão', c('ink-soft'), card],
    ['texto secundário em bloco de código', c('ink-soft'), c('paper')],
    ['badge ok', c('moss-text'), mix(leaf, card, 0.2)],
    ['badge de alerta', c('warn-text'), mix(tangerine, card, 0.22)],
    ['badge e botão de perigo', c('danger-text'), mix(c('danger'), card, 0.2)],
    ['recibo de falha no cartão', c('danger-text'), card],
    ['botão primário', c('on-btn'), c('btn')],
    ['texto sobre post-it', c('on-tone'), postit],
  ];
  return pairs.map(([name, fg, bg]) => [theme, name, fg, bg] as const);
});

describe('contraste do UI kit', () => {
  it.each(cases)('%s: %s atinge 4.5:1', (_theme, _name, fg, bg) => {
    expect(contrast(fg, bg)).toBeGreaterThanOrEqual(4.5);
  });
});
