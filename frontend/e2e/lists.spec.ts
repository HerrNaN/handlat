import { expect, test } from "@playwright/test";

test("keeps item input focused and editable while saving", async ({ page, context }) => {
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

  let releaseSave!: () => void;
  const saveGate = new Promise<void>((resolve) => {
    releaseSave = resolve;
  });
  let requests = 0;
  await page.route("**/api/lists/*/items", async (route) => {
    requests++;
    if (requests === 1) {
      await saveGate;
    }
    await route.continue();
  });
  // Record even temporary focus loss; restoring focus later is insufficient on mobile.
  await input.evaluate((element) => {
    element.setAttribute("data-blurs", "0");
    element.addEventListener("blur", () => {
      element.setAttribute("data-blurs", String(Number(element.getAttribute("data-blurs")) + 1));
    });
  });
  try {
    await page.keyboard.press("Enter");
    await expect.poll(() => requests).toBe(1);
    await expect(input).toHaveValue("");
    await expect(input).toBeFocused();
    await expect(input).toBeEditable();
    // Empty submissions must not create extra requests.
    await page.keyboard.press("Enter");
    expect(requests).toBe(1);
    await page.keyboard.type("Bread");
    await expect(input).toHaveValue("Bread");
  } finally {
    releaseSave();
  }
  await expect(
    page.getByRole("checkbox", { name: "Milk", exact: true, includeHidden: true }),
  ).toHaveCount(1);

  await expect(input).toHaveValue("Bread");
  await expect(input).toBeFocused();
  await expect(input).toHaveAttribute("data-blurs", "0");
  // Use the keyboard directly so the test cannot silently refocus the field.
  await page.keyboard.press("Enter");
  await expect(input).toHaveValue("");
  await expect(
    page.getByRole("checkbox", { name: "Bread", exact: true, includeHidden: true }),
  ).toHaveCount(1);
  await expect(input).toBeFocused();
  await expect(input).toHaveAttribute("data-blurs", "0");
  expect(requests).toBe(2);
  await page.keyboard.press("Escape");
  await expect(
    page.getByRole("checkbox", { name: "Milk", exact: true, includeHidden: true }),
  ).toHaveCount(1);
});

test("creates a list, adds an item, and keeps its checked state after reload", async ({
  page,
  context,
}) => {
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

test("deletes a list using its trash button only after confirmation", async ({ page, context }) => {
  await context.credentials.install();
  await page.goto("/sign-up");
  await page.getByRole("button", { name: "Create account with passkey" }).click();
  await expect(page).toHaveURL(/\/user$/);
  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  const listURL = page.url();
  await page.goto("/lists");

  const deleteButton = page.getByRole("button", { name: "Delete Shopping List", exact: true });
  await deleteButton.click();
  await expect(page).toHaveURL(/\/lists$/);
  const dialog = page.locator("wa-dialog");
  await expect(dialog).toContainText("Shopping List");
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect(page.getByRole("link", { name: /Shopping List/ })).toBeVisible();

  await deleteButton.click();
  let releaseRefresh!: () => void;
  const refreshGate = new Promise<void>((resolve) => {
    releaseRefresh = resolve;
  });
  await page.route("**/api/lists", async (route) => {
    await refreshGate;
    await route.continue();
  });
  try {
    await dialog.getByRole("button", { name: "Delete list", exact: true }).click();
    await expect(page.getByRole("link", { name: /Shopping List/ })).toHaveCount(0);
    // Successful deletion must close the actual modal, even while refresh is stalled.
    await expect(page.getByRole("dialog")).not.toBeVisible();
    await expect(dialog).not.toBeVisible();
  } finally {
    releaseRefresh();
  }
  await expect(page.getByText("No lists yet. Create your first private list.")).toBeVisible();
  await page.reload();
  await expect(page.getByRole("link", { name: /Shopping List/ })).toHaveCount(0);
  const response = await page.request.get(listURL.replace("/lists/", "/api/lists/"));
  expect(response.status()).toBe(404);
});

test("keeps a list after a failed deletion and allows retrying", async ({ page, context }) => {
  await context.credentials.install();
  await page.goto("/sign-up");
  await page.getByRole("button", { name: "Create account with passkey" }).click();
  await expect(page).toHaveURL(/\/user$/);
  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  const listURL = page.url();
  await page.goto("/lists");

  const deleteButton = page.getByRole("button", { name: "Delete Shopping List", exact: true });
  await deleteButton.click();
  await expect(page).toHaveURL(/\/lists$/);
  const dialog = page.locator("wa-dialog");
  await expect(dialog).toContainText("Shopping List");
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect(page.getByRole("link", { name: /Shopping List/ })).toBeVisible();

  await deleteButton.click();
  await page.route("**/api/lists/*", (route) =>
    route.request().method() === "DELETE"
      ? route.fulfill({ status: 500, body: "Server error" })
      : route.continue(),
  );
  await dialog.getByRole("button", { name: "Delete list", exact: true }).click();
  await expect(dialog.getByRole("alert")).toBeVisible();
  await expect(page.getByRole("link", { name: /Shopping List/ })).toBeVisible();
  await page.unroute("**/api/lists/*");
  let releaseRefresh!: () => void;
  const refreshGate = new Promise<void>((resolve) => {
    releaseRefresh = resolve;
  });
  await page.route("**/api/lists", async (route) => {
    await refreshGate;
    await route.continue();
  });
  try {
    await dialog.getByRole("button", { name: "Delete list", exact: true }).click();
    await expect(page.getByRole("link", { name: /Shopping List/ })).toHaveCount(0);
    // Successful deletion must close the actual modal, even while refresh is stalled.
    await expect(page.getByRole("dialog")).not.toBeVisible();
    await expect(dialog).not.toBeVisible();
  } finally {
    releaseRefresh();
  }
  await expect(page.getByText("No lists yet. Create your first private list.")).toBeVisible();
  await page.reload();
  await expect(page.getByRole("link", { name: /Shopping List/ })).toHaveCount(0);
  const response = await page.request.get(listURL.replace("/lists/", "/api/lists/"));
  expect(response.status()).toBe(404);
});
