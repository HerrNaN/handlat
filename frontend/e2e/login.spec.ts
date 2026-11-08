import { expect, test } from "@playwright/test";

test("logs in to an existing account with a passkey and stays logged in after reload", async ({
  page,
  context,
}) => {
  await context.credentials.install();

  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);
  const account = page.getByText(/^You are logged in as: /);
  const name = await account.textContent();

  await page.getByRole("button", { name: "Log out" }).click();

  await expect(page).toHaveURL(/\/login$/);
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page).toHaveURL(/\/user$/);
  await expect(account).toHaveText(name ?? "");
  expect(await context.credentials.get({ rpId: "localhost" })).toHaveLength(1);

  await page.reload();
  await expect(page).toHaveURL(/\/user$/);
  await expect(account).toHaveText(name ?? "");
});
