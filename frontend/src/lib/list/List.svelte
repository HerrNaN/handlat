<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/skeleton/skeleton.js";
  import { createMutation, createQuery, useMutationState } from "@tanstack/svelte-query";

  import { route, navigate } from "../../router";
  import { session } from "../user/session";
  import AddItem from "./AddItem.svelte";
  import { errorMessage } from "./api";
  import EditItem from "./EditItem.svelte";
  import emptyShoppingList from "./empty-shopping-list.svg";
  import EmptyState from "./EmptyState.svelte";
  import ItemList from "./ItemList.svelte";
  import { type Item } from "./list";
  import ListActions from "./ListActions.svelte";
  import ListSkeleton from "./ListSkeleton.svelte";
  import {
    addItemMutation,
    checkItemMutation,
    clearCheckedMutation,
    deleteItemMutation,
    dismissListErrors,
    listQuery,
    updateItemMutation,
  } from "./queries";

  const target = $derived({ account: $session?.id ?? "", listID: route.params.id ?? "" });
  const query = createQuery(() => listQuery($session?.id, target.listID, () => navigate("/lists")));
  const loading = $derived(query.isPending && !query.data);
  const addition = createMutation(() => addItemMutation());
  const update = createMutation(() => updateItemMutation());
  const check = createMutation(() => checkItemMutation());
  const remove = createMutation(() => deleteItemMutation());
  const clear = createMutation(() => clearCheckedMutation());
  const mutations = [addition, update, check, remove, clear];
  const writes = useMutationState({
    filters: { mutationKey: ["private-lists"] },
    select: (mutation) => ({
      status: mutation.state.status,
      error: mutation.state.error,
      variables: mutation.state.variables as
        | { account: string; listID?: string; itemID?: string }
        | undefined,
    }),
  });
  const listWrites = $derived(
    writes.filter(
      (write) =>
        write.variables?.account === target.account && write.variables.listID === target.listID,
    ),
  );
  const pending = $derived(listWrites.filter((write) => write.status === "pending"));
  const pendingItems = $derived(
    new Set(pending.map((write) => write.variables?.itemID).filter((id) => id !== undefined)),
  );
  const error = $derived(
    listWrites.findLast((write) => write.status === "error")?.error ?? query.error,
  );
  const refresh = async () => {
    if (pending.length) return;
    mutations.forEach((mutation) => mutation.reset());
    dismissListErrors();
    void query.refetch();
  };
  let editItemDialog = $state<WaDialog>();
  let itemToEditID = $state("");
  const itemToEdit = $derived(query.data?.items.find((item) => item.id === itemToEditID));
  const onClickEditItem = (item: Item) => {
    if (editItemDialog) editItemDialog.open = true;
    itemToEditID = item.id;
  };
</script>

<article>
  <header>
    <wa-button appearance="plain" onclick={() => navigate("/lists")}>
      <wa-icon name="arrow-left" label="Back"></wa-icon>
    </wa-button>
    <h2>
      {#if loading}
        <wa-skeleton class="title" effect="none" aria-hidden="true"></wa-skeleton>
      {:else}
        {query.data?.name ?? ""}
      {/if}
    </h2>
    {#if query.data}
      <ListActions list={query.data} onleave={() => navigate("/lists")} />
    {:else if loading}
      <wa-button class="loading-actions" appearance="plain" disabled aria-label="List actions">
        <wa-icon name="ellipsis-vertical"></wa-icon>
      </wa-button>
    {/if}
  </header>
  {#if error}
    <aside role="alert">
      <p>{errorMessage(error)}</p>
      <wa-button onclick={refresh}>Refresh</wa-button>
    </aside>
  {/if}
  <div class="list-area" aria-busy={loading}>
    {#if loading}
      <ListSkeleton />
    {:else if query.data}
      {#key query.data.id}
        {#if query.data.items.length === 0 && !error}
          <EmptyState
            illustration={emptyShoppingList}
            heading="What do you need to buy?"
            description="Add your first item using the plus button."
          />
        {/if}
        <ItemList
          items={query.data.items}
          {onClickEditItem}
          {pendingItems}
          clearing={clear.isPending}
          onCheck={(itemID, checked) => check.mutate({ ...target, itemID, checked })}
          onDelete={(itemID) => remove.mutateAsync({ ...target, itemID })}
          onClear={() => clear.mutate({ ...target })}
        />
        <AddItem {error} onAdd={(item) => addition.mutateAsync({ ...target, item })} />
      {/key}
    {/if}
  </div>
</article>

<wa-dialog bind:this={editItemDialog} label="Edit Item" light-dismiss>
  {#if error}
    <aside role="alert">
      <p>{errorMessage(error)}</p>
      <wa-button onclick={refresh}>Refresh</wa-button>
    </aside>
  {/if}
  {#if itemToEdit}
    <EditItem
      item={itemToEdit}
      onUpdate={(itemID, changes) => update.mutate({ ...target, itemID, changes })}
    />
  {/if}
</wa-dialog>

<style>
  aside,
  aside p {
    margin: var(--wa-space-s) var(--wa-space-m);
  }

  article {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: var(--wa-color-surface-default);
    overflow-y: hidden;
  }

  .list-area {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
    background-color: var(--wa-color-surface-lowered);
  }

  header {
    height: 2rem;
    flex-shrink: 0;
    z-index: 1;
    position: sticky;
    top: 0;
    display: flex;
    align-items: center;
    padding: var(--wa-space-m);
    background-color: var(--wa-color-surface-raised);
    box-shadow: 0 var(--wa-shadow-offset-x-m) var(--wa-shadow-offset-y-m) var(--wa-color-shadow);
  }

  h2 {
    flex: 1;
    min-width: 0;
    margin: 0;
    margin-inline-start: var(--wa-space-xs);
    font-size: var(--wa-font-size-xl);
    font-weight: var(--wa-font-weight-normal);
    color: var(--wa-color-text-normal);
  }

  .title {
    display: block;
    width: min(12rem, 80%);
    height: 1.5rem;
  }

  .title,
  .loading-actions {
    visibility: hidden;
    animation: show-loading 0s 200ms forwards;
  }

  @keyframes show-loading {
    to {
      visibility: visible;
    }
  }
</style>
