<script lang="ts">
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import WaInput from "@awesome.me/webawesome/dist/components/input/input.js";
  import "@awesome.me/webawesome/dist/components/option/option.js";
  import WaSelect from "@awesome.me/webawesome/dist/components/select/select.js";

  import { language, messages } from "../i18n";
  import { AmountSchema, formatAmount, maxAmountLength, UnitSchema, units } from "../list/quantity";
  import type { RecipeDraft } from "./api";

  type Ingredient = RecipeDraft["ingredients"][number];
  let {
    ingredient,
    onSave,
    onRemove,
    onCancel,
  }: {
    ingredient: Ingredient;
    onSave: (value: Ingredient) => void;
    onRemove: () => void;
    onCancel: () => void;
  } = $props();

  let nameInput = $state<WaInput>();
  let amountInput = $state<WaInput>();
  let unitInput = $state<WaSelect>();
  let error = $state<"invalidIngredientName" | "invalidAmount">();

  const submit = (event: SubmitEvent) => {
    event.preventDefault();
    const name = nameInput?.value?.trim();
    if (!name || [...name].length > 200) {
      error = "invalidIngredientName";
      nameInput?.focus();
      return;
    }

    const enteredAmount = amountInput?.value?.trim();
    const amount = enteredAmount ? AmountSchema.safeParse(enteredAmount) : undefined;
    if (amount?.success === false) {
      error = "invalidAmount";
      amountInput?.focus();
      return;
    }

    const unit = UnitSchema.safeParse(unitInput?.value);
    onSave({ name, amount: amount?.data, unit: unit.data });
  };
</script>

<form onsubmit={submit}>
  <wa-input
    bind:this={nameInput}
    label={$messages.recipes.ingredientName}
    value={ingredient.name}
    oninput={() => (error = undefined)}
    required
  ></wa-input>
  <div class="quantity">
    <wa-input
      bind:this={amountInput}
      label={$messages.recipes.amount}
      value={formatAmount(ingredient.amount, $language) ?? ""}
      inputmode="decimal"
      oninput={() => (error = undefined)}
    ></wa-input>
    <wa-select
      bind:this={unitInput}
      label={$messages.recipes.unit}
      value={ingredient.unit ?? "none"}
    >
      <wa-option value="none">{$messages.recipes.noUnit}</wa-option>
      {#each units as unit (unit)}
        <wa-option value={unit}>{$messages.units[unit]}</wa-option>
      {/each}
    </wa-select>
  </div>
  {#if error}
    <p role="alert">
      {error === "invalidAmount"
        ? $messages.errors.invalidAmount(maxAmountLength)
        : $messages.recipes.invalidIngredientName}
    </p>
  {/if}
  <div class="actions">
    <wa-button variant="danger" appearance="plain" onclick={onRemove}>
      {$messages.recipes.removeIngredient}
    </wa-button>
    <wa-button onclick={onCancel}>{$messages.common.cancel}</wa-button>
    <wa-button variant="brand" type="submit">{$messages.common.done}</wa-button>
  </div>
</form>

<style>
  form {
    display: grid;
    gap: var(--wa-space-m);
  }

  .quantity {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--wa-space-s);
  }

  p {
    margin: 0;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: end;
    gap: var(--wa-space-s);
  }

  .actions wa-button:first-child {
    margin-inline-end: auto;
  }
</style>
