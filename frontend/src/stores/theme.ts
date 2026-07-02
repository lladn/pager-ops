import { writable } from 'svelte/store';

export type ThemeName = 'light' | 'dark';

const THEME_KEY = 'pagerops_theme';
const DEFAULT_THEME: ThemeName = 'dark';

function loadTheme(): ThemeName {
    try {
        const stored = localStorage.getItem(THEME_KEY);
        if (stored === 'light' || stored === 'dark') {
            return stored;
        }
    } catch (err) {
        console.error('Failed to load theme preference:', err);
    }
    return DEFAULT_THEME;
}

// Shared theme store. Defaults to dark (Radar-style dark grey/purple).
export const theme = writable<ThemeName>(loadTheme());

// Reflect the active theme on <html data-theme="..."> so CSS variables in
// style.css can switch, and persist the choice across restarts.
theme.subscribe((value) => {
    if (typeof document !== 'undefined') {
        document.documentElement.setAttribute('data-theme', value);
    }
    try {
        localStorage.setItem(THEME_KEY, value);
    } catch (err) {
        console.error('Failed to save theme preference:', err);
    }
});

export function toggleTheme(): void {
    theme.update((current) => (current === 'dark' ? 'light' : 'dark'));
}

export function setTheme(value: ThemeName): void {
    theme.set(value);
}
