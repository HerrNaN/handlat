import { expect, test, type Locator, type Page } from "@playwright/test";

// Record short animations without depending on when an assertion polls.
const recordMovement = async (checkbox: Locator) => {
  await checkbox.locator("xpath=ancestor::li").evaluate((row) => {
    const animate = row.animate.bind(row);
    row.dataset.moves = "0";
    row.animate = (frames, options) => {
      if (typeof options === "object" && Number(options.duration) > 0) {
        row.dataset.moves = String(Number(row.dataset.moves) + 1);
      }
      return animate(frames, options);
    };
  });
};
const moves = (checkbox: Locator) => checkbox.locator("xpath=ancestor::li");
const order = (page: Page) => page.getByRole("checkbox").allTextContents();
const settled = async (page: Page) => {
  await expect
    .poll(() =>
      page.locator("li").evaluateAll((rows) => rows.flatMap((row) => row.getAnimations()).length),
    )
    .toBe(0);
};

test.beforeEach(async ({ page, context }) => {
  await page.emulateMedia({ reducedMotion: "no-preference" });
  await context.credentials.install();
  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);
  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  const api = page.url().replace("/lists/", "/api/lists/");
  for (const [id, text, department] of [
    ["00000000-0000-4000-8000-000000000001", "Apple", "produce"],
    ["00000000-0000-4000-8000-000000000002", "Milk", "dairy"],
    ["00000000-0000-4000-8000-000000000003", "Bread", "bakery"],
  ]) {
    const response = await page.request.post(`${api}/items`, { data: { id, text, department } });
    expect(response.ok()).toBe(true);
  }
  await expect(page.getByRole("checkbox")).toHaveCount(3);
});

test("animates between sections, preserves focus and handles rapid reversals", async ({ page }) => {
  const apple = page.getByRole("checkbox", { name: "Apple", exact: true });
  const milk = page.getByRole("checkbox", { name: "Milk", exact: true });
  await recordMovement(apple);
  await recordMovement(milk);
  await apple.focus();
  await page.keyboard.press("Space");
  await expect(apple).toBeChecked();
  await expect(apple).toBeFocused();
  await expect
    .poll(async () => Number(await moves(apple).getAttribute("data-moves")))
    .toBeGreaterThan(0);
  await expect
    .poll(async () => Number(await moves(milk).getAttribute("data-moves")))
    .toBeGreaterThan(0);
  await expect(page.getByRole("heading", { name: "Frukt & grönt", exact: true })).toHaveCount(0);
  await expect.poll(() => order(page)).toEqual(["Milk", "Bread", "Apple"]);
  // Keyboard activation can reverse a row while it is still moving.
  await page.keyboard.press("Space");
  await expect(apple).not.toBeChecked();
  await expect(apple).toBeFocused();
  await expect
    .poll(async () => Number(await moves(apple).getAttribute("data-moves")))
    .toBeGreaterThan(1);
  await settled(page);
  await expect.poll(() => order(page)).toEqual(["Apple", "Milk", "Bread"]);
  await expect(page.getByRole("heading", { name: "Handlat", exact: true })).toHaveCount(0);
});

test("animates streamed changes from another client", async ({ page, context, browser }) => {
  const apple = page.getByRole("checkbox", { name: "Apple", exact: true });
  await recordMovement(apple);
  const otherContext = await browser.newContext({ storageState: await context.storageState() });
  try {
    const other = await otherContext.newPage();
    await other.goto(page.url());
    const otherApple = other.getByRole("checkbox", { name: "Apple", exact: true });
    await otherApple.click();
    await expect(apple).toBeChecked();
    await expect
      .poll(async () => Number(await moves(apple).getAttribute("data-moves")))
      .toBeGreaterThan(0);
    await settled(page);
    await otherApple.click();
    await expect(apple).not.toBeChecked();
    await expect
      .poll(async () => Number(await moves(apple).getAttribute("data-moves")))
      .toBeGreaterThan(1);
    await settled(page);
    await expect.poll(() => order(page)).toEqual(["Apple", "Milk", "Bread"]);
  } finally {
    await otherContext.close();
  }
});

test("animates a failed optimistic change back to its department", async ({ page }) => {
  const apple = page.getByRole("checkbox", { name: "Apple", exact: true });
  await recordMovement(apple);
  let fail!: () => void;
  const gate = new Promise<void>((resolve) => {
    fail = resolve;
  });
  await page.route("**/api/lists/*/items/00000000-0000-4000-8000-000000000001", async (route) => {
    await gate;
    await route.fulfill({ status: 500, body: "Server error" });
  });
  try {
    await apple.click();
    await expect(apple).toBeChecked();
    await expect
      .poll(async () => Number(await moves(apple).getAttribute("data-moves")))
      .toBeGreaterThan(0);
    await settled(page);
  } finally {
    fail();
  }
  await expect(apple).not.toBeChecked();
  await expect
    .poll(async () => Number(await moves(apple).getAttribute("data-moves")))
    .toBeGreaterThan(1);
  await settled(page);
  await expect.poll(() => order(page)).toEqual(["Apple", "Milk", "Bread"]);
});

test("moves instantly when reduced motion is preferred", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  const apple = page.getByRole("checkbox", { name: "Apple", exact: true });
  await recordMovement(apple);
  await apple.click();
  await expect(apple).toBeChecked();
  await expect.poll(() => order(page)).toEqual(["Milk", "Bread", "Apple"]);
  await expect(moves(apple)).toHaveAttribute("data-moves", "0");
  await settled(page);
});
