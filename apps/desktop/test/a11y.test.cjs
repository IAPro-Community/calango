// Verificações estáticas de acessibilidade do renderer (issue #48).
// Não substituem o teste com leitor de tela; evitam regressões nos achados da auditoria
// registrados em DEV/AUDITORIA_ACESSIBILIDADE.md.
const assert = require('node:assert/strict');
const { readFileSync, readdirSync } = require('node:fs');
const { test } = require('node:test');
const path = require('node:path');

const repoRoot = path.resolve(__dirname, '..', '..', '..');
const rendererDir = path.join(__dirname, '..', 'src', 'renderer');
const colors = JSON.parse(readFileSync(path.join(repoRoot, 'packages', 'tokens', 'tokens.json'), 'utf8')).color;

function luminance(token) {
  const value = Number.parseInt(colors[token].value.slice(1), 16);
  const [r, g, b] = [(value >> 16) & 255, (value >> 8) & 255, value & 255].map((channel) => {
    const c = channel / 255;
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(foreground, background) {
  const [light, dark] = [luminance(foreground), luminance(background)].sort((a, b) => b - a);
  return (light + 0.05) / (dark + 0.05);
}

function rendererSources() {
  const screens = readdirSync(path.join(rendererDir, 'screens'))
    .filter((name) => name.endsWith('.tsx'))
    .map((name) => path.join(rendererDir, 'screens', name));
  return [path.join(rendererDir, 'App.tsx'), ...screens].map((file) => ({
    name: path.relative(rendererDir, file),
    // Comentários de linha citam tags (ex.: <input type="date">) e não são marcação.
    source: readFileSync(file, 'utf8').replace(/^\s*\/\/.*$/gm, ''),
  }));
}

// Pares usados pelo tema escuro do desktop (styles.css).
const TEXT_PAIRS = [
  ['paper', 'night'],
  ['paper', 'night-card'],
  ['sand', 'night'],
  ['sand', 'night-card'],
  ['tangerine', 'night'],
  ['tangerine', 'night-card'],
  ['ink', 'postit'],
  ['ink', 'postit-light'],
  ['paper', 'clay-deep'],
];

// Bordas de campos e indicador de foco: componentes de interface pedem 3:1.
const UI_PAIRS = [
  ['clay', 'night'],
  ['clay', 'night-card'],
  ['sand', 'night'],
  ['sand', 'night-card'],
  ['postit', 'night-card'],
];

test('texto do tema escuro atinge contraste WCAG AA (4,5:1)', () => {
  for (const [foreground, background] of TEXT_PAIRS) {
    const ratio = contrast(foreground, background);
    assert.ok(ratio >= 4.5, `${foreground} sobre ${background}: ${ratio.toFixed(2)}`);
  }
});

test('bordas de campo e foco atingem 3:1 contra o fundo', () => {
  for (const [foreground, background] of UI_PAIRS) {
    const ratio = contrast(foreground, background);
    assert.ok(ratio >= 3, `${foreground} sobre ${background}: ${ratio.toFixed(2)}`);
  }
  const css = readFileSync(path.join(rendererDir, 'styles.css'), 'utf8');
  assert.match(css, /input,\s*select,\s*textarea\s*\{\s*border-color:\s*var\(--calango-color-clay\)/);
});

test('mensagens de erro são anunciadas com role="alert"', () => {
  for (const { name, source } of rendererSources()) {
    for (const match of source.matchAll(/<(p|span|div)\b[^>]*className="error"[^>]*>/g)) {
      assert.match(match[0], /role="alert"/, `${name}: ${match[0]}`);
    }
  }
});

test('campos de formulário têm rótulo acessível', () => {
  for (const { name, source } of rendererSources()) {
    for (const match of source.matchAll(/<(input|select|textarea)\b/g)) {
      const before = source.slice(0, match.index);
      const insideLabel = before.lastIndexOf('<label') > before.lastIndexOf('</label>');
      const tag = source.slice(match.index, source.indexOf('>', match.index));
      assert.ok(insideLabel || /aria-label(ledby)?=/.test(tag), `${name}: <${match[1]}> sem rótulo`);
    }
  }
});

test('o idioma do documento acompanha o idioma da interface', () => {
  const app = readFileSync(path.join(rendererDir, 'App.tsx'), 'utf8');
  assert.match(app, /document\.documentElement\.lang = locale/);
});

test('cada tela tem um título de nível 1', () => {
  for (const { name, source } of rendererSources()) {
    if (name.startsWith('screens') && name.endsWith('Screen.tsx')) {
      assert.match(source, /<h1\b/, `${name} sem <h1>`);
    }
  }
});
