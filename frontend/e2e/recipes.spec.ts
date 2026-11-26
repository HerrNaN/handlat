import { randomUUID } from "node:crypto";

import { expect, test, type Locator, type Page } from "@playwright/test";

test.use({ viewport: { width: 390, height: 800 }, isMobile: true, hasTouch: true });

const postFixture = async (page: Page, path: string, data?: object) =>
  page.evaluate(
    async ({ path, data }) => {
      const response = await fetch(path, {
        method: "POST",
        ...(data && {
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(data),
        }),
      });
      return response.status;
    },
    { path, data },
  );

const openSavedRecipe = async (page: Page) => {
  const id = randomUUID();
  expect(
    await postFixture(page, "/api/recipes", {
      id,
      name: "Tomato sauce",
      portions: 4,
      ingredients: [
        { name: "tomatoes", amount: "400", unit: "g" },
        { name: "Onions", amount: "2" },
        { name: "Basil" },
      ],
    }),
  ).toBe(201);
  await page.goto(`/recipes/${id}`);
  await expect(page.getByRole("heading", { name: "Tomato sauce" })).toBeVisible();
};

const expectPinned = async (scrollBody: Locator, control: Locator) => {
  await expect(control).toBeInViewport();
  const y = (await control.boundingBox())?.y;
  const scrollTop = await scrollBody.evaluate((element) => {
    element.scrollTop = element.scrollHeight;
    return element.scrollTop;
  });
  expect(scrollTop).toBeGreaterThan(0);
  await expect(control).toBeInViewport();
  expect((await control.boundingBox())?.y).toBe(y);
};

test.beforeEach(async ({ page, context }) => {
  await context.credentials.install();
  await page.goto("/recipes");
  await page.getByRole("button", { name: "Create Account" }).click();
  await expect(page).toHaveURL(/\/user$/);
});

test("requires acknowledgement before enabling Recipes", async ({ page }) => {
  await page.goto("/recipes");
  await expect(page).toHaveURL(/\/user$/);
  const recipesNav = page.getByRole("button", { name: "Recipes", exact: true });
  const experimentalFeatures = page.getByRole("region", { name: "Experimental features" });
  const openOptIn = experimentalFeatures.getByRole("button", { name: "Enable", exact: true });
  const optIn = page.locator('wa-dialog[label="Enable recipes"]');
  const optInDialog = page.getByRole("dialog", { name: "Enable recipes" });
  const acknowledge = optIn.getByRole("checkbox", { name: "I understand" });
  const enableRecipes = optIn.getByRole("button", { name: "Enable", exact: true });
  const nav = page.getByRole("navigation");
  await expect(recipesNav).toHaveCount(0);
  await nav.getByRole("button", { name: "About Recipes" }).tap();
  await expect(nav.getByText("Enable in profile", { exact: true })).toBeVisible();
  await expect(experimentalFeatures.getByText(/Save your recipes/)).toBeVisible();
  await openOptIn.click();
  await expect(optInDialog).toBeVisible();
  await expect(optIn.locator('wa-callout[variant="warning"]')).toContainText(
    "The format may change",
  );
  await expect(acknowledge).not.toBeChecked();
  await expect(enableRecipes).toBeDisabled();
  await optIn.getByText("I understand", { exact: true }).click();
  await optIn.getByRole("button", { name: "Close", exact: true }).click();
  await expect(optInDialog).toBeHidden();
  await openOptIn.click();
  await expect(acknowledge).not.toBeChecked();
  await expect(enableRecipes).toBeDisabled();
  await optIn.getByText("I understand", { exact: true }).click();
  await expect(acknowledge).toBeChecked();
  await expect(enableRecipes).toBeEnabled();
  await enableRecipes.click();
  await expect(optInDialog).toBeHidden();
  await expect(recipesNav).toBeEnabled();
  await page.reload();
  const enabledStatus = experimentalFeatures.getByRole("status");
  await expect(enabledStatus).toHaveText("Enabled");
  await expect(enabledStatus.locator("wa-icon")).toBeVisible();
  await expect(recipesNav).toBeEnabled();
});

