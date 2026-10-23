import { expect, test, type Page, type BrowserContext } from "@playwright/test";

const register = async (page: Page, context: BrowserContext) => {
  await context.credentials.install();
  await page.goto("/sign-up");
  await page.getByRole("button", { name: "Create account with passkey" }).click();
  await expect(page).toHaveURL(/\/user$/);
};

const createList = async (page: Page) => {
  await page.goto("/lists");
  await expect(page.getByText("No lists yet.", { exact: false })).toBeVisible();
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page).toHaveURL(/\/lists\/[a-f0-9-]+$/);
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
};

const addItem = async (page: Page, text: string) => {
  await page.getByRole("button", { name: "Add item", exact: true }).click();
  const input = page.getByPlaceholder("t.ex. 2 kg potatis");
  await input.fill(text);
  await input.press("Enter");
  await expect(input).toHaveValue("");
  await input.press("Escape");
  await expect(input).not.toBeVisible();
  await expect(page.getByRole("checkbox", { name: text, exact: true })).toBeVisible();
};

test("private lists survive reload and a fresh browser signing in with the same passkey", async ({ page, context, browser }) => {
  await register(page, context);
  // An existing, deliberately non-Loro snapshot must neither be read nor deleted.
  await page.evaluate(() => new Promise<void>((resolve, reject) => {
    const open = indexedDB.open("laga-crdt", 1);
    open.onupgradeneeded = () => open.result.createObjectStore("snapshots");
    open.onerror = () => reject(open.error);
    open.onsuccess = () => {
      const db = open.result;
      const tx = db.transaction("snapshots", "readwrite");
      tx.objectStore("snapshots").put("untouched legacy snapshot", "registry");
      tx.oncomplete = () => { db.close(); resolve(); };
      tx.onerror = () => reject(tx.error);
    };
  }));
  await createList(page);
  const listURL = page.url();
  await page.getByRole("button", { name: "Edit list name" }).click();
  const name = page.locator("header wa-input input");
  await name.fill("Weekend groceries");
  await name.press("Tab");
  await expect(page.getByRole("heading", { name: "Weekend groceries" })).toBeVisible();
  await addItem(page, "Milk");
  await page.reload();
  await expect(page.getByRole("checkbox", { name: "Milk", exact: true })).not.toBeChecked();
  const checkboxVisual = page.getByRole("checkbox", { name: "Milk", exact: true }).locator("wa-checkbox");
  const control = checkboxVisual.locator('[part~="control"]');
  await expect(checkboxVisual.locator('input[type="checkbox"]')).not.toBeChecked();
  const neutralBackground = await checkboxVisual.evaluate((element) => getComputedStyle(element).backgroundColor);
  const uncheckedControl = await control.evaluate((element) => ({
    background: getComputedStyle(element).backgroundColor,
    border: getComputedStyle(element).borderTopColor,
  }));
  await page.getByRole("checkbox", { name: "Milk", exact: true }).click();
  await expect(page.getByRole("checkbox", { name: "Milk", exact: true })).toBeChecked();
  await expect(checkboxVisual.locator('input[type="checkbox"]')).toBeChecked();
  const successColors = await checkboxVisual.evaluate((element) => {
    const probe = document.createElement("span");
    probe.style.backgroundColor = "var(--wa-color-success-fill-quiet)";
    probe.style.color = "var(--wa-color-success-on-loud)";
    probe.style.borderColor = "var(--wa-color-success-fill-loud)";
    element.parentElement!.append(probe);
    const style = getComputedStyle(probe);
    const colors = { background: style.backgroundColor, foreground: style.color, control: style.borderTopColor };
    probe.remove();
    return colors;
  });
  await expect(checkboxVisual).toHaveCSS("background-color", successColors.background);
  await expect(control).toHaveCSS("background-color", successColors.control);
  await expect(control).toHaveCSS("border-top-color", successColors.control);
  await expect(control).toHaveCSS("color", successColors.foreground);
  expect(successColors.background).not.toBe(neutralBackground);
  await page.getByRole("checkbox", { name: "Milk", exact: true }).click();
  await expect(page.getByRole("checkbox", { name: "Milk", exact: true })).not.toBeChecked();
  await expect(checkboxVisual).toHaveCSS("background-color", neutralBackground);
  await expect(checkboxVisual.locator('input[type="checkbox"]')).not.toBeChecked();
  await expect(control).toHaveCSS("background-color", uncheckedControl.background);
  await expect(control).toHaveCSS("border-top-color", uncheckedControl.border);
  await page.getByRole("button", { name: "Edit Milk", exact: true }).click();
  const itemName = page.locator("wa-dialog wa-input input");
  await itemName.fill("Oat milk");
  const saved = page.waitForResponse((response) => response.request().method() === "PATCH" && response.url().includes("/items/"));
  await itemName.press("Tab");
  expect((await saved).ok()).toBe(true);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("checkbox", { name: "Oat milk", exact: true })).toBeVisible();
  expect(await page.evaluate(() => new Promise((resolve, reject) => {
    const open = indexedDB.open("laga-crdt", 1);
    open.onerror = () => reject(open.error);
    open.onsuccess = () => {
      const db = open.result;
      const tx = db.transaction("snapshots", "readonly");
      const request = tx.objectStore("snapshots").get("registry");
      request.onsuccess = () => resolve(request.result);
      tx.oncomplete = () => db.close();
    };
  }))).toBe("untouched legacy snapshot");

  // Seed only the passkey, not cookies or browser data, into another browser.
  const state = await context.storageState({ credentials: true });
  const fresh = await browser.newContext({
    baseURL: new URL(listURL).origin,
    storageState: { ...state, cookies: [], origins: [] },
  });
  try {
    const otherPage = await fresh.newPage();
    await otherPage.goto("/sign-in");
    await otherPage.getByRole("button", { name: "Sign in with passkey" }).click();
    await expect(otherPage).toHaveURL(/\/user$/);
    await otherPage.goto(listURL);
    await expect(otherPage.getByRole("heading", { name: "Weekend groceries" })).toBeVisible();
    await expect(otherPage.getByRole("checkbox", { name: "Oat milk", exact: true })).toBeVisible();
    const list = await fresh.request.get(new URL(listURL).pathname.replace("/lists", "/api/lists"));
    expect(list.headers()["cache-control"]).toBe("no-store");
    expect(await otherPage.evaluate(() => indexedDB.databases())).toEqual([]);
  } finally {
    await fresh.close();
  }

  await page.goto("/user");
  await page.getByRole("button", { name: "Log out" }).click();
  await expect(page).toHaveURL(/\/sign-in$/);
  expect((await context.request.get(new URL(listURL).pathname.replace("/lists", "/api/lists"))).status()).toBe(401);
});

