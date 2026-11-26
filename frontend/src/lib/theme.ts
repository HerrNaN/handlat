import { get, writable } from "svelte/store";

export type ThemePreference = "light" | "dark" | "system";

const storageKey = "handlat-theme";

const parsePreference = (value: unknown): ThemePreference =>
  value === "light" || value === "dark" ? value : "system";

const readPreference = (): ThemePreference => {
  try {
    return parsePreference(localStorage.getItem(storageKey));
  } catch {
    return "system";
  }
};

const current = writable<ThemePreference>(readPreference());
export const theme = { subscribe: current.subscribe };

export const setTheme = (value: string) => {
  const preference = parsePreference(value);
  current.set(preference);
  try {
    localStorage.setItem(storageKey, preference);
  } catch {
    // The choice still applies for this visit when storage is unavailable.
  }
};

export const initializeTheme = () => {
  const system = matchMedia("(prefers-color-scheme: dark)");
  const themeColor = document.querySelector<HTMLMetaElement>(
    'meta[name="theme-color"]:not([media])',
  );
  const initialDarkThemeColor = document.querySelector('meta[name="theme-color"][media]');
  const applyTheme = () => {
    const preference = get(current);
    const dark = preference === "dark" || (preference === "system" && system.matches);
    const root = document.documentElement;
    root.classList.toggle("wa-dark", dark);
    root.classList.toggle("wa-light", !dark);
    root.style.colorScheme = dark ? "dark" : "light";
    if (themeColor) {
      themeColor.content = getComputedStyle(root).backgroundColor;
    }
  };
  const onStorage = (event: StorageEvent) => {
    if (event.key === storageKey || event.key === null) {
      current.set(readPreference());
    }
  };

  current.subscribe(applyTheme);
  // The media-qualified color is only needed before the app resolves its theme.
  initialDarkThemeColor?.remove();
  system.addEventListener("change", applyTheme);
  window.addEventListener("storage", onStorage);
};
