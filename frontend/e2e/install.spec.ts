import { expect, test, type Page } from "@playwright/test";

const offerNativeInstall = (page: Page) =>
  page.evaluate(() => {
    const event = Object.assign(new Event("beforeinstallprompt", { cancelable: true }), {
      prompt: () => {
        const { dataset } = document.documentElement;
        dataset.installPrompts = String(Number(dataset.installPrompts ?? "0") + 1);
        return Promise.resolve({ outcome: "dismissed" });
      },
    });
    return window.dispatchEvent(event);
  });

const installBanner = (page: Page) => page.getByRole("complementary", { name: "Install Handlat" });
const profileInstall = (page: Page) => page.getByRole("region", { name: "Install Handlat" });
const openProfile = async (page: Page) => {
  await page.route("**/api/auth/me", (route) =>
    route.fulfill({ json: { id: "install-user", name: "Install User" } }),
  );
  await page.goto("/user");
  await expect(page.getByRole("heading", { name: "Profile" })).toBeVisible();
};

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    window.addEventListener("beforeinstallprompt", (event) => {
      if (event.isTrusted) {
        event.preventDefault();
        event.stopImmediatePropagation();
      }
    });
  });
});

test("offers installation from an event received before the route loads", async ({ page }) => {
  await page.route("**/api/auth/me", async (route) => {
    await offerNativeInstall(page);
    await route.fulfill({ status: 401 });
  });
  await page.goto("/login");

  const banner = installBanner(page);
  await expect(banner).toBeVisible();
  await expect(page.locator("html")).not.toHaveAttribute("data-install-prompts");
  await banner.getByRole("button", { name: "Install", exact: true }).click();
  await expect(page.locator("html")).toHaveAttribute("data-install-prompts", "1");
  await expect(banner).toHaveCount(0);
});

test("waits for browser eligibility and remembers dismissal in the tab", async ({ page }) => {
  await openProfile(page);

  const banner = installBanner(page);
  const section = profileInstall(page);
  await expect(banner).toHaveCount(0);
  await expect(section).toHaveCount(0);
  expect(await offerNativeInstall(page)).toBe(false);
  await banner.getByRole("button", { name: "Dismiss install banner" }).click();
  await expect(banner).toHaveCount(0);
  await expect(section.getByRole("button", { name: "Install", exact: true })).toBeVisible();

  await page.reload();
  await expect(page.getByRole("heading", { name: "Profile" })).toBeVisible();
  expect(await offerNativeInstall(page)).toBe(false);
  await expect(banner).toHaveCount(0);
  await section.getByRole("button", { name: "Install", exact: true }).click();
  await expect(page.locator("html")).toHaveAttribute("data-install-prompts", "1");
  await expect(section).toHaveCount(0);
});

test("can install from Profile after dismissing the banner without reloading", async ({ page }) => {
  await openProfile(page);
  expect(await offerNativeInstall(page)).toBe(false);

  const banner = installBanner(page);
  await banner.getByRole("button", { name: "Dismiss install banner" }).click();
  await expect(banner).toHaveCount(0);
  await profileInstall(page).getByRole("button", { name: "Install", exact: true }).click();
  await expect(page.locator("html")).toHaveAttribute("data-install-prompts", "1");
});

test("hides installation controls when the browser reports installation", async ({ page }) => {
  await openProfile(page);
  expect(await offerNativeInstall(page)).toBe(false);
  const banner = installBanner(page);
  const section = profileInstall(page);
  await expect(banner).toBeVisible();
  await expect(section).toBeVisible();

  await page.evaluate(() => window.dispatchEvent(new Event("appinstalled")));
  await expect(section).toHaveCount(0);
  await expect(banner).toHaveCount(0);
});

test("does not show installation controls in standalone display mode", async ({ page }) => {
  await page.addInitScript(() => {
    const matchMedia = window.matchMedia.bind(window);
    window.matchMedia = (query) => {
      const result = matchMedia(query);
      if (query === "(display-mode: standalone)") {
        Object.defineProperty(result, "matches", { value: true });
      }
      return result;
    };
  });
  await openProfile(page);

  expect(await offerNativeInstall(page)).toBe(true);
  await expect(installBanner(page)).toHaveCount(0);
  await expect(profileInstall(page)).toHaveCount(0);
});

test.describe("iPhone installation", () => {
  test.use({
    userAgent:
      "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1",
    viewport: { width: 375, height: 667 },
  });

  test("offers home-screen steps in a browser tab", async ({ page }) => {
    await openProfile(page);

    const banner = installBanner(page);
    await expect(banner).toBeVisible();
    const instructions = page.locator('wa-dialog[label="Install Handlat"]');
    const dialog = instructions.getByRole("dialog", { name: "Install Handlat" });
    await expect(dialog).toBeHidden();
    await banner.getByRole("button", { name: "How to install" }).click();
    await expect(dialog).toBeVisible();
    await expect(instructions.getByText(/Add to Home Screen/)).toBeVisible();
    await instructions.getByRole("button", { name: "Done" }).click();
    await expect(dialog).toBeHidden();
    await banner.getByRole("button", { name: "Dismiss install banner" }).click();
    await expect(banner).toHaveCount(0);
    await expect(profileInstall(page).getByText(/Add to Home Screen/)).toBeVisible();
  });

  test("hides installation controls when launched from the home screen", async ({ page }) => {
    await page.addInitScript(() => {
      Object.defineProperty(navigator, "standalone", { value: true });
    });
    await openProfile(page);
    await expect(installBanner(page)).toHaveCount(0);
    await expect(profileInstall(page)).toHaveCount(0);
  });
});
