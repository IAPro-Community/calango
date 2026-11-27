import { useSyncExternalStore } from 'react';

// Theme preference of the desktop. The kit reads data-theme on <html> (theme.css):
// "dark" is the default of the Figma D1 screens, "light" is the solar theme of the site.
export type ThemePreference = 'dark' | 'light' | 'system';
export type Theme = 'dark' | 'light';

const STORAGE_KEY = 'calango.theme';
const LIGHT_QUERY = '(prefers-color-scheme: light)';

const listeners = new Set<() => void>();
let preference: ThemePreference = readStored();

function readStored(): ThemePreference {
  try {
    const value = window.localStorage.getItem(STORAGE_KEY);
    return value === 'light' || value === 'system' ? value : 'dark';
  } catch {
    return 'dark';
  }
}

function systemTheme(): Theme {
  return window.matchMedia?.(LIGHT_QUERY).matches ? 'light' : 'dark';
}

export function resolveTheme(value: ThemePreference): Theme {
  return value === 'system' ? systemTheme() : value;
}

function apply(): void {
  document.documentElement.dataset.theme = resolveTheme(preference);
}

/** initTheme applies the saved theme before the first render and follows the OS in "system". */
export function initTheme(): void {
  apply();
  window.matchMedia?.(LIGHT_QUERY).addEventListener('change', () => {
    if (preference === 'system') apply();
  });
}

export function setThemePreference(next: ThemePreference): void {
  preference = next;
  try {
    window.localStorage.setItem(STORAGE_KEY, next);
  } catch {
    // Sem armazenamento, o tema vale só nesta sessão.
  }
  apply();
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function useThemePreference(): ThemePreference {
  return useSyncExternalStore(subscribe, () => preference);
}
