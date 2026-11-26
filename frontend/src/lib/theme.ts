import { get, writable } from "svelte/store";

type ColorMode = "light" | "dark" | "system";
type Theme = "warm" | "soft-white" | "neutral";

// Keep the existing key for saved Light, Dark, and System preferences.
const colorModeStorageKey = "handlat-theme";
const paletteStorageKey = "handlat-palette";

const parseColorMode = (value: string | null): ColorMode =>
  value === "light" || value === "dark" ? value : "system";

const parseTheme = (value: string | null): Theme =>
  value === "soft-white" || value === "neutral" ? value : "warm";

const readPreference = (key: string): string | null => {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
};

const readAppearance = () => ({
  colorMode: parseColorMode(readPreference(colorModeStorageKey)),
  theme: parseTheme(readPreference(paletteStorageKey)),
});

const current = writable(readAppearance());
export const appearance = { subscribe: current.subscribe };

const savePreference = (key: string, value: string) => {
  try {
    localStorage.setItem(key, value);
  } catch {
    // The choice still applies for this visit when storage is unavailable.
  }
};

export const setColorMode = (value: string) => {
  const colorMode = parseColorMode(value);
  current.update((settings) => ({ ...settings, colorMode }));
  savePreference(colorModeStorageKey, colorMode);
};

export const setTheme = (value: string) => {
  const theme = parseTheme(value);
  current.update((settings) => ({ ...settings, theme }));
  savePreference(paletteStorageKey, theme);
};

export const initializeTheme = () => {
  const system = matchMedia("(prefers-color-scheme: dark)");
  const themeColor = document.querySelector<HTMLMetaElement>(
    'meta[name="theme-color"]:not([media])',
  );
  const initialDarkThemeColor = document.querySelector('meta[name="theme-color"][media]');
  const applyTheme = () => {
    const { colorMode, theme } = get(current);
    const dark = colorMode === "dark" || (colorMode === "system" && system.matches);
    const root = document.documentElement;
    root.dataset.theme = theme;
    root.classList.toggle("wa-dark", dark);
    root.classList.toggle("wa-light", !dark);
    root.style.colorScheme = dark ? "dark" : "light";
    if (themeColor) {
      const headerBackground = getComputedStyle(root)
        .getPropertyValue("--handlat-header-background")
        .trim();
      if (headerBackground) {
        themeColor.content = headerBackground;
      }
    }
  };
  const onStorage = (event: StorageEvent) => {
    if (
      event.key === colorModeStorageKey ||
      event.key === paletteStorageKey ||
      event.key === null
    ) {
      current.set(readAppearance());
    }
  };

  current.subscribe(applyTheme);
  // The media-qualified color is only needed before the app resolves its theme.
  initialDarkThemeColor?.remove();
  system.addEventListener("change", applyTheme);
  window.addEventListener("storage", onStorage);
};
