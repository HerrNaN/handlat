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
  await expect(page.getByRole("link", { name: "New recipe" })).toBeVisible();
  await page.getByRole("link", { name: "Create recipe" }).click();
  await page.getByRole("textbox", { name: "Recipe name" }).fill("Tomato sauce");
  await page.getByRole("spinbutton", { name: "Yield" }).fill("4");

  const ingredientInput = page.getByRole("textbox", { name: "Ingredient", exact: true });
  await ingredientInput.fill("450ml chopped tomatoes");
  const inputY = (await ingredientInput.boundingBox())?.y;
  await ingredientInput.press("Enter");
  await expect(ingredientInput).toHaveValue("");
  await expect(ingredientInput).toBeFocused();
  const tomatoes = page.getByRole("button", { name: "450 ml Chopped tomatoes" });
  await expect(tomatoes).toBeVisible();
  expect((await ingredientInput.boundingBox())?.y).toBe(inputY);
  await tomatoes.click();

  const editor = page.locator('wa-dialog[label="Edit ingredient"]');
  const editName = editor.getByRole("textbox", { name: "Ingredient name" });
  const editAmount = editor.getByRole("textbox", { name: "Amount" });
  const editUnit = editor.getByRole("combobox", { name: "Unit" });
  await expect(editName).toHaveValue("chopped tomatoes");
  await expect(editAmount).toHaveValue("450");
  await expect(editUnit).toHaveValue("ml");
  await editName.fill("tomatoes");
  await editAmount.fill("400");
  await editUnit.click();
  await editor.getByRole("option", { name: "g", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "400 g Tomatoes", includeHidden: true }),
  ).toHaveCount(1);
  await editor.getByRole("button", { name: "Close", exact: true }).click();
  await expect(editor).toHaveJSProperty("open", false);
  await expect(page.getByRole("button", { name: "400 g Tomatoes" })).toBeVisible();

  await ingredientInput.fill("2 Onions");
  await page.getByRole("button", { name: "Add ingredient", exact: true }).click();
  await expect(page.getByRole("button", { name: "2 Onions" })).toBeVisible();
  await expect(ingredientInput).toBeFocused();
  await ingredientInput.fill("Parsley");
  await ingredientInput.press("Enter");
  const parsley = page.getByRole("button", { name: "Parsley", exact: true });
  await page.getByRole("button", { name: "Remove Parsley", exact: true }).click();
  await expect(parsley).toHaveCount(0);
  await expect(editor).toHaveJSProperty("open", false);
  await expect(ingredientInput).toBeFocused();

  await ingredientInput.fill("Basil");
  await page.getByRole("button", { name: "Save recipe" }).click();

  const recipeHeading = page.getByRole("heading", { name: "Tomato sauce" });
  await expect(recipeHeading).toBeVisible();
  const tomatoName = page.getByText("Tomatoes", { exact: true });
  const tomatoQuantity = page.getByText("400 g", { exact: true });
  const basilName = page.getByText("Basil", { exact: true });
  await expect(tomatoName).toBeVisible();
  await expect(tomatoQuantity).toBeVisible();
  await expect(basilName).toBeVisible();
  const nameX = (await tomatoName.boundingBox())?.x;
  expect(nameX).toBeGreaterThan((await tomatoQuantity.boundingBox())?.x ?? Infinity);
  expect((await basilName.boundingBox())?.x).toBe(nameX);
  const portions = page.getByRole("spinbutton", { name: "Portions", exact: true });
  const addToList = page.getByRole("button", { name: "Add to list", exact: true });
  await expect(portions).toHaveValue("4");

  await page.setViewportSize({ width: 320, height: 300 });
  await expect(portions).toBeInViewport();
  const portionBounds = await portions.boundingBox();
  const actionBounds = await addToList.boundingBox();
  const headerBounds = await recipeHeading.boundingBox();
  const ingredientHeadingBounds = await page
    .getByRole("heading", { name: "Ingredients" })
    .boundingBox();
  const ingredientNameBounds = await basilName.boundingBox();
  if (
    !portionBounds ||
    !actionBounds ||
    !headerBounds ||
    !ingredientHeadingBounds ||
    !ingredientNameBounds
  ) {
    throw new Error("The recipe header, controls, and ingredients must be visible.");
  }
  expect(actionBounds.y).toBeGreaterThan(portionBounds.y + portionBounds.height);
  expect(actionBounds.y).toBeLessThan(ingredientHeadingBounds.y + ingredientHeadingBounds.height);
  expect(actionBounds.y + actionBounds.height).toBeGreaterThan(ingredientHeadingBounds.y);
  expect(actionBounds.x + actionBounds.width).toBeCloseTo(
    ingredientNameBounds.x + ingredientNameBounds.width,
  );

  await basilName.scrollIntoViewIfNeeded();
  await expect.poll(async () => (await portions.boundingBox())?.y).toBeLessThan(portionBounds.y);
  await expect.poll(async () => (await recipeHeading.boundingBox())?.y).toBe(headerBounds.y);
  await expect(page.locator("main > .wrapper")).toHaveJSProperty("scrollTop", 0);
  await page.setViewportSize({ width: 390, height: 800 });
  await portions.fill("0");
  await expect(page.getByRole("alert")).toHaveText("Enter a positive whole number of portions.");
  await expect(addToList).toBeDisabled();
  await portions.fill("3");
  await expect(page.getByText("300 g", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Decrement", exact: true }).click();
  await expect(portions).toHaveValue("2");
  await expect(page.getByText("200 g", { exact: true })).toBeVisible();
  await addToList.click();
  const transferDialog = page.getByRole("dialog", { name: "Add to list", exact: true });
  const transfer = page.locator('wa-dialog[label="Add to list"]');
  await expect(transferDialog).toBeVisible();
  const confirmTransfer = transfer.getByRole("button", { name: "Add to list", exact: true });
  const onions = transfer.getByRole("checkbox", { name: /Onions/ });
  await expect(onions).toBeChecked();
  const onionLabel = transfer.getByText("Onions", { exact: true });
  await onionLabel.click();
  await expect(onions).not.toBeChecked();
  const dialogNameX = (await onionLabel.boundingBox())?.x;
  expect(dialogNameX).toBeGreaterThan(
    (await transfer.getByText("1", { exact: true }).boundingBox())?.x ?? Infinity,
  );
  expect((await transfer.getByText("Basil", { exact: true }).boundingBox())?.x).toBe(dialogNameX);
  const destination = transfer.getByRole("combobox", { name: "Shopping list" });
  await expect(destination).toHaveValue("Shopping List");
  await confirmTransfer.click();
  await expect(transferDialog).toBeHidden();
  await expect(page.getByRole("status")).toHaveText("Ingredients added to Shopping List.");
  await page.getByRole("link", { name: "Open shopping list" }).click();

  await expect(page.getByText("300 g", { exact: true })).toBeVisible();
  await expect(page.getByRole("checkbox", { name: /Tomatoes/ })).toHaveCount(1);
  await expect(page.getByRole("checkbox", { name: "Basil" })).toBeVisible();
  await expect(page.getByRole("checkbox", { name: /Onions/ })).toHaveCount(0);

  const secondList = await page.request.post("/api/lists", { data: { name: "Weekend" } });
  await expect(secondList).toBeOK();
  await page.getByRole("button", { name: "Recipes", exact: true }).click();
  await page.getByRole("link", { name: "Tomato sauce" }).click();
  await expect(portions).toHaveValue("4");
  await expect(page.getByText("400 g", { exact: true })).toBeVisible();
  await addToList.click();
  await expect(destination).toHaveValue("");
  await expect(confirmTransfer).toBeDisabled();
  await destination.click();
  await page.getByRole("option", { name: "Weekend", exact: true }).click();
  await expect(destination).toHaveValue("Weekend");
  await expect(confirmTransfer).toBeEnabled();
  await transfer.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(transferDialog).toBeHidden();
  await page.getByRole("button", { name: "Actions for Tomato sauce" }).click();
  await page.getByRole("menuitem", { name: "Edit recipe" }).click();
  await page.getByRole("button", { name: "400 g Tomatoes" }).click();
  await expect(editName).toHaveValue("tomatoes");
  await editor.getByRole("button", { name: "Close", exact: true }).click();
  await expect(editor).toHaveJSProperty("open", false);
  await page.getByRole("textbox", { name: "Recipe name" }).fill("Sunday tomato sauce");
  await page.getByRole("button", { name: "Save recipe" }).click();
  await expect(page.getByRole("heading", { name: "Sunday tomato sauce" })).toBeVisible();

  const actions = page.getByRole("button", { name: "Actions for Sunday tomato sauce" });
  const deleteAction = page.getByRole("menuitem", { name: "Delete recipe" });
  const confirm = page.locator('wa-dialog[label="Delete recipe?"]');
  await actions.click();
  await deleteAction.click();
  await expect(confirm).toContainText("Sunday tomato sauce");
  await confirm.getByRole("button", { name: "Cancel" }).click();
  await expect(page.getByRole("dialog", { name: "Delete recipe?" })).toBeHidden();
  await expect(page.getByRole("heading", { name: "Sunday tomato sauce" })).toBeVisible();

  await page.getByRole("button", { name: "Back", exact: true }).click();
  await actions.click();
  await page.getByRole("menuitem", { name: "Edit recipe" }).click();
  await expect(page.getByRole("textbox", { name: "Recipe name" })).toHaveValue(
    "Sunday tomato sauce",
  );
  await page.getByRole("button", { name: "Back", exact: true }).click();
  await actions.click();
  await deleteAction.click();
  await expect(confirm).toContainText("Sunday tomato sauce");
  await confirm.getByRole("button", { name: "Delete recipe" }).click();
  await expect(page.getByText(/Save your first recipe/)).toBeVisible();
  await expect(page).toHaveURL(/\/recipes$/);
});
