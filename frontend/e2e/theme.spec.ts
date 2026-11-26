import { expect, test, type Page } from "@playwright/test";

test.beforeEach(async ({ context }) => {
  await context.route("**/api/auth/me", (route) =>
    route.fulfill({ json: { id: "theme-user", name: "Theme User" } }),
  );
});

const themeColors = {
  warm: {
    light: ["rgb(247, 243, 237)", "rgb(239, 232, 222)", "#efe8de"],
    dark: ["rgb(32, 28, 25)", "rgb(32, 28, 25)", "#201c19"],
  },
  "soft-white": {
    light: ["rgb(252, 251, 249)", "rgb(241, 235, 226)", "#f1ebe2"],
    dark: ["rgb(37, 34, 31)", "rgb(37, 34, 31)", "#25221f"],
  },
  neutral: {
    light: ["rgb(252, 251, 249)", "rgb(232, 234, 236)", "#e8eaec"],
    dark: ["rgb(37, 34, 31)", "rgb(38, 40, 43)", "#26282b"],
  },
} as const;

const expectScheme = async (
  page: Page,
  scheme: "light" | "dark",
  theme: keyof typeof themeColors = "warm",
) => {
  const root = page.locator("html");
  const [background, headerBackground, themeColor] = themeColors[theme][scheme];
  await expect(root).toHaveCSS("color-scheme", scheme);
  await expect(root).toHaveClass(`wa-${scheme}`);
  await expect(root).toHaveAttribute("data-theme", theme);
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

const selectRadio = async (page: Page, name: string) => {
  const radio = page.getByRole("radio", { name, exact: true });
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

test("remembers explicit color modes and can return to System", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/user");
  await selectRadio(page, "Dark");
  await expectScheme(page, "dark");
  await page.reload();
  await expect(page.getByRole("radio", { name: "Dark", exact: true })).toBeChecked();
  await expectScheme(page, "dark");

  await page.emulateMedia({ colorScheme: "dark" });
  await selectRadio(page, "Light");
  await expectScheme(page, "light");
  await page.reload();
  await expect(page.getByRole("radio", { name: "Light", exact: true })).toBeChecked();
  await expectScheme(page, "light");

  await selectRadio(page, "System");
  await expectScheme(page, "dark");
  await page.reload();
  await expect(page.getByRole("radio", { name: "System", exact: true })).toBeChecked();
  await page.emulateMedia({ colorScheme: "light" });
  await expectScheme(page, "light");
});

test("compares themes and remembers the choice independently of color mode", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/user");
  const warm = page.getByRole("radio", { name: "Warm", exact: true });
  const softWhite = page.getByRole("radio", { name: "Soft White", exact: true });
  const neutral = page.getByRole("radio", { name: "Neutral", exact: true });
  await expect(warm).toBeChecked();
  await expectScheme(page, "light");

  await warm.press("ArrowRight");
  await expect(softWhite).toBeChecked();
  await expect(softWhite).toBeFocused();
  await expectScheme(page, "light", "soft-white");
  await expect(warm).toHaveCSS("background-color", "rgb(255, 255, 255)");

  await softWhite.press("ArrowRight");
  await expect(neutral).toBeChecked();
  await expect(neutral).toBeFocused();
  await expectScheme(page, "light", "neutral");

  await page.reload();
  await expect(neutral).toBeChecked();
  await expectScheme(page, "light", "neutral");
  await selectRadio(page, "Dark");
  await expect(neutral).toBeChecked();
  await expectScheme(page, "dark", "neutral");
  await selectRadio(page, "Soft White");
  await expectScheme(page, "dark", "soft-white");
  await selectRadio(page, "Warm");
  await expect(page.getByRole("radio", { name: "Dark", exact: true })).toBeChecked();
  await expectScheme(page, "dark");
});

test("syncs the theme across open tabs", async ({ page, context }) => {
  await page.goto("/user");
  const other = await context.newPage();
  await other.goto("/user");
  await expect(other.getByRole("radio", { name: "Warm", exact: true })).toBeChecked();

  await selectRadio(page, "Soft White");
  await expect(other.getByRole("radio", { name: "Soft White", exact: true })).toBeChecked();
  await expect(other.locator("html")).toHaveAttribute("data-theme", "soft-white");
});

test("falls back to System and Warm for invalid stored preferences", async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem("handlat-theme", "invalid");
    localStorage.setItem("handlat-palette", "invalid");
  });
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/user");
  await expect(page.getByRole("radio", { name: "System", exact: true })).toBeChecked();
  await expect(page.getByRole("radio", { name: "Warm", exact: true })).toBeChecked();
  await expectScheme(page, "dark");
});

test("changes color mode using the segmented control's arrow keys", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/user");
  await expect(page.getByRole("radiogroup", { name: "Color mode" })).toBeVisible();
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
      ["Theme", "Warm", "Soft White"],
      ["Color mode", "System", "Light"],
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
  await selectRadio(page, "Soft White");
  await expectScheme(page, "dark", "soft-white");
  await selectRadio(page, "Light");
  await expectScheme(page, "light", "soft-white");
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
