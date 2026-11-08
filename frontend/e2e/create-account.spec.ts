import { expect, test } from "@playwright/test";

test("creates an account with a passkey and stays logged in after reload", async ({
  page,
  context,
}) => {
  await context.credentials.install();

  await page.goto("/user");
  await expect(page).toHaveURL(/\/login$/);
  await expect(page.getByRole("heading", { name: "Welcome to Handlat" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Login", exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);

  const account = page.getByText(/^You are logged in as: /);
  await expect(account).toBeVisible();
  const name = await account.textContent();
  expect(await context.credentials.get({ rpId: "localhost" })).toHaveLength(1);

  const cookies = await context.cookies();
  const session = cookies.find((cookie) => cookie.name === "laga_session");
  expect(session).toBeDefined();
  expect(session?.secure).toBe(true);
  expect(session?.httpOnly).toBe(true);
  expect(session?.sameSite).toBe("Strict");

  await page.reload();
  await expect(page).toHaveURL(/\/user$/);
  await expect(account).toHaveText(name ?? "");
});
