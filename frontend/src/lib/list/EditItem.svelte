<script lang="ts">
  import "@awesome.me/webawesome/dist/components/select/select.js";
  import "@awesome.me/webawesome/dist/components/option/option.js";
  import "@awesome.me/webawesome/dist/components/input/input.js";
  import type WaInput from "@awesome.me/webawesome/dist/components/input/input.js";

  import { type ChangeEvent } from "../utils/events";
  import type { ItemChanges } from "./api";
  import { departments } from "./classifier";
  import type { Item } from "./list";
  import { AmountSchema, UnitSchema, units } from "./quantity";

  type Props = {
    item: Item;
    onUpdate: (itemID: string, changes: ItemChanges) => void;
  };

  let { item, onUpdate }: Props = $props();

  // Only update a control when its own server value changes. An unrelated save
  // must not overwrite text currently being entered in another field.
  const text = $derived(item.text);
  const amount = $derived(item.amount?.replace(".", ",") ?? "");
  const unit = $derived(item.unit ?? "none");
  const department = $derived(item.department);

  const onChangeAmount = (event: Event) => {
    const input = event.currentTarget as WaInput;
    if (!input.value?.trim()) {
      input.setCustomValidity("");
      onUpdate(item.id, { amount: null });
      return;
    }
    const amount = AmountSchema.safeParse(input.value);
    input.setCustomValidity(
      amount.success
        ? ""
        : "Enter a positive amount using a comma or dot (at most 100 characters).",
    );
    if (amount.success) onUpdate(item.id, { amount: amount.data });
    else input.reportValidity();
  };

  const onChangeUnit = (event: ChangeEvent) =>
    onUpdate(item.id, {
      unit: event.target.value === "none" ? null : UnitSchema.parse(event.target.value),
    });

  const onChangeName = (e: ChangeEvent) => onUpdate(item.id, { text: e.target.value });

  const onChangeDepartement = (e: ChangeEvent) => onUpdate(item.id, { department: e.target.value });
</script>

<section>
  <wa-input value={text} label="Name" onchange={onChangeName}></wa-input>
  <div>
    <wa-input
      value={amount}
      label="Amount"
      inputmode="decimal"
      onchange={onChangeAmount}
      oninput={(event: Event) => (event.currentTarget as WaInput).setCustomValidity("")}
    ></wa-input>
    <wa-select label="Unit" value={unit} onchange={onChangeUnit}>
      <wa-option value="none">No unit</wa-option>
      {#each units as unit (unit)}
        <wa-option value={unit}>{unit}</wa-option>
      {/each}
    </wa-select>
  </div>
  <wa-select label="Departement" value={department} onchange={onChangeDepartement}>
    {#each departments as dep (dep.id)}
      <wa-option value={dep.id}>
        {dep.name}
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

  section {
    display: flex;
    flex-direction: column;
    gap: var(--wa-space-m);
  }
</style>
