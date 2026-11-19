import { expect, test } from "@playwright/test";

test.describe("browser language preference", () => {
  test.use({ locale: "sv-SE" });

  test("uses Swedish by default and remembers an explicit choice", async ({ page }) => {
    await page.route("**/api/auth/me", (route) =>
      route.fulfill({ json: { id: "language-user", name: "Language User" } }),
    );
    await page.goto("/user");
    await expect(page.getByRole("heading", { name: "Profil", exact: true })).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute("lang", "sv");
    const username = page.getByText("Language User", { exact: true });
    const boldWeight = await username.evaluate((element) =>
      getComputedStyle(element).getPropertyValue("--wa-font-weight-bold").trim(),
    );
    await expect(username).toHaveCSS("font-weight", boldWeight);

    await page.getByRole("radio", { name: "English", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Profile", exact: true })).toBeVisible();
    await expect(username).toHaveCSS("font-weight", boldWeight);
    await page.reload();
    await expect(page.getByRole("radio", { name: "English", exact: true })).toBeChecked();
    await expect(page.locator("html")).toHaveAttribute("lang", "en");

    const auto = page.getByRole("radio", { name: "Auto", exact: true });
    await auto.click();
    await expect(auto).toBeChecked();
    await expect(page.locator("html")).toHaveAttribute("lang", "sv");
  });
});

test("retranslates a visible API error without resubmitting or clearing the draft", async ({
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
  await page.getByRole("button", { name: "Add item", exact: true }).click();
  const itemInput = page.getByRole("textbox", { name: "Add item", exact: true });
  await itemInput.fill("1.50 pcs mjölk");
  await itemInput.press("Enter");
  await itemInput.press("Escape");
  await expect(page.getByRole("checkbox", { name: "Mjölk 1.50 pcs", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Dairy", exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Actions for Shopping List", exact: true }).click();
  await page.getByRole("menuitem", { name: "Rename", exact: true }).click();
  const dialog = page.locator(
    'wa-dialog[label="Rename list"], wa-dialog[label="Byt namn på listan"]',
  );
  const draft = "x".repeat(201);
  await dialog.getByRole("textbox", { name: /^List name/ }).fill(draft);
  let saves = 0;
  page.on("request", (request) => {
    if (request.method() === "PATCH") saves += 1;
  });
  await dialog.getByRole("button", { name: "Save", exact: true }).click();
  await expect(dialog.getByRole("alert")).toHaveText(
    "The list name must contain between 1 and 200 characters.",
  );

  const profile = await context.newPage();
  await profile.goto("/user");
  await profile.getByRole("radio", { name: "Svenska", exact: true }).click();
  await expect(dialog.getByRole("alert")).toHaveText(
    "Listans namn måste innehålla mellan 1 och 200 tecken.",
  );
  await expect(dialog.getByRole("textbox", { name: /^Listans namn/ })).toHaveValue(draft);
  expect(saves).toBe(1);
  await expect(dialog.getByRole("button", { name: "Stäng", exact: true })).toBeVisible();
  await dialog.getByRole("button", { name: "Avbryt", exact: true }).click();
  await expect(page.getByRole("checkbox", { name: "Mjölk 1,50 st", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Mejeri", exact: true })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Shopping List", exact: true })).toBeVisible();
  await page.getByRole("checkbox", { name: "Mjölk 1,50 st", exact: true }).click();
  await expect(page.getByRole("heading", { name: /^Handlat/ })).toBeVisible();
});
