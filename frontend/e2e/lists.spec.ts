import { expect, test } from "@playwright/test";

test("keeps item input focused and editable while saving", async ({ page, context }) => {
  await context.credentials.install();
  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account" }).click();
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
  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);

  await page.goto("/lists");
  await expect(page.getByRole("heading", { name: "Your first list starts here" })).toBeVisible();
  await page.getByRole("button", { name: "Create list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  const emptyList = page.getByRole("heading", { name: "What do you need to buy?" });
  await expect(emptyList).toBeVisible();

  await page.getByRole("button", { name: "Add item", exact: true }).click();
  const input = page.getByPlaceholder("t.ex. 2 kg potatis");
  await input.fill("Milk");
  await input.press("Enter");
  await expect(input).toHaveValue("");
  await input.press("Escape");
  await expect(input).not.toBeVisible();

  await expect(emptyList).not.toBeVisible();
  const milk = page.getByRole("checkbox", { name: "Milk", exact: true });
  await milk.click();
  await expect(milk).toBeChecked();
  await expect(milk).toBeEnabled();
  await expect(emptyList).not.toBeVisible();

  await page.reload();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  await expect(milk).toBeChecked();
});

test("deletes a list from its actions menu only after confirmation", async ({ page, context }) => {
  await context.credentials.install();
  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);
  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  const listURL = page.url();
  await page.goto("/lists");

  const actions = page.getByRole("button", { name: "Actions for Shopping List", exact: true });
  const deleteButton = page.getByRole("menuitem", { name: "Delete list", exact: true });
  await actions.click();
  await deleteButton.click();
  await expect(page).toHaveURL(/\/lists$/);
  const dialog = page.locator('wa-dialog[label="Delete list?"]');
  await expect(dialog).toContainText("Shopping List");
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect(page.getByRole("link", { name: /Shopping List/ })).toBeVisible();

  await actions.click();
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
  await expect(page.getByRole("heading", { name: "Your first list starts here" })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("link", { name: /Shopping List/ })).toHaveCount(0);
  const response = await page.request.get(listURL.replace("/lists/", "/api/lists/"));
  expect(response.status()).toBe(404);
});

test("keeps a list after a failed deletion and allows retrying", async ({ page, context }) => {
  await context.credentials.install();
  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);
  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  const listURL = page.url();
  await page.goto("/lists");

  const actions = page.getByRole("button", { name: "Actions for Shopping List", exact: true });
  const deleteButton = page.getByRole("menuitem", { name: "Delete list", exact: true });
  await actions.click();
  await deleteButton.click();
  await expect(page).toHaveURL(/\/lists$/);
  const dialog = page.locator('wa-dialog[label="Delete list?"]');
  await expect(dialog).toContainText("Shopping List");
  await dialog.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(dialog).not.toBeVisible();
  await expect(page.getByRole("link", { name: /Shopping List/ })).toBeVisible();

  await actions.click();
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
  await expect(page.getByRole("heading", { name: "Your first list starts here" })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("link", { name: /Shopping List/ })).toHaveCount(0);
  const response = await page.request.get(listURL.replace("/lists/", "/api/lists/"));
  expect(response.status()).toBe(404);
});

test("streams list changes between independent clients without refreshing", async ({
  page,
  context,
  browser,
}) => {
  await context.credentials.install();
  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);
  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();

  // Separate browser contexts share authentication, not application/query state.
  const otherContext = await browser.newContext({ storageState: await context.storageState() });
  try {
    const other = await otherContext.newPage();
    await Promise.all([
      other.waitForResponse(
        (response) => response.url().endsWith("/events") && response.status() === 200,
      ),
      other.goto(page.url()),
    ]);
    await expect(other.getByRole("heading", { name: "Shopping List" })).toBeVisible();

    await page.getByRole("button", { name: "Add item", exact: true }).click();
    const input = page.getByPlaceholder("t.ex. 2 kg potatis");
    await input.fill("Milk");
    await input.press("Enter");
    await input.press("Escape");
    const otherMilk = other.getByRole("checkbox", { name: "Milk", exact: true });
    await expect(otherMilk).toBeVisible();

    // Exercise the reverse direction using the real UI and backend as well.
    await otherMilk.click();
    await expect(page.getByRole("checkbox", { name: "Milk", exact: true })).toBeChecked();
  } finally {
    await otherContext.close();
  }
});

