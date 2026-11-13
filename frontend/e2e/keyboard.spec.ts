import { expect, test } from "@playwright/test";

test.use({ hasTouch: true, viewport: { width: 390, height: 844 } });

test("closes the item drawer after keyboard viewport recovery and preserves the draft", async ({
  page,
  context,
}) => {
  await context.credentials.install();
  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account", exact: true }).click();
  await expect(page).toHaveURL(/\/user$/);
  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List", exact: true })).toBeVisible();

  const add = page.getByRole("button", { name: "Add item", exact: true });
  const input = page.getByPlaceholder("t.ex. 2 kg potatis");
  const illustration = page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "What do you need to buy?", exact: true }) })
    .locator("img");
  await expect(illustration).toBeVisible();
  const initialBounds = await illustration.boundingBox();
  expect(initialBounds).not.toBeNull();
  const expectIllustrationInPlace = async () => {
    await expect
      .poll(async () => (await illustration.boundingBox())?.y)
      .toBeCloseTo(initialBounds!.y, 0);
  };
  await add.click();
  await expect(input).toBeFocused();
  await expectIllustrationInPlace();
  let additions = 0;
  page.on("request", (request) => {
    if (request.method() === "POST" && /\/api\/lists\/[^/]+\/items$/.test(request.url())) {
      additions++;
    }
  });
  await input.fill("Milk");

  // resizes-content makes the keyboard shrink the layout viewport and CSS units.
  const keyboardHeight = async (height = 844) => {
    await page.setViewportSize({ width: 390, height });
    await page.evaluate(
      () =>
        new Promise<void>((resolve) => {
          requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
        }),
    );
  };
  await keyboardHeight(500);
  await expect(input).toBeVisible();
  await expectIllustrationInPlace();
  await keyboardHeight();
  await expect(input).not.toBeVisible();
  await expect(add).toBeVisible();
  await expectIllustrationInPlace();

  await add.click();
  await expect(input).toBeFocused();
  await expect(input).toHaveValue("Milk");
  await keyboardHeight(500);
  await expectIllustrationInPlace();
  await page.keyboard.press("Escape");
  await expect(input).not.toBeVisible();
  // Dismiss the input first, then let the keyboard finish its closing animation.
  await expectIllustrationInPlace();
  for (const height of [600, 720, 800]) {
    await keyboardHeight(height);
    await expectIllustrationInPlace();
  }
  await keyboardHeight();
  await expectIllustrationInPlace();
  expect(additions).toBe(0);
});
