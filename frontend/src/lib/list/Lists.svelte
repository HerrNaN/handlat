<script lang="ts">
  import { createMutation, createQuery } from "@tanstack/svelte-query";

  import { p, navigate } from "../../router";
  import { session } from "../user/session";
  import { errorMessage } from "./api";
  import ListActions from "./ListActions.svelte";
  import { createListMutation, summariesQuery } from "./queries";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/callout/callout.js";

  const createList = createMutation(() => createListMutation());
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
        <ListActions {list} />
      </li>
    {/each}
  </ul>
</article>

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
