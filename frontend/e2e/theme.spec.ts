import { expect, test, type Page } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.route("**/api/auth/me", (route) =>
    route.fulfill({ json: { id: "theme-user", name: "Theme User" } }),
  );
});

const expectScheme = async (page: Page, scheme: "light" | "dark") => {
  await expect(page.locator("html")).toHaveCSS("color-scheme", scheme);
  await expect(page.locator("html")).toHaveClass(`wa-${scheme}`);
  const background = await page
    .locator("html")
    .evaluate((root) => getComputedStyle(root).backgroundColor);
  const themeColors = page.locator('meta[name="theme-color"]');
  await expect(themeColors).toHaveCount(2);
  for (const meta of await themeColors.all()) {
    await expect(meta).toHaveAttribute("content", background);
  }
};

const selectTheme = async (page: Page, theme: "Light" | "Dark" | "System") => {
  await page.getByRole("combobox", { name: "Theme" }).click();
  await page.getByRole("option", { name: theme, exact: true }).click();
};

test("defaults to System and follows device appearance changes", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/user");
  await expect(page.locator("wa-select")).toHaveJSProperty("value", "system");
  await expectScheme(page, "light");
  const lightBackground = await page
    .locator("html")
    .evaluate((root) => getComputedStyle(root).backgroundColor);

  await page.emulateMedia({ colorScheme: "dark" });
  await expectScheme(page, "dark");
  await expect(page.locator("html")).not.toHaveCSS("background-color", lightBackground);

  await page.emulateMedia({ colorScheme: "light" });
  await expectScheme(page, "light");
});

test("remembers explicit choices and can return to System", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/user");
  await selectTheme(page, "Dark");
  await expectScheme(page, "dark");
  await page.reload();
  await expect(page.locator("wa-select")).toHaveJSProperty("value", "dark");
  await expectScheme(page, "dark");

  await page.emulateMedia({ colorScheme: "dark" });
  await selectTheme(page, "Light");
  await expectScheme(page, "light");
  await page.reload();
  await expect(page.locator("wa-select")).toHaveJSProperty("value", "light");
  await expectScheme(page, "light");

  await selectTheme(page, "System");
  await expectScheme(page, "dark");
  await page.reload();
  await expect(page.locator("wa-select")).toHaveJSProperty("value", "system");
  await page.emulateMedia({ colorScheme: "light" });
  await expectScheme(page, "light");
});

test("falls back to System for an invalid stored preference", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("handlat-theme", "invalid"));
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/user");
  await expect(page.locator("wa-select")).toHaveJSProperty("value", "system");
  await expectScheme(page, "dark");
});

test("allows theme changes when browser storage is unavailable", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(window, "localStorage", {
      get: () => {
        throw new DOMException("Storage disabled", "SecurityError");
      },
    });
  });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/user");
  await expectScheme(page, "dark");
  await selectTheme(page, "Light");
  await expectScheme(page, "light");
});

test("updates the empty-list illustration when the theme changes", async ({ page }) => {
  await page.route("**/api/lists/theme-list/events", (route) =>
    route.fulfill({
      contentType: "text/event-stream",
      body: `data: ${JSON.stringify({ id: "theme-list", name: "Shopping List", items: [], memberCount: 1 })}\n\n`,
    }),
  );
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/lists/theme-list");
  const illustration = page.locator("svg.illustration");
  await expect(illustration).toBeVisible();
  const circle = illustration.locator("circle");
  const lightBackground = await circle.evaluate((element) => getComputedStyle(element).fill);
  const stops = illustration.locator("stop");
  await expect(stops).toHaveCount(4);
  const lightColors = await stops.evaluateAll((elements) =>
    elements.map((element) => getComputedStyle(element).stopColor),
  );

  await page.emulateMedia({ colorScheme: "dark" });
  await expectScheme(page, "dark");
  // Only the background circle changes; the artwork retains its light-theme colors.
  await expect(circle).not.toHaveCSS("fill", lightBackground);
  for (let index = 0; index < lightColors.length; index++) {
    await expect(stops.nth(index)).toHaveCSS("stop-color", lightColors[index]!);
  }

  await page.emulateMedia({ colorScheme: "light" });
  await expectScheme(page, "light");
  await expect(circle).toHaveCSS("fill", lightBackground);
  for (let index = 0; index < lightColors.length; index++) {
    await expect(stops.nth(index)).toHaveCSS("stop-color", lightColors[index]!);
  }
});
