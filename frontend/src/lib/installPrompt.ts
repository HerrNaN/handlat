import { writable } from "svelte/store";

type InstallPromptEvent = Event & { prompt: () => Promise<unknown> };

const storageKey = "handlat-install-dismissed";
const offer = writable<"native" | "ios" | null>(null);
export const installOffer = { subscribe: offer.subscribe };

let deferredPrompt: InstallPromptEvent | undefined;
let dismissed = false;

export const dismissInstallOffer = () => {
  deferredPrompt = undefined;
  dismissed = true;
  offer.set(null);
  try {
    sessionStorage.setItem(storageKey, "true");
  } catch {
    // Dismissal still applies until the page is reloaded.
  }
};

export const requestInstall = async () => {
  const prompt = deferredPrompt;
  if (!prompt) {
    return;
  }

  // Each event can open the native prompt once, directly from the click.
  dismissInstallOffer();
  try {
    await prompt.prompt();
  } catch {
    // The browser can withdraw the single-use prompt before it opens.
  }
};

export const initializeInstallPrompt = () => {
  try {
    dismissed = sessionStorage.getItem(storageKey) === "true";
  } catch {
    // Installation is still available when storage is blocked.
  }

  const standalone = window.matchMedia("(display-mode: standalone)");
  const appleNavigator: Navigator & { standalone?: boolean } = navigator;
  const isBrowserTab = () => !standalone.matches && !appleNavigator.standalone;
  // iPad Safari can identify itself as a Mac. In-app web views usually omit Safari.
  const isiOSBrowser =
    window.isSecureContext &&
    /Safari\//.test(navigator.userAgent) &&
    (/iPhone|iPad|iPod/.test(navigator.userAgent) ||
      (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1));

  const updateOffer = () => {
    if (dismissed || !isBrowserTab()) {
      offer.set(null);
    } else if (deferredPrompt) {
      offer.set("native");
    } else {
      offer.set(isiOSBrowser ? "ios" : null);
    }
  };

  window.addEventListener("beforeinstallprompt", (event) => {
    if (dismissed || !isBrowserTab()) {
      return;
    }
    event.preventDefault();
    deferredPrompt = event as InstallPromptEvent;
    updateOffer();
  });
  window.addEventListener("appinstalled", dismissInstallOffer);
  standalone.addEventListener("change", updateOffer);
  updateOffer();
};
