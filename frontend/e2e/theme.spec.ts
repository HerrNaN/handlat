import { expect, test, type Page } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.route("**/api/auth/me", (route) =>
    route.fulfill({ json: { id: "theme-user", name: "Theme User" } }),
  );
});

const expectScheme = async (page: Page, scheme: "light" | "dark") => {
  const root = page.locator("html");
  const background = scheme === "light" ? "rgb(247, 243, 237)" : "rgb(32, 28, 25)";
  const headerBackground = scheme === "light" ? "rgb(239, 232, 222)" : "rgb(32, 28, 25)";
  const themeColor = scheme === "light" ? "#efe8de" : "#201c19";
  await expect(root).toHaveCSS("color-scheme", scheme);
  await expect(root).toHaveClass(`wa-${scheme}`);
  await expect(root).toHaveCSS("background-color", headerBackground);
  await expect(page.locator("#app")).toHaveCSS("background-color", background);
  await expect(page.locator("body")).toHaveCSS("background-color", headerBackground);
  await expect(page.locator("article > header")).toHaveCSS("background-color", headerBackground);
  await expect(page.locator('meta[name="theme-color"]')).toHaveCount(1);
  await expect(page.locator('meta[name="theme-color"]:not([media])')).toHaveAttribute(
    "content",
    themeColor,
  );
};

const selectTheme = async (page: Page, theme: "Light" | "Dark" | "System") => {
  const radio = page.getByRole("radio", { name: theme, exact: true });
  await radio.click();
  await expect(radio).toBeChecked();
};

test("provides the light palette’s page and header colors for the installed app", async ({
  request,
}) => {
  const manifest = await request.get("/manifest.webmanifest");
  await expect(manifest).toBeOK();
  expect(await manifest.json()).toMatchObject({
    display: "standalone",
    background_color: "#f7f3ed",
    theme_color: "#efe8de",
  });
});

test("defaults to System and follows device appearance changes", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/user");
  await expect(page.getByRole("radio", { name: "System", exact: true })).toBeChecked();
  await expectScheme(page, "light");

  await page.emulateMedia({ colorScheme: "dark" });
  await expectScheme(page, "dark");

  await page.emulateMedia({ colorScheme: "light" });
  await expectScheme(page, "light");
});

test("remembers explicit choices and can return to System", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/user");
  await selectTheme(page, "Dark");
  await expectScheme(page, "dark");
  await page.reload();
  await expect(page.getByRole("radio", { name: "Dark", exact: true })).toBeChecked();
  await expectScheme(page, "dark");

  await page.emulateMedia({ colorScheme: "dark" });
  await selectTheme(page, "Light");
  await expectScheme(page, "light");
  await page.reload();
  await expect(page.getByRole("radio", { name: "Light", exact: true })).toBeChecked();
  await expectScheme(page, "light");

  await selectTheme(page, "System");
  await expectScheme(page, "dark");
  await page.reload();
  await expect(page.getByRole("radio", { name: "System", exact: true })).toBeChecked();
  await page.emulateMedia({ colorScheme: "light" });
  await expectScheme(page, "light");
});

test("falls back to System for an invalid stored preference", async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem("handlat-theme", "invalid"));
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/user");
  await expect(page.getByRole("radio", { name: "System", exact: true })).toBeChecked();
  await expectScheme(page, "dark");
});

test("changes theme using the segmented control's arrow keys", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/user");
  await expect(page.getByRole("radiogroup", { name: "Theme" })).toBeVisible();
  const system = page.getByRole("radio", { name: "System", exact: true });
  await expect(system).toBeChecked();
  await system.press("ArrowLeft");
  const dark = page.getByRole("radio", { name: "Dark", exact: true });
  await expect(dark).toBeChecked();
  await expect(dark).toBeFocused();
  await expectScheme(page, "dark");
  await dark.press("ArrowLeft");
  await expect(page.getByRole("radio", { name: "Light", exact: true })).toBeChecked();
  await expectScheme(page, "light");
});

