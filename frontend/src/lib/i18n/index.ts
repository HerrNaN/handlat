import { derived, writable } from "svelte/store";

import { en } from "./en";
import { sv } from "./sv";

const catalogs = { en, sv };
export type Language = keyof typeof catalogs;
type LanguagePreference = Language | "browser";

const storageKey = "handlat-language";

const parsePreference = (value: unknown): LanguagePreference =>
  value === "en" || value === "sv" ? value : "browser";

const readPreference = (): LanguagePreference => {
  try {
    return parsePreference(localStorage.getItem(storageKey));
  } catch {
    return "browser";
  }
};

const preferredBrowserLanguage = (): Language => {
  for (const tag of navigator.languages) {
    const primary = tag.toLowerCase().split("-")[0];
    if (primary === "en" || primary === "sv") {
      return primary;
    }
  }
  return "en";
};

const preference = writable(readPreference());
const browserLanguage = writable<Language>("en");
export const languagePreference = { subscribe: preference.subscribe };
export const language = derived([preference, browserLanguage], ([$preference, $browserLanguage]) =>
  $preference === "browser" ? $browserLanguage : $preference,
);
export const messages = derived(language, ($language) => catalogs[$language]);

export const setLanguage = (value: string) => {
  const selected = parsePreference(value);
  preference.set(selected);
  try {
    localStorage.setItem(storageKey, selected);
  } catch {
    // The selection still applies for this visit when storage is unavailable.
  }
};

export const initializeLanguage = () => {
  const updateBrowserLanguage = () => browserLanguage.set(preferredBrowserLanguage());
  updateBrowserLanguage();
  language.subscribe((value) => {
    document.documentElement.lang = value;
  });
  window.addEventListener("languagechange", updateBrowserLanguage);
  window.addEventListener("storage", (event) => {
    if (event.key === storageKey || event.key === null) {
      preference.set(readPreference());
    }
  });
};