test("network failures are visible but competing edits are resolved by the backend", async ({ page, context }) => {
  await register(page, context);
  await createList(page);
  await addItem(page, "Bread");
  const checkbox = page.getByRole("checkbox", { name: "Bread", exact: true });
  await context.setOffline(true);
  await checkbox.click();
  await expect(page.getByRole("alert")).toContainText("internet connection");
  await expect(checkbox).not.toBeChecked();
  await page.getByRole("button", { name: "Add item", exact: true }).click();
  const draft = page.getByPlaceholder("t.ex. 2 kg potatis");
  await draft.fill("Not saved");
  await draft.press("Enter");
  await expect(draft).toHaveValue("Not saved");
  await draft.press("Escape");
  await context.setOffline(false);
  await page.getByRole("button", { name: "Refresh", exact: true }).click();
  await expect(checkbox).not.toBeChecked();
  await expect(page.getByRole("checkbox", { name: "Not saved", exact: true })).toHaveCount(0);

  const apiURL = new URL(page.url()).pathname.replace("/lists", "/api/lists");
  const current = await (await context.request.get(apiURL)).json();
  const changed = await context.request.patch(apiURL, { data: { name: "Changed elsewhere" } });
  expect(changed.ok()).toBe(true);
  const itemURL = `${apiURL}/items/${current.items[0].id}`;
  expect((await context.request.patch(itemURL, { data: { checked: true, department: "other" } })).ok()).toBe(true);
  // The screen still shows unchecked. Clicking means "checked", not "toggle";
  // it must not undo the other client's check or require a conflict refresh.
  await checkbox.click();
  await expect(checkbox).toBeChecked();
  await expect(page.getByRole("alert")).toHaveCount(0);
  await expect(page.getByRole("heading", { name: "Changed elsewhere" })).toBeVisible();
  await page.getByRole("button", { name: "Clear", exact: true }).click();
  await expect(checkbox).toHaveCount(0);
  await page.reload();
  await expect(page.getByRole("heading", { name: "Changed elsewhere" })).toBeVisible();
  await expect(checkbox).toHaveCount(0);
});