const channels = (color: string) =>
  color
    .slice(color.indexOf("(") + 1, -1)
    .split(",")
    .map(Number);

const luminance = (rgb: number[]) => {
  const [red, green, blue] = rgb
    .map((channel) => channel / 255)
    .map((channel) => (channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4));
  return 0.2126 * red! + 0.7152 * green! + 0.0722 * blue!;
};

const contrast = (foreground: string, background: string, backdrop = "rgb(255, 255, 255)") => {
  const fill = channels(background);
  const surface = channels(backdrop);
  const opacity = fill[3] ?? 1;
  const visibleFill = fill
    .slice(0, 3)
    .map((channel, index) => channel * opacity + surface[index]! * (1 - opacity));
  const a = luminance(channels(foreground));
  const b = luminance(visibleFill);
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
};

for (const scheme of ["light", "dark"] as const) {
  test(`keeps selection and keyboard focus distinct in ${scheme} mode`, async ({ page }) => {
    await page.emulateMedia({ colorScheme: scheme });
    await page.goto("/user");
    const pageBackground = await page
      .locator("html")
      .evaluate((root) => getComputedStyle(root).backgroundColor);
    await page.getByRole("button", { name: "Log out" }).focus();

    for (const [name, current, other] of [
      ["Theme", "System", "Light"],
      ["Language", "Auto", "English"],
    ] as const) {
      await expect(page.getByRole("radiogroup", { name, exact: true })).toBeVisible();
      const selected = page.getByRole("radio", { name: current, exact: true, checked: true });
      const unselected = page.getByRole("radio", { name: other, exact: true, checked: false });
      await page.keyboard.press("Tab");
      await expect(selected).toBeFocused();
      await expect(selected).toHaveCSS("outline-style", "solid");
      await expect(selected).toHaveCSS("outline-width", "3px");
      await expect(selected).toHaveCSS("outline-offset", "2px");

      const active = await selected.evaluate((radio) => {
        const style = getComputedStyle(radio);
        const marker = getComputedStyle(radio, "::after");
        return {
          text: style.color,
          fill: style.backgroundColor,
          focus: style.outlineColor,
          marker: marker.borderBlockEndColor,
          markerWidth: marker.borderBlockEndWidth,
        };
      });
      const idle = await unselected.evaluate((radio) => {
        const style = getComputedStyle(radio);
        return {
          text: style.color,
          fill: style.backgroundColor,
          border: style.borderBlockEndColor,
        };
      });
      expect(contrast(active.text, active.fill)).toBeGreaterThanOrEqual(4.5);
      expect(contrast(idle.text, idle.fill)).toBeGreaterThanOrEqual(4.5);
      expect(contrast(active.fill, idle.fill)).toBeGreaterThanOrEqual(3);
      expect(contrast(idle.border, idle.fill)).toBeGreaterThanOrEqual(3);
      expect(contrast(idle.border, pageBackground)).toBeGreaterThanOrEqual(3);
      expect(contrast(active.focus, pageBackground)).toBeGreaterThanOrEqual(3);
      expect(contrast(active.focus, idle.fill)).toBeGreaterThanOrEqual(3);
      expect(active.markerWidth).toBe("2px");
      expect(contrast(active.marker, active.fill)).toBeGreaterThanOrEqual(3);
    }
  });
}

