import { expect, test } from "@playwright/test";

test("creates a list, adds an item, and keeps its checked state after reload", async ({ page, context }) => {
  await context.credentials.install();
  await page.goto("/sign-up");
  await page.getByRole("button", { name: "Create account with passkey" }).click();
  await expect(page).toHaveURL(/\/user$/);

  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();

  await page.getByRole("button", { name: "Add item", exact: true }).click();
  const input = page.getByPlaceholder("t.ex. 2 kg potatis");
  await input.fill("Milk");
  await input.press("Enter");
  await expect(input).toHaveValue("");
  await input.press("Escape");
  await expect(input).not.toBeVisible();

  const milk = page.getByRole("checkbox", { name: "Milk", exact: true });
  await milk.click();
  await expect(milk).toBeChecked();
  await expect(milk).toBeEnabled();

  await page.reload();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  await expect(milk).toBeChecked();
});
