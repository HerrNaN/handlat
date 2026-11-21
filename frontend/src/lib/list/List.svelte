<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import { createMutation, createQuery, useMutationState } from "@tanstack/svelte-query";
  import { onMount } from "svelte";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/skeleton/skeleton.js";

  import { route, navigate } from "../../router";
  import PageHeader from "../components/PageHeader.svelte";
  import { messages } from "../i18n";
  import { errorMessage } from "../i18n/errors";
  import { session } from "../user/session";
  import AddItem from "./AddItem.svelte";
  import EditItem from "./EditItem.svelte";
  import EmptyShoppingList from "./EmptyShoppingList.svelte";
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
  const query = createQuery(() =>
    listQuery($session?.id, target.listID, () => {
      void navigate("/lists");
    }),
  );
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
  const writeError = $derived(listWrites.findLast((write) => write.status === "error")?.error);
  const error = $derived(writeError ?? query.error);
  const displayedError = $derived(
    errorMessage(
      error,
      $messages,
      writeError ? $messages.errors.saveList : $messages.errors.loadList,
    ),
  );
  const refresh = async () => {
    if (pending.length) {
      return;
    }
    mutations.forEach((mutation) => mutation.reset());
    dismissListErrors();
    void query.refetch();
  };
  let article = $state<HTMLElement>();
  let emptyCenter = $state(0);
  onMount(() => {
    // Keep a pixel offset through keyboard opening and dismissal.
    emptyCenter = (article?.clientHeight ?? 0) / 2;
  });

  let editItemDialog = $state<WaDialog>();
  let itemToEditID = $state("");
  const itemToEdit = $derived(query.data?.items.find((item) => item.id === itemToEditID));
  const onClickEditItem = (item: Item) => {
    if (editItemDialog) {
      editItemDialog.open = true;
    }
    itemToEditID = item.id;
  };
</script>

<article bind:this={article}>
  <PageHeader title={query.data?.name ?? ""}>
    {#snippet leading()}
      <wa-button appearance="plain" onclick={() => navigate("/lists")}>
        <wa-icon name="arrow-left" label={$messages.common.back}></wa-icon>
      </wa-button>
    {/snippet}
    {#snippet titleContent()}
      {#if loading}
        <wa-skeleton class="title" effect="none" aria-hidden="true"></wa-skeleton>
      {:else}
        {query.data?.name ?? ""}
      {/if}
    {/snippet}
    {#snippet actions()}
      {#if query.data}
        <ListActions list={query.data} onleave={() => navigate("/lists")} />
      {:else if loading}
        <wa-button
          class="loading-actions"
          appearance="plain"
          disabled
          aria-label={$messages.lists.actions}
        >
          <wa-icon name="ellipsis-vertical"></wa-icon>
        </wa-button>
      {/if}
    {/snippet}
  </PageHeader>
  {#if error}
    <aside role="alert">
      <p>{displayedError}</p>
      <wa-button onclick={refresh}>{$messages.common.refresh}</wa-button>
    </aside>
  {/if}
  <div class="list-area" aria-busy={loading}>
    {#if loading}
      <ListSkeleton />
    {:else if query.data}
      {#key query.data.id}
        {#if query.data.items.length === 0 && !error}
          <div class="empty-area" style:top={`${emptyCenter}px`}>
            <EmptyState
              heading={$messages.lists.emptyListTitle}
              description={$messages.lists.emptyListDescription}
            >
              {#snippet illustration()}
                <EmptyShoppingList />
              {/snippet}
            </EmptyState>
          </div>
        {/if}
        <ItemList
          items={query.data.items}
          {onClickEditItem}
          onCheck={(itemID, checked) => check.mutate({ ...target, itemID, checked })}
          onDelete={(itemID) => remove.mutateAsync({ ...target, itemID })}
          onClear={() => clear.mutate({ ...target })}
        />
        <AddItem
          error={error ? displayedError : undefined}
          onAdd={(item) => addition.mutateAsync({ ...target, item })}
        />
      {/key}
    {/if}
  </div>
</article>

<wa-dialog bind:this={editItemDialog} label={$messages.items.edit} light-dismiss>
  {#if error}
    <aside role="alert">
      <p>{displayedError}</p>
      <wa-button onclick={refresh}>{$messages.common.refresh}</wa-button>
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
    position: relative;
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

  .empty-area {
    position: absolute;
    inset-inline: 0;
    transform: translateY(-50%);
    display: flex;
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
