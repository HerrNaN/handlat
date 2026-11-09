import { expect, test } from "@playwright/test";

test("shares a list with a logged-out invitee who creates an account and accepts", async ({
  page,
  context,
  browser,
}) => {
  await context.credentials.install();
  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account", exact: true }).click();
  await expect(page).toHaveURL(/\/user$/);
  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  const listURL = page.url();

  // Desktop browsers use the copy fallback. Keep the real invitation creation.
  await page.evaluate(() =>
    Object.defineProperty(navigator, "share", { value: undefined, configurable: true }),
  );
  await page.getByRole("button", { name: "Share list", exact: true }).click();
  const link = await page.getByRole("textbox", { name: "Invitation link" }).inputValue();
  expect(link).toContain("/invitations/");
  await page.getByRole("button", { name: "Done", exact: true }).click();

  const inviteeContext = await browser.newContext({ baseURL: new URL(listURL).origin });
  try {
    await inviteeContext.credentials.install();
    const invitee = await inviteeContext.newPage();
    await invitee.goto(link);
    await expect(invitee).toHaveURL(link);
    await expect(invitee.getByText("You've been invited to join")).toContainText("Shopping List");
    await expect(invitee.getByRole("button", { name: "Login", exact: true })).toBeVisible();
    await expect(
      invitee.getByRole("button", { name: "Accept invitation", exact: true }),
    ).toHaveCount(0);

    // Account creation happens inline and does not accept the invitation.
    await invitee.getByRole("button", { name: "Create Account", exact: true }).click();
    await expect(
      invitee.getByRole("button", { name: "Accept invitation", exact: true }),
    ).toBeVisible();
    await expect(invitee).toHaveURL(link);
    const beforeAccept = await invitee.request.get("/api/lists");
    expect(await beforeAccept.json()).toEqual([]);
    await invitee.getByRole("button", { name: "Accept invitation", exact: true }).click();
    await expect(invitee).toHaveURL(listURL);
    await expect(invitee.getByRole("heading", { name: "Shopping List" })).toBeVisible();

    // Both independent users can collaborate through the existing stream.
    await invitee.getByRole("button", { name: "Add item", exact: true }).click();
    const input = invitee.getByPlaceholder("t.ex. 2 kg potatis");
    await input.fill("Milk");
    await input.press("Enter");
    await input.press("Escape");
    await expect(page.getByRole("checkbox", { name: "Milk", exact: true })).toBeVisible();
    await page.getByRole("checkbox", { name: "Milk", exact: true }).click();
    await expect(invitee.getByRole("checkbox", { name: "Milk", exact: true })).toBeChecked();

    await invitee.getByRole("button", { name: "Back", exact: true }).click();
    await expect(invitee.getByRole("link", { name: /Shopping List/ })).toBeVisible();
    await invitee.getByRole("button", { name: "Leave Shopping List", exact: true }).click();
    await invitee.getByRole("button", { name: "Leave list", exact: true }).click();
    await expect(invitee.getByRole("link", { name: /Shopping List/ })).toHaveCount(0);
    await expect(page.getByRole("checkbox", { name: "Milk", exact: true })).toBeChecked();

    await invitee.goto(link);
    await expect(invitee.getByText("This invitation is no longer available.")).toBeVisible();
  } finally {
    await inviteeContext.close();
  }
});
