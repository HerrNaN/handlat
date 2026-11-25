<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import WaNumberInput from "@awesome.me/webawesome/dist/components/number-input/number-input.js";
  import WaSelect from "@awesome.me/webawesome/dist/components/select/select.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/checkbox/checkbox.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/option/option.js";
  import { createMutation, createQuery } from "@tanstack/svelte-query";
  import { untrack } from "svelte";

  import { p } from "../../router";
  import { language, messages } from "../i18n";
  import { addItems, type ListSummary, type NewItem, type ShoppingList } from "../list/api";
  import { classify, departments } from "../list/classifier";
  import { formatQuantity, maxAmountLength } from "../list/quantity";
  import { listKey, summariesKey, summariesQuery } from "../list/queries";
  import { queryClient } from "../query";
  import { session } from "../user/session";
  import type { Recipe } from "./api";
  import { formatIngredientName } from "./ingredientName";
  import { parsePortions, scaleAmount } from "./scale";

  let { recipe, account }: { recipe: Recipe; account: string } = $props();
  const base = $derived(recipe.portions);
  let portions = $state(untrack(() => String(base)));
  let dialog = $state<WaDialog>();
  let reviewing = $state(false);
  let excluded = $state<number[]>([]);
  let destinationID = $state("");
  let addedTo = $state<ShoppingList>();
  const target = $derived(parsePortions(portions));
  const scaled = $derived(
    recipe.ingredients.map((ingredient, index) => {
      const scaled =
        ingredient.amount && target ? scaleAmount(ingredient.amount, target, base) : undefined;
      const quantity = formatQuantity(
        scaled?.amount ?? ingredient.amount,
        ingredient.unit ? $messages.units[ingredient.unit] : undefined,
        $language,
      );
      return {
        ...ingredient,
        index,
        amount: scaled?.amount ?? ingredient.amount,
        label: scaled?.approximate ? `≈${quantity}` : quantity,
        approximate: scaled?.approximate,
        tooPrecise: (scaled?.amount.length ?? 0) > maxAmountLength,
      };
    }),
  );
  const selected = $derived(scaled.filter(({ index }) => !excluded.includes(index)));
  const lists = createQuery(() => ({ ...summariesQuery(account), enabled: reviewing }));
  const destination = $derived(
    lists.data?.length === 1 ? lists.data[0] : lists.data?.find(({ id }) => id === destinationID),
  );
  const canTransfer = $derived(
    Boolean(
      target && destination && selected.length && !selected.some(({ tooPrecise }) => tooPrecise),
    ),
  );
  let attempt: { key: string; items: NewItem[] } | undefined;
  const transfer = createMutation(() => ({
    mutationFn: ({ list, items }: { list: ListSummary; items: NewItem[] }) =>
      addItems(list.id, items),
    onSuccess: (list) => {
      if (account !== $session?.id) {
        return;
      }
      queryClient.setQueryData(listKey(account, list.id), list);
      void queryClient.invalidateQueries({ queryKey: summariesKey(account) });
      addedTo = list;
      attempt = undefined;
      if (dialog) {
        dialog.open = false;
      }
    },
  }));

  const review = () => {
    reviewing = true;
    excluded = [];
    addedTo = undefined;
    transfer.reset();
    if (dialog) {
      dialog.open = true;
    }
  };
  const toggle = (index: number) => {
    excluded = excluded.includes(index)
      ? excluded.filter((number) => number !== index)
      : [...excluded, index];
  };
  const submit = () => {
    if (!canTransfer || !destination || transfer.isPending) {
      return;
    }
    const items = selected.map(({ name, amount, unit }) => ({
      text: name,
      amount,
      unit,
      department: classify(name, departments).id,
    }));
    const key = JSON.stringify([destination.id, items]);
    // Reuse item IDs after a failed response in case the server already committed the transfer.
    if (attempt?.key !== key) {
      attempt = { key, items: items.map((item) => ({ ...item, id: crypto.randomUUID() })) };
    }
    transfer.mutate({ list: destination, items: attempt.items });
  };
</script>

<wa-number-input
  value={portions}
  label={$messages.recipes.targetPortions}
  min={1}
  max={Number.MAX_SAFE_INTEGER}
  step={1}
  inputmode="numeric"
  aria-invalid={target === undefined}
  disabled={transfer.isPending}
  oninput={(event: Event) => {
    portions = (event.currentTarget as WaNumberInput).value ?? "";
  }}
></wa-number-input>

<div class="ingredient-heading">
  <h2>{$messages.recipes.ingredients}</h2>
  <wa-button variant="brand" size="s" disabled={target === undefined} onclick={review}>
    <wa-icon slot="start" name="cart-plus" aria-hidden="true"></wa-icon>
    {$messages.recipes.addToList}
  </wa-button>
</div>

