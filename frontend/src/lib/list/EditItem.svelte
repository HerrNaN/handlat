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
    busy: boolean;
    onUpdate: (itemID: string, changes: ItemChanges) => void;
  };

  let { item, busy, onUpdate }: Props = $props();

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
  <wa-input value={item.text} label="Name" onchange={onChangeName} disabled={busy}></wa-input>
  <div>
    <wa-input
      value={item.amount?.replace(".", ",") ?? ""}
      label="Amount"
      inputmode="decimal"
      onchange={onChangeAmount}
      oninput={(event: Event) => (event.currentTarget as WaInput).setCustomValidity("")}
      disabled={busy}
    ></wa-input>
    <wa-select label="Unit" value={item.unit ?? "none"} onchange={onChangeUnit} disabled={busy}>
      <wa-option value="none">No unit</wa-option>
      {#each units as unit (unit)}
        <wa-option value={unit}>{unit}</wa-option>
      {/each}
    </wa-select>
  </div>
  <wa-select
    label="Departement"
    value={item.department}
    onchange={onChangeDepartement}
    disabled={busy}
  >
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
