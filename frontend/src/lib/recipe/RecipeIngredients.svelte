<script lang="ts">
  import WaNumberInput from "@awesome.me/webawesome/dist/components/number-input/number-input.js";
  import { createMutation, createQuery } from "@tanstack/svelte-query";
  import "@awesome.me/webawesome/dist/components/button/button.js";
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
  import { parsePortions, scaleAmount } from "./scale";

  let { recipe, account }: { recipe: Recipe; account: string } = $props();
  const base = $derived(recipe.portions);
  let portions = $state(untrack(() => String(base)));
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
  const destination = $derived(lists.data?.find(({ id }) => id === destinationID));
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
      if (account !== $session?.id) return;
      queryClient.setQueryData(listKey(account, list.id), list);
      void queryClient.invalidateQueries({ queryKey: summariesKey(account) });
      addedTo = list;
      reviewing = false;
      attempt = undefined;
    },
  }));

  const review = () => {
    reviewing = true;
    excluded = [];
    addedTo = undefined;
    transfer.reset();
  };
  const toggle = (index: number) => {
    excluded = excluded.includes(index)
      ? excluded.filter((number) => number !== index)
      : [...excluded, index];
  };
  const submit = () => {
    if (!canTransfer || !destination || transfer.isPending) return;
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

<h2>{reviewing ? $messages.recipes.review : $messages.recipes.ingredients}</h2>
<ul>
  {#each scaled as item (item.index)}
    <li>
      {#if reviewing}
        <label class="choice">
          <input
            type="checkbox"
            checked={!excluded.includes(item.index)}
            onchange={() => toggle(item.index)}
            disabled={transfer.isPending}
          />
          {@render ingredient(item)}
        </label>
      {:else}
        <div class="choice">{@render ingredient(item)}</div>
      {/if}
    </li>
  {/each}
</ul>

{#snippet ingredient(item: (typeof scaled)[number])}
  <span class="name">{item.name}</span>
  {#if item.label}
    <span>{item.label}{item.tooPrecise ? " *" : ""}</span>
  {/if}
{/snippet}

{#if target === undefined}
  <p role="alert">{$messages.recipes.invalidPortions}</p>
{/if}
{#if scaled.some(({ approximate }) => approximate)}
  <p>{$messages.recipes.rounded}</p>
{/if}
{#if scaled.some(({ tooPrecise }) => tooPrecise)}
  <p>{$messages.recipes.tooPrecise}</p>
{/if}

{#if reviewing}
  {#if lists.isPending}
    <p role="status">{$messages.lists.loading}</p>
  {:else if lists.isError}
    <p role="alert">{$messages.errors.loadLists}</p>
    <wa-button onclick={() => lists.refetch()}>{$messages.common.tryAgain}</wa-button>
  {:else if lists.data?.length}
    <label class="destination">
      {$messages.recipes.destination}
      <select bind:value={destinationID} disabled={transfer.isPending}>
        <option value="">{$messages.recipes.chooseList}</option>
        {#each lists.data as list (list.id)}
          <option value={list.id}>{list.name}</option>
        {/each}
      </select>
    </label>
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
  <div class="actions">
    <wa-button disabled={transfer.isPending} onclick={() => (reviewing = false)}>
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
{:else}
  {#if addedTo}
    <p role="status">{$messages.recipes.added(addedTo.name)}</p>
    <wa-button href={p("/lists/:id", { params: { id: addedTo.id } })}>
      {$messages.recipes.openList}
    </wa-button>
  {/if}
  <div class="actions">
    <wa-button variant="brand" disabled={target === undefined} onclick={review}>
      {$messages.recipes.addToList}
    </wa-button>
  </div>
{/if}

<style>
  wa-number-input {
    max-inline-size: 12rem;
  }

  .destination {
    display: grid;
    gap: var(--wa-space-2xs);
    margin-block-start: var(--wa-space-l);
  }

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

  ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .choice {
    display: flex;
    align-items: center;
    min-block-size: 2.75rem;
    padding-block: var(--wa-space-xs);
    gap: var(--wa-space-s);
    border-bottom: 1px solid var(--wa-color-surface-border);
    overflow-wrap: anywhere;
  }

  input[type="checkbox"] {
    width: 1.25rem;
    height: 1.25rem;
    flex-shrink: 0;
    accent-color: var(--wa-color-brand-fill-loud);
  }

  .name {
    flex: 1;
  }

  .choice span:last-child:not(.name) {
    max-inline-size: 50%;
    text-align: end;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: end;
    gap: var(--wa-space-s);
    margin-block-start: var(--wa-space-l);
  }
</style>
