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
  const illustration = page.locator('img[src*="empty-shopping-list"]');
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

  // Desktop emulation has no software keyboard: simulate its viewport changes.
  await page.setViewportSize({ width: 390, height: 500 });
  // Let the observer read the reduced viewport before simulating dismissal.
  await page.evaluate(
    () =>
      new Promise<void>((resolve) => {
        requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
      }),
  );
  await expect(input).toBeVisible();
  await expectIllustrationInPlace();
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(input).not.toBeVisible();
  await expect(add).toBeVisible();
  await expectIllustrationInPlace();

  await add.click();
  await expect(input).toBeFocused();
  await expect(input).toHaveValue("Milk");
  await page.keyboard.press("Escape");
  await expect(input).not.toBeVisible();
  expect(additions).toBe(0);
});
