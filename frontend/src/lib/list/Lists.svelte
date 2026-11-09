<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import { createMutation, createQuery } from "@tanstack/svelte-query";

  import { p, navigate } from "../../router";
  import { session } from "../user/session";
  import { errorMessage, type ListSummary } from "./api";
  import { createListMutation, leaveListMutation, summariesQuery } from "./queries";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/callout/callout.js";

  const createList = createMutation(() => createListMutation());
  const departure = createMutation(() => leaveListMutation());
  let leaveDialog = $state<WaDialog>();
  let listToLeave = $state<ListSummary>();
  let leaving = $state(false);
  const shared = $derived((listToLeave?.memberCount ?? 1) > 1);
  const confirmLeave = (list: ListSummary) => {
    departure.reset();
    listToLeave = list;
    if (leaveDialog) leaveDialog.open = true;
  };
  const handleLeave = async () => {
    if (!$session || !listToLeave || leaving) return;
    leaving = true;
    try {
      await departure.mutateAsync({ account: $session.id, listID: listToLeave.id });
    } catch {
      // Keep the dialog open; the mutation error is displayed below.
      return;
    } finally {
      leaving = false;
    }
    if (leaveDialog) leaveDialog.open = false;
  };
  const query = createQuery(() => summariesQuery($session?.id));
  const handleCreateList = () => {
    if (!$session) return;
    createList.mutate(
      { account: $session.id, name: "Shopping List" },
      {
        onSuccess: (list, { account }) => {
          if (account === $session?.id) void navigate("/lists/:id", { params: { id: list.id } });
        },
      },
    );
  };
</script>

<article>
  <header>
    <h2>Lists</h2>
    <wa-button
      onclick={handleCreateList}
      appearance="plain"
      disabled={!$session || query.isFetching || createList.isPending}
    >
      <wa-icon name="plus" label="New list"></wa-icon>
    </wa-button>
  </header>
  {#if query.isFetching && !query.data}
    <p role="status">Loading…</p>
  {/if}
  {#if createList.error ?? query.error}
    <aside role="alert">
      <wa-callout variant="danger">
        <wa-icon slot="icon" name="circle-exclamation"></wa-icon>
        {errorMessage(createList.error ?? query.error)}
      </wa-callout>
    </aside>
  {/if}
  {#if !query.isFetching && !query.error && !createList.error && query.data?.length === 0}
    <p>No lists yet. Create your first list.</p>
  {/if}
  <ul>
    {#each query.data ?? [] as list (list.id)}
      <li>
        <a href={p("/lists/:id", { params: { id: list.id } })}>
          <wa-icon name="list-check"></wa-icon>
          <span class="name">{list.name}</span>
          <span class="count">{list.itemCount}</span>
        </a>
        <wa-button
          appearance="plain"
          variant="danger"
          onclick={() => confirmLeave(list)}
          disabled={leaving}
        >
          <wa-icon
            name={list.memberCount > 1 ? "arrow-right-from-bracket" : "trash-can"}
            variant={list.memberCount > 1 ? "solid" : "regular"}
            label={`${list.memberCount > 1 ? "Leave" : "Delete"} ${list.name}`}
          ></wa-icon>
        </wa-button>
      </li>
    {/each}
  </ul>
</article>

<wa-dialog
  bind:this={leaveDialog}
  label={shared ? "Leave list?" : "Delete list?"}
  onwa-hide={(event: Event) => {
    if (leaving) event.preventDefault();
  }}
>
  {#if shared}
    <p>Leave “{listToLeave?.name}”? Other members will keep the list.</p>
  {:else}
    <p>
      You're the only member. Leaving will delete “{listToLeave?.name}” and all its items. This
      cannot be undone.
    </p>
  {/if}
  {#if departure.error}
    <p role="alert">{errorMessage(departure.error)}</p>
  {/if}
  <wa-button slot="footer" data-dialog="close" disabled={leaving}>Cancel</wa-button>
  <wa-button
    slot="footer"
    variant="danger"
    onclick={handleLeave}
    disabled={leaving}
    loading={leaving}>{shared ? "Leave list" : "Delete list"}</wa-button
  >
</wa-dialog>

<style>
  aside,
  p[role="status"] {
    margin: var(--wa-space-s) var(--wa-space-m);
  }

  article {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: var(--wa-color-surface-default);
    overflow-y: hidden;
  }

  header {
    height: 2rem;
    z-index: 1;
    position: sticky;
    top: 0;
    padding: var(--wa-space-m) var(--wa-space-l);
    background-color: var(--wa-color-surface-raised);
    box-shadow: 0 var(--wa-shadow-offset-x-m) var(--wa-shadow-offset-y-m) var(--wa-color-shadow);
    display: flex;
    align-items: center;
  }

  h2 {
    flex: 1;
    margin: 0;
    font-size: var(--wa-font-size-xl);
    font-weight: var(--wa-font-weight-normal);
    color: var(--wa-color-text-normal);
  }

  ul {
    flex: 1;
    list-style: none;
    margin: 0;
    padding: var(--wa-space-2xs);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: var(--wa-space-2xs);
  }

  li {
    display: flex;
    align-items: center;
    padding-inline-end: var(--wa-space-xs);
    background-color: var(--wa-color-surface-lowered);
    border-radius: var(--wa-border-width-l);
  }

  a {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: var(--wa-space-s);
    padding: var(--wa-space-m);
    text-decoration: none;
    color: var(--wa-color-text-normal);
    background-color: var(--wa-color-surface-lowered);
    border-radius: var(--wa-border-width-l);
  }

  .name {
    flex: 1;
  }

  .count {
    color: var(--wa-color-neutral-on-quiet);
    font-size: var(--wa-font-size-s);
  }
</style>
