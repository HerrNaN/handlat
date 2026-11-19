import { expect, test } from "@playwright/test";

test.use({ viewport: { width: 390, height: 800 }, isMobile: true, hasTouch: true });

test("saves a recipe and adds selected scaled ingredients to a shopping list on mobile", async ({
  page,
  context,
}) => {
  test.setTimeout(60_000);
  await context.credentials.install();
  await page.goto("/recipes");
  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);

  await page.goto("/lists");
  await page.getByRole("button", { name: "New list", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
  await page.getByRole("button", { name: "Add item", exact: true }).click();
  const itemInput = page.getByRole("textbox", { name: "Add item", exact: true });
  await itemInput.fill("100 g Tomatoes");
  await itemInput.press("Enter");
  await itemInput.press("Escape");
  await expect(page.getByText("100 g", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Recipes", exact: true }).click();
  await expect(page.getByText(/The format may change/)).toBeVisible();
  await page.getByRole("button", { name: "Enable recipes" }).click();
  await expect(page.getByText(/Save your first recipe/)).toBeVisible();
  await page.getByRole("link", { name: "New recipe" }).click();
  await page.getByRole("textbox", { name: "Recipe name" }).fill("Tomato sauce");
  await page.getByRole("textbox", { name: "Base portions" }).fill("4");
  const first = page.getByRole("group", { name: "Ingredient 1", exact: true });
  await first.getByRole("textbox", { name: "Ingredient name" }).fill("Tomatoes");
  await first.getByRole("textbox", { name: "Amount" }).fill("400");
  await first.getByRole("combobox", { name: "Unit" }).selectOption("g");
  await page.getByRole("button", { name: "Add ingredient", exact: true }).click();
  const second = page.getByRole("group", { name: "Ingredient 2", exact: true });
  await second.getByRole("textbox", { name: "Ingredient name" }).fill("Onions");
  await second.getByRole("textbox", { name: "Amount" }).fill("2");
  await page.getByRole("button", { name: "Add ingredient", exact: true }).click();
  await page
    .getByRole("group", { name: "Ingredient 3", exact: true })
    .getByRole("textbox", { name: "Ingredient name" })
    .fill("Basil");
  await page.getByRole("button", { name: "Save recipe" }).click();

  await expect(page.getByRole("heading", { name: "Tomato sauce" })).toBeVisible();
  await expect(page.getByText("400 g", { exact: true })).toBeVisible();
  await page.getByRole("textbox", { name: "Portions", exact: true }).fill("2");
  await expect(page.getByText("200 g", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Add to shopping list" }).click();
  const onions = page.getByRole("checkbox", { name: /Onions/ });
  await expect(onions).toBeChecked();
  await onions.uncheck();
  await page
    .getByRole("combobox", { name: "Shopping list" })
    .selectOption({ label: "Shopping List" });
  await page.getByRole("button", { name: "Add to shopping list" }).click();
  await expect(page.getByRole("status")).toHaveText("Ingredients added to Shopping List.");
  await page.getByRole("link", { name: "Open shopping list" }).click();

  await expect(page.getByText("300 g", { exact: true })).toBeVisible();
  await expect(page.getByRole("checkbox", { name: /Tomatoes/ })).toHaveCount(1);
  await expect(page.getByRole("checkbox", { name: "Basil" })).toBeVisible();
  await expect(page.getByRole("checkbox", { name: /Onions/ })).toHaveCount(0);

  await page.getByRole("button", { name: "Recipes", exact: true }).click();
  await page.getByRole("link", { name: "Tomato sauce" }).click();
  await expect(page.getByRole("textbox", { name: "Portions", exact: true })).toHaveValue("4");
  await expect(page.getByText("400 g", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Edit recipe" }).click();
  await page.getByRole("textbox", { name: "Recipe name" }).fill("Sunday tomato sauce");
  await page.getByRole("button", { name: "Save recipe" }).click();
  await expect(page.getByRole("heading", { name: "Sunday tomato sauce" })).toBeVisible();

  page.once("dialog", async (dialog) => {
    await dialog.accept();
  });
  await page.getByRole("button", { name: "Delete recipe" }).click();
  await expect(page.getByText(/Save your first recipe/)).toBeVisible();
});
