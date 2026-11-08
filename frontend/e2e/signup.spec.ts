import { expect, test } from "@playwright/test";

test("signs up with a passkey and stays signed in after reload", async ({ page, context }) => {
  await context.credentials.install();

  await page.goto("/user");
  await expect(page).toHaveURL(/\/sign-in$/);
  await expect(page.getByRole("heading", { name: "Welcome to Handlat" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Log in with passkey" })).toBeVisible();

  // A failed request must leave both choices available for another attempt.
  let releaseSignup!: () => void;
  const signupGate = new Promise<void>((resolve) => {
    releaseSignup = resolve;
  });
  await page.route(
    "**/api/auth/register/begin",
    async (route) => {
      await signupGate;
      await route.fulfill({ status: 500, body: "Could not create account" });
    },
    { times: 1 },
  );
  try {
    await page.getByRole("button", { name: "Create account with passkey" }).click();
    await expect(page.getByRole("button", { name: "Creating account…" })).toBeDisabled();
    await expect(page.getByRole("button", { name: "Log in with passkey" })).toBeDisabled();
  } finally {
    releaseSignup();
  }
  await expect(page.getByRole("alert")).toContainText("Could not create account");
  await expect(page.getByRole("button", { name: "Log in with passkey" })).toBeEnabled();
  await expect(page.getByRole("button", { name: "Create account with passkey" })).toBeEnabled();
  await page.getByRole("button", { name: "Create account with passkey" }).click();
  await expect(page).toHaveURL(/\/user$/);

  const account = page.getByText(/^You are signed in as: /);
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