{#if addedTo}
  <p role="status">{$messages.recipes.added(addedTo.name)}</p>
  <wa-button href={p("/lists/:id", { params: { id: addedTo.id } })}>
    {$messages.recipes.openList}
  </wa-button>
{/if}

{@render ingredientList(false)}

{#if target === undefined}
  <p role="alert">{$messages.recipes.invalidPortions}</p>
{/if}
{@render quantityNotes()}

{#snippet ingredientList(selectable: boolean)}
  <ul class:selectable>
    {#each scaled as item (item.index)}
      <li>
        {#if selectable}
          <wa-checkbox
            class="choice"
            checked={!excluded.includes(item.index)}
            onchange={() => toggle(item.index)}
            disabled={transfer.isPending}
          >
            {@render ingredient(item)}
          </wa-checkbox>
        {:else}
          <div class="choice">{@render ingredient(item)}</div>
        {/if}
      </li>
    {/each}
  </ul>
{/snippet}

{#snippet ingredient(item: (typeof scaled)[number])}
  <span>{item.label}{item.tooPrecise ? " *" : ""}</span>
  <span>{formatIngredientName(item.name)}</span>
{/snippet}

{#snippet quantityNotes()}
  {#if scaled.some(({ approximate }) => approximate)}
    <p>{$messages.recipes.rounded}</p>
  {/if}
  {#if scaled.some(({ tooPrecise }) => tooPrecise)}
    <p>{$messages.recipes.tooPrecise}</p>
  {/if}
{/snippet}

<wa-dialog
  bind:this={dialog}
  label={$messages.recipes.addToList}
  light-dismiss
  onwa-hide={(event: Event) => {
    if (event.target === dialog && transfer.isPending && !addedTo) {
      event.preventDefault();
    }
  }}
  onwa-after-hide={(event: Event) => {
    if (event.target === dialog) {
      reviewing = false;
    }
  }}
>
  {#if reviewing}
    {@render ingredientList(true)}
    {@render quantityNotes()}
  {/if}
  <div slot="footer" class="transfer-footer">
    {#if reviewing}
      {#if lists.isPending}
        <p role="status">{$messages.lists.loading}</p>
      {:else if lists.isError}
        <p role="alert">{$messages.errors.loadLists}</p>
        <wa-button onclick={() => lists.refetch()}>{$messages.common.tryAgain}</wa-button>
      {:else if lists.data?.length}
        <wa-select
          label={$messages.recipes.destination}
          placeholder={$messages.recipes.chooseList}
          value={destination?.id ?? ""}
          disabled={transfer.isPending}
          onchange={(event: Event) => {
            const value = (event.currentTarget as WaSelect).value;
            destinationID = typeof value === "string" ? value : "";
          }}
        >
          {#each lists.data as list (list.id)}
            <wa-option value={list.id}>{list.name}</wa-option>
          {/each}
        </wa-select>
        {#if destination && destination.memberCount > 1}
          <p>{$messages.recipes.sharedList}</p>
        {/if}
      {:else}
        <p>{$messages.recipes.noLists}</p>
        <wa-button href="/lists">{$messages.lists.title}</wa-button>
      {/if}
      {#if transfer.isError}
        <p role="alert">{$messages.recipes.transferFailed}</p>
      {/if}
    {/if}
    <div class="actions">
      <wa-button data-dialog="close" disabled={transfer.isPending}>
        {$messages.common.cancel}
      </wa-button>
      <wa-button
        variant="brand"
        disabled={!canTransfer}
        loading={transfer.isPending}
        onclick={submit}
      >
        {$messages.recipes.addToList}
      </wa-button>
    </div>
  </div>
</wa-dialog>

<style>
  .ingredient-heading {
    display: flex;
    align-items: center;
    gap: var(--wa-space-xs);
    margin-block: var(--wa-space-m) var(--wa-space-s);
  }

  .ingredient-heading h2 {
    margin: 0;
    font-size: var(--wa-font-size-l);
  }

  .ingredient-heading wa-button {
    min-inline-size: 0;
    margin-inline-start: auto;

    &::part(label) {
      white-space: normal;
    }
  }

  wa-number-input {
    inline-size: min(12rem, 100%);
  }

  wa-dialog::part(footer) {
    padding-block-start: var(--wa-space-s);
  }

  .transfer-footer {
    display: grid;
    gap: var(--wa-space-m);
    inline-size: 100%;
  }

  .transfer-footer p {
    margin: 0;
  }

  .transfer-footer > wa-button {
    justify-self: start;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: end;
    gap: var(--wa-space-xs);
  }

  ul {
    display: grid;
    grid-template-columns: fit-content(45%) minmax(0, 1fr) var(--ingredient-row-height);
    column-gap: var(--wa-space-s);
    list-style: none;
    padding: 0;
    margin: 0;
  }

  ul.selectable {
    grid-template-columns: auto fit-content(45%) minmax(0, 1fr);
  }

  li,
  .choice {
    display: grid;
    grid-column: 1 / -1;
    grid-template-columns: subgrid;
  }

  .choice {
    align-items: center;
    overflow-wrap: anywhere;
  }

  li {
    border-bottom: 1px solid var(--wa-color-surface-border);
  }

  div.choice {
    box-sizing: border-box;
    min-block-size: var(--ingredient-row-height);
    padding-block: var(--wa-space-2xs);
  }

  wa-checkbox.choice {
    min-block-size: 2.75rem;
    padding-block: var(--wa-space-xs);
  }

  wa-checkbox::part(checkbox) {
    display: grid;
    grid-column: 1 / -1;
    grid-template-columns: subgrid;
    align-items: center;
  }

  wa-checkbox::part(control) {
    margin-inline-end: 0;
  }

  wa-checkbox::part(label) {
    display: contents;
  }
</style>
