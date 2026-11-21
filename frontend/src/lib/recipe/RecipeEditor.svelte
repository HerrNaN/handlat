<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import WaInput from "@awesome.me/webawesome/dist/components/input/input.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import WaNumberInput from "@awesome.me/webawesome/dist/components/number-input/number-input.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import { createMutation } from "@tanstack/svelte-query";
  import { untrack } from "svelte";

  import "@awesome.me/webawesome/dist/styles/utilities/visually-hidden.css";

  import { ApiError } from "../api";
  import { language, messages } from "../i18n";
  import { formatQuantity, parseItemInput } from "../list/quantity";
  import { saveRecipe, type Recipe, type RecipeDraft } from "./api";
  import EditIngredient from "./EditIngredient.svelte";
  import { parsePortions } from "./scale";

  const maxIngredients = 100;
  type Ingredient = RecipeDraft["ingredients"][number];
  let nextKey = 0;

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
  let rows = $state(
    initial?.ingredients.map((ingredient) => ({ key: nextKey++, ingredient })) ?? [],
  );
  let nameInput = $state<WaInput>();
  let yieldInput = $state<WaNumberInput>();
  let ingredientInput = $state<WaInput>();
  let editDialog = $state<WaDialog>();
  let editingKey = $state<number>();
  const editing = $derived(rows.find(({ key }) => key === editingKey));
  let ingredientError = $state<"invalidIngredientName" | "tooManyIngredients">();
  let invalid = $state(false);

  const save = createMutation(() => ({
    mutationFn: (draft: RecipeDraft) => saveRecipe(id, draft, !initial),
    onSuccess: onSave,
  }));

  const appendIngredient = () => {
    if (!ingredientInput) {
      return true;
    }
    const input = ingredientInput.value?.trim();
    if (!input) {
      return true;
    }
    if (rows.length >= maxIngredients) {
      ingredientError = "tooManyIngredients";
      return false;
    }
    const { text, amount, unit } = parseItemInput(input);
    if ([...text].length > 200) {
      ingredientError = "invalidIngredientName";
      return false;
    }
    rows.push({ key: nextKey++, ingredient: { name: text, amount, unit } });
    ingredientInput.value = "";
    ingredientError = undefined;
    invalid = false;
    return true;
  };

  const addIngredient = () => {
    if (save.isPending) {
      return;
    }
    appendIngredient();
    ingredientInput?.focus();
  };
  const onIngredientKeydown = (event: KeyboardEvent) => {
    if (event.key === "Enter" && !event.isComposing) {
      event.preventDefault();
      addIngredient();
    }
  };

  const submit = (event: SubmitEvent) => {
    event.preventDefault();
    if (save.isPending) {
      return;
    }
    save.reset();
    const name = nameInput?.value?.trim();
    const portions = parsePortions(yieldInput?.value ?? "");
    if (!appendIngredient()) {
      ingredientInput?.focus();
      return;
    }
    invalid = !name || portions === undefined || rows.length === 0;
    if (name && portions !== undefined && rows.length) {
      save.mutate({ name, portions, ingredients: rows.map(({ ingredient }) => ingredient) });
    }
  };

  const openEditor = (key: number) => {
    editingKey = key;
    if (editDialog) {
      editDialog.open = true;
    }
  };
  const closeEditor = () => {
    if (editDialog) {
      editDialog.open = false;
    }
  };
  const updateIngredient = (ingredient: Ingredient) => {
    if (editing) {
      editing.ingredient = ingredient;
    }
    closeEditor();
  };
  const removeIngredient = () => {
    rows = rows.filter(({ key }) => key !== editingKey);
    ingredientError = undefined;
    closeEditor();
  };
</script>

<form onsubmit={submit}>
  <fieldset disabled={save.isPending}>
    <div class="details">
      <wa-input
        bind:this={nameInput}
        value={initial?.name ?? ""}
        label={$messages.recipes.name}
        maxlength={200}
        required
      ></wa-input>
      <wa-number-input
        bind:this={yieldInput}
        value={String(initial?.portions ?? 1)}
        label={$messages.recipes.yield}
        min={1}
        max={Number.MAX_SAFE_INTEGER}
        step={1}
        inputmode="numeric"
        required
      >
        <span slot="end">{$messages.recipes.targetPortions}</span>
      </wa-number-input>
    </div>
    <h2>{$messages.recipes.ingredients}</h2>
    <div class="add">
      <wa-input
        bind:this={ingredientInput}
        enterkeyhint="go"
        placeholder={$messages.items.example}
        oninput={() => (ingredientError = undefined)}
        onkeydown={onIngredientKeydown}
      >
        <span slot="label" class="wa-visually-hidden-force">{$messages.recipes.ingredient}</span>
      </wa-input>
      <wa-button onclick={addIngredient}>
        <wa-icon name="plus" label={$messages.recipes.addIngredient}></wa-icon>
      </wa-button>
    </div>
    {#if ingredientError}
      <p role="alert">{$messages.recipes[ingredientError]}</p>
    {/if}
    {#if rows.length}
      <ul>
        {#each rows as { key, ingredient } (key)}
          <li>
            <button type="button" aria-haspopup="dialog" onclick={() => openEditor(key)}>
              <span class="name">{ingredient.name}</span>
              {#if ingredient.amount || ingredient.unit}
                <span class="quantity">
                  {formatQuantity(
                    ingredient.amount,
                    ingredient.unit ? $messages.units[ingredient.unit] : undefined,
                    $language,
                  )}
                </span>
              {/if}
              <wa-icon name="pen" aria-hidden="true"></wa-icon>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
    {#if !ingredientError && (invalid || (save.error instanceof ApiError && save.error.code === "invalid_recipe"))}
      <p role="alert">{$messages.recipes.invalid}</p>
    {:else if !ingredientError && save.isError}
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

<wa-dialog
  bind:this={editDialog}
  label={$messages.recipes.editIngredient}
  light-dismiss
  onwa-after-hide={(event: Event) => {
    if (event.target === editDialog) editingKey = undefined;
  }}
>
  {#if editing}
    {#key editing.key}
      <EditIngredient
        ingredient={editing.ingredient}
        onSave={updateIngredient}
        onRemove={removeIngredient}
        onCancel={closeEditor}
      />
    {/key}
  {/if}
</wa-dialog>

<style>
  fieldset {
    border: 0;
    padding: 0;
    margin: 0;
    min-inline-size: 0;
  }

  .details,
  .add {
    display: grid;
    gap: var(--wa-space-s);
  }

  wa-number-input {
    max-inline-size: 18rem;
  }

  ul {
    list-style: none;
    padding: 0;
    margin: var(--wa-space-s) 0 0;
  }

  li button {
    appearance: none;
    border: 0;
    border-bottom: 1px solid var(--wa-color-surface-border);
    width: 100%;
    min-block-size: 2.75rem;
    display: flex;
    align-items: center;
    gap: var(--wa-space-s);
    padding: var(--wa-space-xs) var(--wa-space-2xs);
    background: none;
    color: inherit;
    font: inherit;
    text-align: start;
    cursor: pointer;
    overflow-wrap: anywhere;
  }

  li button:hover {
    background: var(--wa-color-surface-raised);
  }

  .name {
    flex: 1;
  }

  .quantity {
    max-inline-size: 45%;
    text-align: end;
  }

  .add {
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: end;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: end;
    gap: var(--wa-space-s);
    margin-block-start: var(--wa-space-xl);
  }
</style>
