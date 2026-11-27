import { derived, get, writable } from "svelte/store";

type InstallPromptEvent = Event & { prompt: () => Promise<unknown> };

const storageKey = "handlat-install-dismissed";
const nativePrompt = writable<InstallPromptEvent | null>(null);
const bannerDismissed = writable(false);

export const canInstall = derived(nativePrompt, ($prompt) => $prompt !== null);
export const showInstallBanner = derived(
  [canInstall, bannerDismissed],
  ([$canInstall, $dismissed]) => $canInstall && !$dismissed,
);

export const dismissInstallBanner = () => {
  bannerDismissed.set(true);
  try {
    sessionStorage.setItem(storageKey, "true");
  } catch {
    // Dismissal still applies until the page is reloaded.
  }
};

export const requestInstall = async () => {
  const prompt = get(nativePrompt);
  if (!prompt) {
    return;
  }

  // Each event can open the native prompt once, directly from the click.
  nativePrompt.set(null);
  dismissInstallBanner();
  try {
    await prompt.prompt();
  } catch {
    // The browser can withdraw the single-use prompt before it opens.
  }
};

export const initializeInstallPrompt = () => {
  try {
    bannerDismissed.set(sessionStorage.getItem(storageKey) === "true");
  } catch {
    // Installation is still available when storage is blocked.
  }

  const standalone = window.matchMedia("(display-mode: standalone)");
  const appleNavigator: Navigator & { standalone?: boolean } = navigator;
  const isStandalone = () => standalone.matches || appleNavigator.standalone === true;
  let installed = false;

  window.addEventListener("beforeinstallprompt", (event) => {
    if (installed || isStandalone() || !("prompt" in event) || typeof event.prompt !== "function") {
      return;
    }
    event.preventDefault();
    nativePrompt.set(event as InstallPromptEvent);
  });
  window.addEventListener("appinstalled", () => {
    installed = true;
    nativePrompt.set(null);
    dismissInstallBanner();
  });
  standalone.addEventListener("change", () => {
    if (isStandalone()) {
      nativePrompt.set(null);
    }
  });
};
