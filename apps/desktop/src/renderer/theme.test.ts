import { afterEach, describe, expect, it, vi } from 'vitest';

// theme.ts keeps the preference in module state, so each test loads a fresh copy.
async function loadTheme() {
  vi.resetModules();
  return import('./theme');
}

function mockSystem(light: boolean) {
  const listeners: (() => void)[] = [];
  const query = {
    matches: light,
    addEventListener: (_: string, listener: () => void) => listeners.push(listener),
  };
  vi.stubGlobal('matchMedia', () => query);
  return {
    change(next: boolean) {
      query.matches = next;
      for (const listener of listeners) listener();
    },
  };
}

describe('theme', () => {
  afterEach(() => {
    window.localStorage.clear();
    delete document.documentElement.dataset.theme;
    vi.unstubAllGlobals();
  });

  it('starts dark when nothing is saved', async () => {
    mockSystem(true);
    const theme = await loadTheme();
    theme.initTheme();
    expect(document.documentElement.dataset.theme).toBe('dark');
  });

  it('applies and saves the chosen theme', async () => {
    mockSystem(false);
    const theme = await loadTheme();
    theme.initTheme();
    theme.setThemePreference('light');
    expect(document.documentElement.dataset.theme).toBe('light');
    expect(window.localStorage.getItem('calango.theme')).toBe('light');

    const reloaded = await loadTheme();
    reloaded.initTheme();
    expect(document.documentElement.dataset.theme).toBe('light');
  });

  it('follows the operating system in "system"', async () => {
    const system = mockSystem(false);
    const theme = await loadTheme();
    theme.initTheme();
    theme.setThemePreference('system');
    expect(document.documentElement.dataset.theme).toBe('dark');
    system.change(true);
    expect(document.documentElement.dataset.theme).toBe('light');
  });
});
