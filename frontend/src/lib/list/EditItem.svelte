<script lang="ts">
  import "@awesome.me/webawesome/dist/components/select/select.js";
  import "@awesome.me/webawesome/dist/components/option/option.js";
  import "@awesome.me/webawesome/dist/components/input/input.js";
  import type WaInput from "@awesome.me/webawesome/dist/components/input/input.js";

  import { language, messages } from "../i18n";
  import { type ChangeEvent } from "../utils/events";
  import type { ItemChanges } from "./api";
  import { departments } from "./classifier";
  import type { Item } from "./list";
  import { AmountSchema, formatAmount, maxAmountLength, UnitSchema, units } from "./quantity";

  type Props = {
    item: Item;
    onUpdate: (itemID: string, changes: ItemChanges) => void;
  };

  let { item, onUpdate }: Props = $props();

  // Only update a control when its own server value changes. An unrelated save
  // must not overwrite text currently being entered in another field.
  const text = $derived(item.text);
  let amountDraft = $state<{ itemID: string; savedAmount: string | undefined; value: string }>();
  let invalidAmount = $state(false);
  let amountInput = $state<WaInput>();
  const currentDraft = $derived(
    amountDraft?.itemID === item.id && amountDraft.savedAmount === item.amount
      ? amountDraft
      : undefined,
  );
  const amount = $derived(currentDraft?.value ?? formatAmount(item.amount, $language) ?? "");
  const amountError = $derived(
    currentDraft && invalidAmount ? $messages.errors.invalidAmount(maxAmountLength) : "",
  );
  $effect(() => {
    amountInput?.setCustomValidity(amountError);
  });
  const unit = $derived(item.unit ?? "none");
  const department = $derived(item.department);

  const onInputAmount = (event: Event) => {
    const input = event.currentTarget as WaInput;
    amountDraft = { itemID: item.id, savedAmount: item.amount, value: input.value ?? "" };
    invalidAmount = false;
  };

  const onChangeAmount = (event: Event) => {
    const input = event.currentTarget as WaInput;
    if (!input.value?.trim()) {
      invalidAmount = false;
      onUpdate(item.id, { amount: null });
      return;
    }
    const amount = AmountSchema.safeParse(input.value);
    invalidAmount = !amount.success;
    if (amount.success) onUpdate(item.id, { amount: amount.data });
  };

  const onChangeUnit = (event: ChangeEvent) =>
    onUpdate(item.id, {
      unit: event.target.value === "none" ? null : UnitSchema.parse(event.target.value),
    });

  const onChangeName = (e: ChangeEvent) => onUpdate(item.id, { text: e.target.value });

  const onChangeDepartment = (e: ChangeEvent) => onUpdate(item.id, { department: e.target.value });
</script>

<section>
  <wa-input value={text} label={$messages.items.name} onchange={onChangeName}></wa-input>
  <div>
    <wa-input
      bind:this={amountInput}
      value={amount}
      label={$messages.items.amount}
      inputmode="decimal"
      onchange={onChangeAmount}
      oninput={onInputAmount}
    ></wa-input>
    <wa-select label={$messages.items.unit} value={unit} onchange={onChangeUnit}>
      <wa-option value="none">–</wa-option>
      {#each units as unit (unit)}
        <wa-option value={unit}>{$messages.units[unit]}</wa-option>
      {/each}
    </wa-select>
  </div>
  {#if amountError}<p role="alert">{amountError}</p>{/if}
  <wa-select label={$messages.items.department} value={department} onchange={onChangeDepartment}>
    {#each departments as dep (dep.id)}
      <wa-option value={dep.id}>
        {$messages.departments[dep.id]}
      </wa-option>
    {/each}
  </wa-select>
</section>

<style>
  div {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: var(--wa-space-m);
  }

  p {
    margin: 0;
  }

  section {
    display: flex;
    flex-direction: column;
    gap: var(--wa-space-m);
  }
</style>
