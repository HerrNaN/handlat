<script lang="ts">
  import WaInput from "@awesome.me/webawesome/dist/components/input/input.js";
  import "@awesome.me/webawesome/dist/components/option/option.js";
  import WaSelect from "@awesome.me/webawesome/dist/components/select/select.js";

  import { language, messages } from "../i18n";
  import { AmountSchema, formatAmount, maxAmountLength, UnitSchema, units } from "../list/quantity";
  import type { RecipeDraft } from "./api";

  type Ingredient = RecipeDraft["ingredients"][number];
  let {
    ingredient,
    onUpdate,
  }: {
    ingredient: Ingredient;
    onUpdate: (changes: Partial<Ingredient>) => void;
  } = $props();

  const name = $derived(ingredient.name);
  const amount = $derived(formatAmount(ingredient.amount, $language) ?? "");
  const unit = $derived(ingredient.unit ?? "none");
  let invalidName = $state(false);
  let invalidAmount = $state(false);

  const changeName = (event: Event) => {
    const value = (event.currentTarget as WaInput).value?.trim() ?? "";
    invalidName = !value || [...value].length > 200;
    if (!invalidName) {
      onUpdate({ name: value });
    }
  };

  const changeAmount = (event: Event) => {
    const value = (event.currentTarget as WaInput).value?.trim();
    const parsed = value ? AmountSchema.safeParse(value) : undefined;
    invalidAmount = parsed?.success === false;
    if (!invalidAmount) {
      onUpdate({ amount: parsed?.data });
    }
  };

  const changeUnit = (event: Event) => {
    const parsed = UnitSchema.safeParse((event.currentTarget as WaSelect).value);
    onUpdate({ unit: parsed.data });
  };
</script>

<section>
  <wa-input
    label={$messages.recipes.ingredientName}
    value={name}
    oninput={() => (invalidName = false)}
    onchange={changeName}
    required
  ></wa-input>
  {#if invalidName}
    <p role="alert">{$messages.recipes.invalidIngredientName}</p>
  {/if}
  <div class="quantity">
    <wa-input
      label={$messages.recipes.amount}
      value={amount}
      inputmode="decimal"
      oninput={() => (invalidAmount = false)}
      onchange={changeAmount}
    ></wa-input>
    <wa-select label={$messages.recipes.unit} value={unit} onchange={changeUnit}>
      <wa-option value="none">{$messages.recipes.noUnit}</wa-option>
      {#each units as unit (unit)}
        <wa-option value={unit}>{$messages.units[unit]}</wa-option>
      {/each}
    </wa-select>
  </div>
  {#if invalidAmount}
    <p role="alert">{$messages.errors.invalidAmount(maxAmountLength)}</p>
  {/if}
</section>

<style>
  section {
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
</style>
