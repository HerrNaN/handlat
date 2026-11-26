import { derived, get, writable } from "svelte/store";

type InstallPromptEvent = Event & { prompt: () => Promise<unknown> };

const storageKey = "handlat-install-dismissed";
const browserContext = writable<"browser" | "ios" | null>(null);
const nativePrompt = writable<InstallPromptEvent | null>(null);
const bannerDismissed = writable(false);

export const installOffer = derived([browserContext, nativePrompt], ([$context, $prompt]) => {
  if (!$context) {
    return null;
  }
  if ($prompt) {
    return "native";
  }
  return $context === "ios" ? "ios" : null;
});

export const installBannerOffer = derived([installOffer, bannerDismissed], ([$offer, $dismissed]) =>
  $dismissed ? null : $offer,
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
  if (!prompt || !get(browserContext)) {
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
  // iPad Safari can identify itself as a Mac. In-app web views usually omit Safari.
  const isiOSBrowser =
    window.isSecureContext &&
    /Safari\//.test(navigator.userAgent) &&
    (/iPhone|iPad|iPod/.test(navigator.userAgent) ||
      (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1));

  let installed = false;
  const updateBrowserContext = () => {
    if (installed || standalone.matches || appleNavigator.standalone) {
      browserContext.set(null);
    } else {
      browserContext.set(isiOSBrowser ? "ios" : "browser");
    }
  };

  window.addEventListener("beforeinstallprompt", (event) => {
    if (!get(browserContext)) {
      return;
    }
    event.preventDefault();
    nativePrompt.set(event as InstallPromptEvent);
  });
  window.addEventListener("appinstalled", () => {
    installed = true;
    updateBrowserContext();
    nativePrompt.set(null);
    dismissInstallBanner();
  });
  standalone.addEventListener("change", updateBrowserContext);
  updateBrowserContext();
};