test("renames a list from both actions menus and deletes it from the open list", async ({
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
  const actions = page.getByRole("button", { name: "Actions for Shopping List", exact: true });
  await actions.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("menuitem", { name: "Rename", exact: true })).toBeFocused();
  await page.keyboard.press("Enter");
  const rename = page.locator('wa-dialog[label="Rename list"]');
  await expect(rename.getByRole("dialog")).toBeVisible();
  const name = rename.getByRole("textbox", { name: /^List name/ });
  await expect(name).toBeFocused();
  await name.fill("Weekly groceries");
  await name.press("Enter");
  await expect(rename.getByRole("dialog")).not.toBeVisible();
  await expect(page.getByRole("heading", { name: "Weekly groceries", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Back", exact: true }).click();
  await page.getByRole("button", { name: "Actions for Weekly groceries", exact: true }).click();
  await page.getByRole("menuitem", { name: "Rename", exact: true }).click();
  await name.fill("Weekend groceries");
  await rename.getByRole("button", { name: "Save", exact: true }).click();
  await expect(rename.getByRole("dialog")).not.toBeVisible();
  await page.getByRole("link", { name: /Weekend groceries/ }).click();
  await expect(page.getByRole("heading", { name: "Weekend groceries", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Actions for Weekend groceries", exact: true }).click();
  await page.getByRole("menuitem", { name: "Delete list", exact: true }).click();
  await page
    .locator('wa-dialog[label="Delete list?"]')
    .getByRole("button", { name: "Delete list", exact: true })
    .click();
  await expect(page).toHaveURL(/\/lists$/);
  await expect(page.getByRole("link", { name: /Weekend groceries/ })).toHaveCount(0);
});

test("parses quantities and edits amount and unit independently", async ({ page, context }) => {
  await context.credentials.install();
  await page.goto("/login");
  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);
  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await page.getByRole("button", { name: "Add item", exact: true }).click();
  const input = page.getByPlaceholder("t.ex. 2 kg potatis");
  await input.fill("1,5 kg Potatoes");
  await input.press("Enter");
  await input.press("Escape");
  await expect(page.getByRole("checkbox", { name: "Potatoes 1,5 kg", exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Edit Potatoes", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Edit Item", exact: true });
  const amount = dialog.getByRole("textbox", { name: "Amount", exact: true });
  const unit = dialog.getByRole("combobox", { name: "Unit", exact: true });
  await expect(amount).toHaveValue("1,5");

  const patch = () => page.waitForRequest((request) => request.method() === "PATCH");
  let update = patch();
  await amount.fill("2.25");
  await amount.press("Tab");
  expect((await update).postDataJSON()).toEqual({ amount: "2.25" });
  await expect(amount).toBeEnabled();
  await expect(page.getByRole("checkbox", { name: "Potatoes 2,25 kg", exact: true })).toHaveCount(
    1,
  );

  update = patch();
  await unit.click();
  await page.getByRole("option", { name: "g", exact: true }).click();
  expect((await update).postDataJSON()).toEqual({ unit: "g" });
  await expect(amount).toBeEnabled();
  await expect(amount).toHaveValue("2,25");

  update = patch();
  await amount.fill("");
  await amount.press("Tab");
  expect((await update).postDataJSON()).toEqual({ amount: null });
  await expect(amount).toBeEnabled();
  await expect(page.getByRole("checkbox", { name: "Potatoes g", exact: true })).toHaveCount(1);

  update = patch();
  await unit.click();
  await page.getByRole("option", { name: "No unit", exact: true }).click();
  expect((await update).postDataJSON()).toEqual({ unit: null });
  await expect(amount).toBeEnabled();
  await dialog.getByRole("button", { name: "Close", exact: true }).click();
  await expect(page.getByRole("checkbox", { name: "Potatoes", exact: true })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("checkbox", { name: "Potatoes", exact: true })).toBeVisible();
});