test("retrying an addition after a lost response does not duplicate it", async ({ page, context }) => {
  await register(page, context);
  await createList(page);
  const apiURL = new URL(page.url()).pathname.replace("/lists", "/api/lists");
  await page.route("**/api/lists/*/items", async (route) => {
    // Commit the request but lose the response on its way back to the browser.
    expect((await route.fetch()).ok()).toBe(true);
    await route.abort("failed");
  });
  await page.getByRole("button", { name: "Add item", exact: true }).click();
  const draft = page.getByPlaceholder("t.ex. 2 kg potatis");
  await draft.fill("Milk");
  await draft.press("Enter");
  await expect(page.getByRole("alert").first()).toContainText("internet connection");
  await expect(draft).toHaveValue("Milk");
  const committed = await (await context.request.get(apiURL)).json();
  expect(committed.items).toHaveLength(1);
  await page.unroute("**/api/lists/*/items");
  await draft.press("Enter");
  await expect(draft).toHaveValue("");
  await draft.press("Escape");
  await expect(draft).not.toBeVisible();
  const retried = await (await context.request.get(apiURL)).json();
  expect(retried.items).toEqual(committed.items);
  await expect(page.getByRole("checkbox", { name: "Milk", exact: true })).toHaveCount(1);
});

test("an edit of an item deleted elsewhere is a harmless no-op", async ({ page, context }) => {
  await register(page, context);
  await createList(page);
  await addItem(page, "Deleted elsewhere");
  const apiURL = new URL(page.url()).pathname.replace("/lists", "/api/lists");
  const current = await (await context.request.get(apiURL)).json();
  expect((await context.request.delete(`${apiURL}/items/${current.items[0].id}`)).ok()).toBe(true);
  const checkbox = page.getByRole("checkbox", { name: "Deleted elsewhere", exact: true });
  await checkbox.click();
  await expect(checkbox).toHaveCount(0);
  await expect(page.getByRole("alert")).toHaveCount(0);
  expect((await (await context.request.get(apiURL)).json()).items).toEqual([]);
});

test("another account cannot discover or open a private list", async ({ page, context, browser }) => {
  await register(page, context);
  await createList(page);
  await addItem(page, "Private item");
  const listURL = page.url();
  const stranger = await browser.newContext({ baseURL: new URL(listURL).origin });
  try {
    const strangerPage = await stranger.newPage();
    await register(strangerPage, stranger);
    expect(await (await stranger.request.get("/api/lists")).json()).toEqual([]);
    await strangerPage.goto(listURL);
    await expect(strangerPage.getByRole("alert")).toContainText("no longer available");
    await expect(strangerPage.getByText("Private item", { exact: true })).toHaveCount(0);
    await expect(strangerPage.getByRole("button", { name: "Add item", exact: true })).toHaveCount(0);
  } finally {
    await stranger.close();
  }
});

test("a rejected session clears private data and asks the user to sign in", async ({ page, context }) => {
  await register(page, context);
  await createList(page);
  await addItem(page, "Secret item");
  await context.clearCookies();
  await page.getByRole("checkbox", { name: "Secret item", exact: true }).click();
  await expect(page.getByRole("alert")).toContainText("session expired");
  await expect(page.getByRole("alert").getByRole("link", { name: "Sign in" })).toBeVisible();
  await expect(page.getByText("Secret item", { exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Add item", exact: true })).toHaveCount(0);
  await page.reload();
  await expect(page).toHaveURL(/\/sign-in$/);
});

test("a failed swipe deletion restores the row and can be retried", async ({ page, context }) => {
  await register(page, context);
  await createList(page);
  await addItem(page, "Delete me");
  const checkbox = page.getByRole("checkbox", { name: "Delete me", exact: true });
  const swipe = async () => {
    const box = await checkbox.boundingBox();
    expect(box).not.toBeNull();
    if (!box) throw new Error("Missing item row");
    await page.mouse.move(box.x + box.width - 10, box.y + box.height / 2);
    await page.mouse.down();
    await page.mouse.move(box.x + 10, box.y + box.height / 2, { steps: 6 });
    await page.mouse.up();
  };
  await page.route("**/api/lists/*/items/*", (route) => route.abort("failed"));
  await swipe();
  await expect(page.getByRole("alert")).toContainText("internet connection");
  await expect(checkbox).toBeInViewport();
  // Being partially in the viewport does not mean the reset animation is done.
  // A second drag must start from the row's settled position.
  await expect(checkbox.locator("../..")).toHaveCSS("transform", "matrix(1, 0, 0, 1, 0, 0)");
  await page.unroute("**/api/lists/*/items/*");
  await swipe();
  await expect(checkbox).toHaveCount(0);
  await page.reload();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  await expect(checkbox).toHaveCount(0);
});
