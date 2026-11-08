import { expect, test } from "@playwright/test";

test("signs in to an existing account with a passkey and stays signed in after reload", async ({
  page,
  context,
}) => {
  await context.credentials.install();

  await page.goto("/sign-up");
  await page.getByRole("button", { name: "Create account with passkey" }).click();
  await expect(page).toHaveURL(/\/user$/);
  const account = page.getByText(/^You are signed in as: /);
  const name = await account.textContent();

  await page.getByRole("button", { name: "Log out" }).click();

  await expect(page).toHaveURL(/\/sign-in$/);
  let releaseLogin!: () => void;
  const loginGate = new Promise<void>((resolve) => {
    releaseLogin = resolve;
  });
  await page.route("**/api/auth/login/begin", async (route) => {
    await loginGate;
    await route.continue();
  });
  try {
    await page.getByRole("button", { name: "Log in with passkey" }).click();
    await expect(page.getByRole("button", { name: "Logging in…" })).toBeDisabled();
    await expect(page.getByRole("button", { name: "Create account with passkey" })).toBeDisabled();
  } finally {
    releaseLogin();
  }
  await expect(page).toHaveURL(/\/user$/);
  await expect(account).toHaveText(name ?? "");
  expect(await context.credentials.get({ rpId: "localhost" })).toHaveLength(1);

  await page.reload();
  await expect(page).toHaveURL(/\/user$/);
  await expect(account).toHaveText(name ?? "");
});
