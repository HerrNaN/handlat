import { expect, test, type Page } from "@playwright/test";

const order = async (page: Page) =>
  (await page.getByRole("checkbox").allTextContents()).map((text) => text.trim());
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

test("animates the moved item and surrounding rows", async ({ page }) => {
  // Keep transitions observable until the test explicitly finishes them.
  await page.addStyleTag({ content: ":root { --wa-transition-normal: 3600s; }" });
  const apple = page.getByRole("checkbox", { name: "Apple", exact: true });
  const milk = page.getByRole("checkbox", { name: "Milk", exact: true });
  await apple.click();
  await expect(apple).toBeChecked();
  for (const item of [apple, milk]) {
    await expect
      .poll(() => item.evaluate((element) => element.closest("li")!.getAnimations().length))
      .toBeGreaterThan(0);
  }
  await page.locator("li").evaluateAll((rows) => {
    for (const row of rows) {
      for (const animation of row.getAnimations()) animation.finish();
    }
  });
  await settled(page);
  await expect.poll(() => order(page)).toEqual(["Milk", "Bread", "Apple"]);
});

test("preserves focus and ordering when checking and immediately unchecking", async ({ page }) => {
  const apple = page.getByRole("checkbox", { name: "Apple", exact: true });
  await apple.focus();
  await page.keyboard.press("Space");
  await expect(apple).toBeChecked();
  await expect(apple).toBeFocused();
  await expect(page.getByRole("heading", { name: "Fruit & vegetables", exact: true })).toHaveCount(
    0,
  );
  await expect.poll(() => order(page)).toEqual(["Milk", "Bread", "Apple"]);
  const checkedHeading = page.getByRole("heading", { name: /^Handlat/ });
  await expect(checkedHeading).toBeVisible();
  // Keyboard activation can reverse a row while it is still moving.
  await page.keyboard.press("Space");
  await expect(apple).not.toBeChecked();
  await expect(apple).toBeFocused();
  await settled(page);
  await expect.poll(() => order(page)).toEqual(["Apple", "Milk", "Bread"]);
  await expect(checkedHeading).toHaveCount(0);
});

test("applies checked state and ordering from another client", async ({
  page,
  context,
  browser,
}) => {
  const apple = page.getByRole("checkbox", { name: "Apple", exact: true });
  const otherContext = await browser.newContext({ storageState: await context.storageState() });
  try {
    const other = await otherContext.newPage();
    await other.goto(page.url());
    const otherApple = other.getByRole("checkbox", { name: "Apple", exact: true });
    await otherApple.click();
    await expect(apple).toBeChecked();
    await expect.poll(() => order(page)).toEqual(["Milk", "Bread", "Apple"]);
    await settled(page);
    await otherApple.click();
    await expect(apple).not.toBeChecked();
    await settled(page);
    await expect.poll(() => order(page)).toEqual(["Apple", "Milk", "Bread"]);
  } finally {
    await otherContext.close();
  }
});

test("restores checked state and ordering after a failed save", async ({ page }) => {
  const apple = page.getByRole("checkbox", { name: "Apple", exact: true });
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
    await settled(page);
  } finally {
    fail();
  }
  await expect(apple).not.toBeChecked();
  await settled(page);
  await expect.poll(() => order(page)).toEqual(["Apple", "Milk", "Bread"]);
});

test("moves instantly when reduced motion is preferred", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.addStyleTag({ content: ":root { --wa-transition-normal: 3600s; }" });
  const apple = page.getByRole("checkbox", { name: "Apple", exact: true });
  await apple.click();
  await expect(apple).toBeChecked();
  await expect.poll(() => order(page)).toEqual(["Milk", "Bread", "Apple"]);
  expect(await apple.evaluate((element) => element.closest("li")!.getAnimations().length)).toBe(0);
  await settled(page);
});