test.describe("with Recipes enabled", () => {
  test.beforeEach(async ({ page }) => {
    expect(await postFixture(page, "/api/recipes/enable")).toBe(204);
  });

  test("creates a recipe from the empty collection and keeps ingredient entry focused", async ({
    page,
  }) => {
    await page.goto("/recipes");
    await expect(page.getByText(/Save your first recipe/)).toBeVisible();
    await expect(page.getByRole("link", { name: "New recipe" })).toBeVisible();
    await page.getByRole("link", { name: "Create recipe" }).click();
    await page.getByRole("textbox", { name: "Recipe name" }).fill("Tomato sauce");
    await page.getByRole("spinbutton", { name: "Yield" }).fill("4");

    const ingredientInput = page.getByRole("textbox", { name: "Ingredient", exact: true });
    await ingredientInput.fill("400g tomatoes");
    const inputY = (await ingredientInput.boundingBox())?.y;
    await ingredientInput.press("Enter");
    await expect(ingredientInput).toHaveValue("");
    await expect(ingredientInput).toBeFocused();
    const tomatoes = page.getByRole("button", { name: "400 g Tomatoes" });
    await expect(tomatoes).toBeVisible();
    expect((await ingredientInput.boundingBox())?.y).toBe(inputY);

    await ingredientInput.fill("2 Onions");
    await page.getByRole("button", { name: "Add ingredient", exact: true }).click();
    await expect(page.getByRole("button", { name: "2 Onions" })).toBeVisible();
    await expect(ingredientInput).toBeFocused();

    await ingredientInput.fill("Basil");
    await page.getByRole("button", { name: "Save recipe" }).click();
    await expect(page.getByRole("heading", { name: "Tomato sauce" })).toBeVisible();
    await expect(page.getByRole("spinbutton", { name: "Portions", exact: true })).toHaveValue("4");
    await expect(page.getByText("400 g", { exact: true })).toBeVisible();
    await expect(page.getByText("Basil", { exact: true })).toBeVisible();
  });

  test("edits and removes ingredients in a recipe draft", async ({ page }) => {
    await page.goto("/recipes/new");
    const ingredientInput = page.getByRole("textbox", { name: "Ingredient", exact: true });
    await ingredientInput.fill("450ml chopped tomatoes");
    await ingredientInput.press("Enter");
    await page.getByRole("button", { name: "450 ml Chopped tomatoes" }).click();

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

    await ingredientInput.fill("Parsley");
    await ingredientInput.press("Enter");
    const parsley = page.getByRole("button", { name: "Parsley", exact: true });
    await page.getByRole("button", { name: "Remove Parsley", exact: true }).click();
    await expect(parsley).toHaveCount(0);
    await expect(editor).toHaveJSProperty("open", false);
    await expect(ingredientInput).toBeFocused();
  });

  test("lays out ingredients and keeps the recipe header pinned on mobile", async ({ page }) => {
    await openSavedRecipe(page);
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
    const addToList = page.getByRole("button", { name: "Add to shopping list", exact: true });
    await expect(portions).toHaveValue("4");

    await page.setViewportSize({ width: 320, height: 300 });
    await expect(portions).toBeInViewport();
    const portionBounds = await portions.boundingBox();
    const actionBounds = await addToList.boundingBox();
    const headerBounds = await recipeHeading.boundingBox();
    const ingredientHeadingBounds = await page
      .getByRole("heading", { name: "Ingredients" })
      .boundingBox();
    const ingredientRowBounds = await page
      .getByRole("listitem")
      .filter({ has: basilName })
      .boundingBox();
    if (
      !portionBounds ||
      !actionBounds ||
      !headerBounds ||
      !ingredientHeadingBounds ||
      !ingredientRowBounds
    ) {
      throw new Error("The recipe header, controls, and ingredients must be visible.");
    }
    expect(actionBounds.y).toBeGreaterThan(portionBounds.y + portionBounds.height);
    expect(actionBounds.y).toBeLessThan(ingredientHeadingBounds.y + ingredientHeadingBounds.height);
    expect(actionBounds.y + actionBounds.height).toBeGreaterThan(ingredientHeadingBounds.y);
    expect(actionBounds.x + actionBounds.width).toBeCloseTo(
      ingredientRowBounds.x + ingredientRowBounds.width,
    );

    await basilName.scrollIntoViewIfNeeded();
    await expect.poll(async () => (await portions.boundingBox())?.y).toBeLessThan(portionBounds.y);
    await expect.poll(async () => (await recipeHeading.boundingBox())?.y).toBe(headerBounds.y);
    await expect(page.locator("main > .wrapper")).toHaveJSProperty("scrollTop", 0);
  });

  test("scales portions and adds only selected ingredients to the shopping list", async ({
    page,
  }) => {
    await page.goto("/lists");
    await page.getByRole("button", { name: "New list", exact: true }).click();
    await expect(page.getByRole("heading", { name: "Shopping List" })).toBeVisible();
    await page.getByRole("button", { name: "Add item", exact: true }).click();
    const itemInput = page.getByRole("textbox", { name: "Add item", exact: true });
    await itemInput.fill("100 g Tomatoes");
    await itemInput.press("Enter");
    await itemInput.press("Escape");
    await expect(page.getByText("100 g", { exact: true })).toBeVisible();

    await openSavedRecipe(page);
    const portions = page.getByRole("spinbutton", { name: "Portions", exact: true });
    const addToList = page.getByRole("button", { name: "Add to shopping list", exact: true });
    await portions.fill("0");
    await expect(page.getByRole("alert")).toHaveText("Enter a positive whole number of portions.");
    await expect(addToList).toBeDisabled();
    await portions.fill("3");
    await expect(page.getByText("300 g", { exact: true })).toBeVisible();
    await page.getByRole("button", { name: "Decrement", exact: true }).click();
    await expect(portions).toHaveValue("2");
    await expect(page.getByText("200 g", { exact: true })).toBeVisible();
    await addToList.click();

    const transferDialog = page.getByRole("dialog", { name: "Add to shopping list", exact: true });
    const transfer = page.locator('wa-dialog[label="Add to shopping list"]');
    const transferFooter = transfer.locator('[slot="footer"]');
    const confirmTransfer = transferFooter.getByRole("button", {
      name: "Add to shopping list",
      exact: true,
    });
    const onions = transfer.getByRole("checkbox", { name: /Onions/ });
    await expect(onions).toBeChecked();
    const onionLabel = transfer.getByText("Onions", { exact: true });
    await onionLabel.click();
    await expect(onions).not.toBeChecked();

    const destination = transferFooter.getByRole("combobox", { name: "Shopping list" });
    await expect(destination).toHaveValue("Shopping List");
    await confirmTransfer.click();
    await expect(transferDialog).toBeHidden();
    await expect(page.getByRole("status")).toHaveText("Ingredients added to Shopping List.");
    await page.getByRole("link", { name: "Open shopping list" }).click();

    await expect(page.getByText("300 g", { exact: true })).toBeVisible();
    await expect(page.getByRole("checkbox", { name: /Tomatoes/ })).toHaveCount(1);
    await expect(page.getByRole("checkbox", { name: "Basil" })).toBeVisible();
    await expect(page.getByRole("checkbox", { name: /Onions/ })).toHaveCount(0);

    await page.getByRole("button", { name: "Recipes", exact: true }).click();
    await page.getByRole("link", { name: "Tomato sauce" }).click();
    await expect(portions).toHaveValue("4");
    await expect(page.getByText("400 g", { exact: true })).toBeVisible();
  });

  test("keeps the shopping-list selector and confirmation visible while ingredients scroll", async ({
    page,
  }) => {
    expect(await postFixture(page, "/api/lists", { name: "Shopping List" })).toBe(201);
    await openSavedRecipe(page);
    await page.getByRole("spinbutton", { name: "Portions", exact: true }).fill("2");
    await page.getByRole("button", { name: "Add to shopping list", exact: true }).click();

    const transferDialog = page.getByRole("dialog", { name: "Add to shopping list", exact: true });
    const transfer = page.locator('wa-dialog[label="Add to shopping list"]');
    const transferFooter = transfer.locator('[slot="footer"]');
    await expect(transferDialog).toBeVisible();
    await transferDialog.evaluate(async (dialog) => {
      // The opening animation transforms the dialog while its controls are visible.
      await Promise.all(dialog.getAnimations().map((animation) => animation.finished));
    });
    const confirmTransfer = transferFooter.getByRole("button", {
      name: "Add to shopping list",
      exact: true,
    });
    const onionLabel = transfer.getByText("Onions", { exact: true });
    const dialogNameX = (await onionLabel.boundingBox())?.x;
    expect(dialogNameX).toBeGreaterThan(
      (await transfer.getByText("1", { exact: true }).boundingBox())?.x ?? Infinity,
    );
    const basilLabel = transfer.getByText("Basil", { exact: true });
    await expect
      .poll(async () => {
        const [onion, basil] = await Promise.all([
          onionLabel.boundingBox(),
          basilLabel.boundingBox(),
        ]);
        return onion && basil ? onion.x - basil.x : Infinity;
      })
      .toBeCloseTo(0);
    const destination = transferFooter.getByRole("combobox", { name: "Shopping list" });
    await expect(destination).toHaveValue("Shopping List");
    await page.setViewportSize({ width: 390, height: 360 });
    await expectPinned(transfer.locator('[part="body"]'), destination);
    await expect(confirmTransfer).toBeInViewport();
  });

  test("preselects the first shopping list when several exist and can cancel", async ({ page }) => {
    expect(await postFixture(page, "/api/lists", { name: "Shopping List" })).toBe(201);
    expect(await postFixture(page, "/api/lists", { name: "Weekend" })).toBe(201);
    await openSavedRecipe(page);
    await page.getByRole("button", { name: "Add to shopping list", exact: true }).click();

    const transferDialog = page.getByRole("dialog", { name: "Add to shopping list", exact: true });
    const transfer = page.locator('wa-dialog[label="Add to shopping list"]');
    const transferFooter = transfer.locator('[slot="footer"]');
    const destination = transferFooter.getByRole("combobox", { name: "Shopping list" });
    const confirmTransfer = transferFooter.getByRole("button", {
      name: "Add to shopping list",
      exact: true,
    });
    await expect(destination).toHaveValue("Shopping List");
    await expect(confirmTransfer).toBeEnabled();
    await destination.click();
    await page.getByRole("option", { name: "Weekend", exact: true }).click();
    await expect(destination).toHaveValue("Weekend");
    await expect(confirmTransfer).toBeEnabled();
    await transferFooter.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(transferDialog).toBeHidden();
  });

  test("switches the default shopping list from the list actions", async ({ page }) => {
    expect(await postFixture(page, "/api/lists", { name: "Shopping List" })).toBe(201);
    await page.goto("/lists");

    const firstList = page.getByRole("link", { name: /Shopping List/ });
    const weekendList = page.getByRole("link", { name: /Weekend/ });
    const firstBadge = firstList.getByText("Default", { exact: true });
    const weekendBadge = weekendList.getByText("Default", { exact: true });
    const firstActions = page.getByRole("button", { name: "Actions for Shopping List" });
    const setDefault = page.getByRole("menuitem", { name: "Set as default", exact: true });
    await expect(firstBadge).toBeVisible();
    await firstActions.click();
    await expect(page.getByRole("menuitem", { name: "Rename", exact: true })).toBeVisible();
    await expect(setDefault).toHaveCount(0);
    await page.keyboard.press("Escape");

    expect(await postFixture(page, "/api/lists", { name: "Weekend" })).toBe(201);
    await page.reload();
    await expect(firstBadge).toBeVisible();
    await expect(weekendBadge).toHaveCount(0);
    const [firstRowHeight, secondRowHeight] = await page
      .getByRole("listitem")
      .filter({ has: page.getByRole("link") })
      .evaluateAll((rows) => rows.map((row) => row.getBoundingClientRect().height));
    expect(firstRowHeight).toBe(secondRowHeight);
    await page.getByRole("button", { name: "Actions for Weekend" }).click();
    await setDefault.click();
    await expect(weekendBadge).toBeVisible();
    await expect(firstBadge).toHaveCount(0);
    await page.reload();
    await expect(weekendBadge).toBeVisible();

    const listsTab = page.getByRole("navigation").getByRole("button", { name: "Lists" });
    await listsTab.click();
    await expect(page.getByRole("heading", { name: "Weekend" })).toBeVisible();
    await page.getByRole("button", { name: "Back", exact: true }).click();
    await expect(page).toHaveURL(/\/lists$/);
    await expect(firstList).toBeVisible();
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Weekend" })).toBeVisible();

    await openSavedRecipe(page);
    const openTransfer = page.getByRole("button", { name: "Add to shopping list", exact: true });
    const dialog = page.getByRole("dialog", { name: "Add to shopping list", exact: true });
    const footer = page.locator('wa-dialog[label="Add to shopping list"] [slot="footer"]');
    const destination = footer.getByRole("combobox", { name: "Shopping list" });
    await openTransfer.click();
    await expect(destination).toHaveValue("Weekend");
    await destination.click();
    await page.getByRole("option", { name: "Shopping List", exact: true }).click();
    await expect(destination).toHaveValue("Shopping List");
    await footer.getByRole("button", { name: "Add to shopping list", exact: true }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByRole("status")).toHaveText("Ingredients added to Shopping List.");
    await openTransfer.click();
    await expect(destination).toHaveValue("Weekend");
    await footer.getByRole("button", { name: "Cancel", exact: true }).click();
    await expect(dialog).toBeHidden();

    await page.goto("/lists");
    await firstList.click();
    await firstActions.click();
    await setDefault.click();
    await page.getByRole("button", { name: "Back", exact: true }).click();
    await expect(firstBadge).toBeVisible();
  });

  test("edits a saved recipe and keeps the save action pinned on mobile", async ({ page }) => {
    await openSavedRecipe(page);
    const viewRowHeights = await page
      .locator("article section li")
      .evaluateAll((rows) => rows.map((row) => row.getBoundingClientRect().height));
    await page.getByRole("button", { name: "Actions for Tomato sauce" }).click();
    await page.getByRole("menuitem", { name: "Edit recipe" }).click();
    const editTomatoes = page.getByRole("button", { name: "400 g Tomatoes" });
    await expect(editTomatoes).toBeVisible();
    const editRowHeights = await page
      .locator("form li")
      .evaluateAll((rows) => rows.map((row) => row.getBoundingClientRect().height));
    expect(editRowHeights).toEqual(viewRowHeights);
    await editTomatoes.click();
    const editor = page.locator('wa-dialog[label="Edit ingredient"]');
    await expect(editor.getByRole("textbox", { name: "Ingredient name" })).toHaveValue("tomatoes");
    await editor.getByRole("button", { name: "Close", exact: true }).click();
    await expect(editor).toHaveJSProperty("open", false);
    const saveRecipe = page.locator("form > footer").getByRole("button", { name: "Save recipe" });
    await page.setViewportSize({ width: 390, height: 360 });
    await expectPinned(page.locator("form > .fields"), saveRecipe);
    await page.getByRole("textbox", { name: "Recipe name" }).fill("Sunday tomato sauce");
    await saveRecipe.click();
    await page.setViewportSize({ width: 390, height: 800 });
    await expect(page.getByRole("heading", { name: "Sunday tomato sauce" })).toBeVisible();

    await page.goto("/recipes");
    await page.getByRole("button", { name: "Actions for Sunday tomato sauce" }).click();
    await page.getByRole("menuitem", { name: "Edit recipe" }).click();
    await expect(page.getByRole("textbox", { name: "Recipe name" })).toHaveValue(
      "Sunday tomato sauce",
    );
  });

  test("can cancel deletion on the recipe page and delete from the collection", async ({
    page,
  }) => {
    await openSavedRecipe(page);
    const actions = page.getByRole("button", { name: "Actions for Tomato sauce" });
    const deleteAction = page.getByRole("menuitem", { name: "Delete recipe" });
    const confirm = page.locator('wa-dialog[label="Delete recipe?"]');
    await actions.click();
    await deleteAction.click();
    await expect(confirm).toContainText("Tomato sauce");
    await confirm.getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByRole("dialog", { name: "Delete recipe?" })).toBeHidden();
    await expect(page.getByRole("heading", { name: "Tomato sauce" })).toBeVisible();

    await page.goto("/recipes");
    await actions.click();
    await deleteAction.click();
    await expect(confirm).toContainText("Tomato sauce");
    await confirm.getByRole("button", { name: "Delete recipe" }).click();
    await expect(page.getByText(/Save your first recipe/)).toBeVisible();
    await expect(page).toHaveURL(/\/recipes$/);
  });
});