test("keeps the selected segment readable in forced colors", async ({ page }) => {
  await page.emulateMedia({ forcedColors: "active" });
  await page.goto("/user");
  const selected = page.getByRole("radio", { name: "System", exact: true, checked: true });
  const unselected = page.getByRole("radio", { name: "Light", exact: true, checked: false });
  const background = await unselected.evaluate((radio) => getComputedStyle(radio).backgroundColor);
  await expect(selected).not.toHaveCSS("background-color", background);
  await expect(selected.getByText("System", { exact: true })).toHaveCSS(
    "forced-color-adjust",
    "none",
  );
  const active = await selected.evaluate((radio) => {
    const style = getComputedStyle(radio);
    const marker = getComputedStyle(radio, "::after");
    return {
      fill: style.backgroundColor,
      text: style.color,
      markerWidth: marker.borderBlockEndWidth,
      backdrop: getComputedStyle(radio.closest("article")!).backgroundColor,
    };
  });
  expect(contrast(active.text, active.fill, active.backdrop)).toBeGreaterThanOrEqual(4.5);
  expect(active.markerWidth).toBe("2px");
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

test("applies the palette to shopping-list rows and actions", async ({ page }) => {
  const list = {
    id: "theme-list",
    name: "Shopping List",
    items: [
      {
        id: "milk",
        text: "Milk",
        createdAt: "2026-01-01T00:00:00Z",
        checkedAt: "2026-01-01T00:01:00Z",
      },
    ],
    memberCount: 1,
  };
  await page.route("**/api/lists/theme-list/events", (route) =>
    route.fulfill({ contentType: "text/event-stream", body: `data: ${JSON.stringify(list)}\n\n` }),
  );
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/lists/theme-list");

  const row = page.getByRole("button", { name: "Edit Milk", exact: true }).locator("..");
  const add = page.getByRole("button", { name: "Add item", exact: true });
  const checkbox = row.locator("wa-checkbox");
  const checkmark = checkbox.locator('[part~="control"]');
  const swipeDelete = page.locator('.underlay:has(> wa-icon[name="trash"])');
  await expect(page.getByRole("checkbox", { name: "Milk" })).toBeChecked();
  await expect(row).toHaveCSS("background-color", "rgb(255, 252, 248)");
  await expect(add).toHaveCSS("background-color", "rgb(168, 62, 50)");
  await expect(add).toHaveCSS("color", "rgb(255, 255, 255)");
  await expect(checkbox).toHaveCSS("background-color", "rgb(237, 240, 228)");
  await expect(checkmark).toHaveCSS("background-color", "rgb(72, 97, 58)");
  await expect(checkmark).toHaveCSS("color", "rgb(255, 252, 248)");
  await expect(swipeDelete).toHaveCSS("background-color", "rgb(147, 52, 71)");

  await page.emulateMedia({ colorScheme: "dark" });
  await expect(row).toHaveCSS("background-color", "rgb(43, 37, 33)");
  await expect(add).toHaveCSS("background-color", "rgb(229, 141, 120)");
  await expect(add).toHaveCSS("color", "rgb(43, 24, 18)");
  await expect(checkbox).toHaveCSS("background-color", "rgb(41, 49, 35)");
  await expect(checkmark).toHaveCSS("background-color", "rgb(169, 194, 141)");
  await expect(checkmark).toHaveCSS("color", "rgb(43, 37, 33)");
  await expect(swipeDelete).toHaveCSS("background-color", "rgb(240, 162, 174)");
});

test("uses the warning palette for the recipe opt-in", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/user");

  const enable = page.getByRole("button", { name: "Enable", exact: true });
  await expect(enable).toHaveCSS("background-color", "rgb(246, 229, 200)");
  await expect(enable).toHaveCSS("color", "rgb(128, 84, 20)");

  await page.emulateMedia({ colorScheme: "dark" });
  await expect(enable).toHaveCSS("background-color", "rgb(69, 50, 31)");
  await expect(enable).toHaveCSS("color", "rgb(237, 188, 115)");
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
  const listBackground = await page
    .locator(".list-area")
    .evaluate((element) => getComputedStyle(element).backgroundColor);
  await expect(circle).not.toHaveCSS("fill", listBackground);
  const stops = illustration.locator("stop");
  await expect(stops).toHaveCount(4);
  const lightColors = await stops.evaluateAll((elements) =>
    elements.map((element) => getComputedStyle(element).stopColor),
  );

  await page.emulateMedia({ colorScheme: "dark" });
  await expectScheme(page, "dark");
  // The basket and carton keep the same gradient colors in both themes.
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
