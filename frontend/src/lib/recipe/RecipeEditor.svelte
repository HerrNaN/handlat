<script lang="ts">
  import { createMutation } from "@tanstack/svelte-query";
  import { untrack } from "svelte";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  import { ApiError } from "../api";
  import { language, messages } from "../i18n";
  import { AmountSchema, formatAmount, maxAmountLength, type Unit, units } from "../list/quantity";
  import { saveRecipe, type Recipe, type RecipeDraft } from "./api";
  import { parsePortions } from "./scale";

  const maxIngredients = 100;
  type IngredientRow = { key: number; name: string; amount: string; unit: Unit | "" };
  let nextKey = 0;
  const blankRow = (): IngredientRow => ({ key: nextKey++, name: "", amount: "", unit: "" });

  let {
    recipe,
    onSave,
    onCancel,
  }: {
    recipe?: Recipe;
    onSave: (value: Recipe) => void;
    onCancel: () => void;
  } = $props();

  // The draft is seeded once so background refetches cannot overwrite unsaved edits.
  const initial = untrack(() => recipe);
  const id = initial?.id ?? crypto.randomUUID();
  let name = $state(initial?.name ?? "");
  let portions = $state(String(initial?.portions ?? 1));
  let rows = $state<IngredientRow[]>(
    initial?.ingredients.map((item) => ({
      key: nextKey++,
      name: item.name,
      amount: formatAmount(item.amount, $language) ?? "",
      unit: item.unit ?? "",
    })) ?? [blankRow()],
  );

  let invalid = $state(false);
  const save = createMutation(() => ({
    mutationFn: (draft: RecipeDraft) => saveRecipe(id, draft, !initial),
    onSuccess: onSave,
  }));
  const submit = (event: SubmitEvent) => {
    event.preventDefault();
    if (save.isPending) return;
    save.reset();
    const count = parsePortions(portions);
    const ingredients: RecipeDraft["ingredients"] = [];
    for (const row of rows) {
      const amount = row.amount.trim() ? AmountSchema.safeParse(row.amount) : undefined;
      if (!row.name.trim() || amount?.success === false) {
        invalid = true;
        return;
      }
      ingredients.push({
        name: row.name.trim(),
        amount: amount?.data,
        unit: row.unit || undefined,
      });
    }
    invalid = !name.trim() || count === undefined;
    if (!invalid && count !== undefined)
      save.mutate({ name: name.trim(), portions: count, ingredients });
  };
</script>

<form onsubmit={submit}>
  <fieldset disabled={save.isPending}>
    <div class="details">
      <label>
        {$messages.recipes.name}
        <input bind:value={name} required maxlength="200" />
      </label>
      <label>
        {$messages.recipes.basePortions}
        <input bind:value={portions} required inputmode="numeric" pattern="[0-9]+" />
      </label>
    </div>
    <h2>{$messages.recipes.ingredients}</h2>
    <ol>
      {#each rows as row, index (row.key)}
        <li>
          <fieldset class="ingredient">
            <legend>{$messages.recipes.ingredient(index + 1)}</legend>
            <label class="name">
              {$messages.recipes.ingredientName}
              <input bind:value={row.name} required maxlength="200" />
            </label>
            <label>
              {$messages.recipes.amount}
              <input bind:value={row.amount} inputmode="decimal" maxlength={maxAmountLength} />
            </label>
            <label>
              {$messages.recipes.unit}
              <select bind:value={row.unit}>
                <option value="">{$messages.recipes.noUnit}</option>
                {#each units as unit (unit)}
                  <option value={unit}>{$messages.units[unit]}</option>
                {/each}
              </select>
            </label>
            <wa-button
              appearance="plain"
              disabled={rows.length === 1}
              onclick={() => (rows = rows.filter((item) => item.key !== row.key))}
            >
              <wa-icon name="trash" label={$messages.recipes.removeIngredient(index + 1)}></wa-icon>
            </wa-button>
          </fieldset>
        </li>
      {/each}
    </ol>
    <wa-button disabled={rows.length >= maxIngredients} onclick={() => rows.push(blankRow())}>
      <wa-icon slot="start" name="plus"></wa-icon>
      {$messages.recipes.addIngredient}
    </wa-button>
    {#if invalid || (save.error instanceof ApiError && save.error.code === "invalid_recipe")}
      <p role="alert">{$messages.recipes.invalid}</p>
    {:else if save.isError}
      <p role="alert">{$messages.recipes.saveFailed}</p>
    {/if}
    <div class="actions">
      <wa-button onclick={onCancel}>{$messages.common.cancel}</wa-button>
      <wa-button variant="brand" type="submit" loading={save.isPending}>
        {$messages.recipes.save}
      </wa-button>
    </div>
  </fieldset>
</form>

<style>
  fieldset {
    border: 0;
    padding: 0;
    margin: 0;
    min-inline-size: 0;
  }

  .details,
  .ingredient {
    display: grid;
    gap: var(--wa-space-s);
  }

  .ingredient {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
    align-items: end;
    padding: var(--wa-space-s);
    border: 1px solid var(--wa-color-surface-border);
    border-radius: var(--wa-border-radius-m);
  }

  .name {
    grid-column: 1 / -1;
  }

  ol {
    display: grid;
    gap: var(--wa-space-s);
    list-style: none;
    margin: 0 0 var(--wa-space-m);
    padding: 0;
  }

  label {
    display: grid;
    gap: var(--wa-space-2xs);
  }

  input,
  select {
    box-sizing: border-box;
    width: 100%;
    min-block-size: 2.75rem;
    padding-inline: var(--wa-space-s);
    border: 1px solid var(--wa-color-surface-border);
    border-radius: var(--wa-border-radius-m);
    background: var(--wa-color-surface-default);
    color: var(--wa-color-text-normal);
    font: inherit;
  }

  .details input[inputmode="numeric"] {
    max-inline-size: 12rem;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: end;
    gap: var(--wa-space-s);
    margin-block-start: var(--wa-space-xl);
  }
</style>
