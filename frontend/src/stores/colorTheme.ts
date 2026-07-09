import { writable } from 'svelte/store';

export type ColorThemeName =
    | 'default' | 'purple' | 'blue' | 'green' | 'orange' | 'pink'
    | 'stormy-morning' | 'blue-eclipse' | 'lush-forest' | 'green-juice'
    | 'wisteria-bloom' | 'lavender-fields' | 'golden-hour' | 'pistachio-dream'
    | 'electropop' | 'neon-noir' | 'yacht-club' | 'honeycomb';

// Simple options only recolor the accent; the rest of the palette stays the
// standard light/dark look.
export const ACCENT_COLOR_OPTIONS: { value: ColorThemeName; label: string; swatch: string }[] = [
    { value: 'default', label: 'Default', swatch: 'linear-gradient(135deg, #3b82f6 50%, #9d6bff 50%)' },
    { value: 'purple', label: 'Purple', swatch: '#9d6bff' },
    { value: 'blue', label: 'Blue', swatch: '#3b82f6' },
    { value: 'green', label: 'Green', swatch: '#10b981' },
    { value: 'orange', label: 'Orange', swatch: '#f97316' },
    { value: 'pink', label: 'Pink', swatch: '#ec4899' },
];

// Full themes recolor the entire surface/text/accent palette (both light and
// dark variants), inspired by Figma's color combination library.
export const FULL_THEME_OPTIONS: { value: ColorThemeName; label: string; swatch: string }[] = [
    { value: 'stormy-morning', label: 'Stormy Morning', swatch: 'linear-gradient(135deg, #384959, #88bdf2)' },
    { value: 'blue-eclipse', label: 'Blue Eclipse', swatch: 'linear-gradient(135deg, #0f0e47, #8686ac)' },
    { value: 'lush-forest', label: 'Lush Forest', swatch: 'linear-gradient(135deg, #253d2c, #68ba7f)' },
    { value: 'green-juice', label: 'Green Juice', swatch: 'linear-gradient(135deg, #293325, #4cbb17)' },
    { value: 'wisteria-bloom', label: 'Wisteria Bloom', swatch: 'linear-gradient(135deg, #9400d3, #ed80e9)' },
    { value: 'lavender-fields', label: 'Lavender Fields', swatch: 'linear-gradient(135deg, #cf6dfc, #c1bfff)' },
    { value: 'golden-hour', label: 'Golden Hour', swatch: 'linear-gradient(135deg, #cc5500, #ffbf00)' },
    { value: 'pistachio-dream', label: 'Pistachio Dream', swatch: 'linear-gradient(135deg, #42d674, #e3f0a3)' },
    { value: 'electropop', label: 'Electropop', swatch: 'linear-gradient(135deg, #5200ff, #f900ff)' },
    { value: 'neon-noir', label: 'Neon Noir', swatch: 'linear-gradient(135deg, #000000, #bf00ff)' },
    { value: 'yacht-club', label: 'Yacht Club', swatch: 'linear-gradient(135deg, #733e24, #245f73)' },
    { value: 'honeycomb', label: 'Honeycomb', swatch: 'linear-gradient(135deg, #895129, #ffc107)' },
];

export const COLOR_THEME_OPTIONS = [...ACCENT_COLOR_OPTIONS, ...FULL_THEME_OPTIONS];

const COLOR_THEME_KEY = 'pagerops_color_theme';
const DEFAULT_COLOR_THEME: ColorThemeName = 'default';

function isColorThemeName(value: unknown): value is ColorThemeName {
    return COLOR_THEME_OPTIONS.some((option) => option.value === value);
}

function loadColorTheme(): ColorThemeName {
    try {
        const stored = localStorage.getItem(COLOR_THEME_KEY);
        if (isColorThemeName(stored)) {
            return stored;
        }
    } catch (err) {
        console.error('Failed to load color theme preference:', err);
    }
    return DEFAULT_COLOR_THEME;
}

// Shared accent color theme store, independent of light/dark mode.
export const colorTheme = writable<ColorThemeName>(loadColorTheme());

// Reflect the active color theme on <html data-color-theme="..."> so CSS
// variables in style.css can switch, and persist the choice across restarts.
colorTheme.subscribe((value) => {
    if (typeof document !== 'undefined') {
        document.documentElement.setAttribute('data-color-theme', value);
    }
    try {
        localStorage.setItem(COLOR_THEME_KEY, value);
    } catch (err) {
        console.error('Failed to save color theme preference:', err);
    }
});

export function setColorTheme(value: ColorThemeName): void {
    colorTheme.set(value);
}
