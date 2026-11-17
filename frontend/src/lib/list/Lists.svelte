<script lang="ts">
  import { createMutation, createQuery } from "@tanstack/svelte-query";

  import { p, navigate } from "../../router";
  import PageHeader from "../components/PageHeader.svelte";
  import { session } from "../user/session";
  import { errorMessage } from "./api";
  import EmptyState from "./EmptyState.svelte";
  import ListActions from "./ListActions.svelte";
  import ListSkeleton from "./ListSkeleton.svelte";
  import { createListMutation, summariesQuery } from "./queries";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/callout/callout.js";
  import "@awesome.me/webawesome/dist/components/badge/badge.js";

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
  <PageHeader title="Lists">
    {#snippet actions()}
      <wa-button onclick={handleCreateList} appearance="plain" disabled={!$session}>
        <wa-icon name="plus" label="New list"></wa-icon>
      </wa-button>
    {/snippet}
  </PageHeader>
  {#if createList.error ?? query.error}
    <aside role="alert">
      <wa-callout variant="danger">
        <wa-icon slot="icon" name="circle-exclamation"></wa-icon>
        {errorMessage(createList.error ?? query.error)}
      </wa-callout>
    </aside>
  {/if}
  <div class="content">
    {#if query.isPending}
      <ListSkeleton overview />
    {:else if query.isSuccess && query.data.length === 0}
      <EmptyState
        icon="list-check"
        heading="Your first list starts here"
        description="For the weekly shop, a quick stop, or anything you don’t want to forget."
      >
        <wa-button onclick={handleCreateList} appearance="outlined" disabled={!$session}>
          <wa-icon slot="start" name="plus"></wa-icon>
          Create list
        </wa-button>
      </EmptyState>
    {:else}
      <ul>
        {#each query.data ?? [] as list (list.id)}
          <li>
            <a href={p("/lists/:id", { params: { id: list.id } })}>
              <wa-icon name="list-check"></wa-icon>
              <span class="name">{list.name}</span>
              {#if list.memberCount > 1}
                <wa-badge variant="neutral" appearance="filled" pill>Shared</wa-badge>
              {/if}
              <span class="count">{list.itemCount}</span>
            </a>
            <ListActions {list} />
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</article>

<style>
  aside {
    margin: var(--wa-space-s) var(--wa-space-m);
  }

  article {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: var(--wa-color-surface-default);
    overflow-y: hidden;
  }

  .content {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
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
    background-color: var(--wa-color-neutral-fill-quiet);
    border-radius: var(--wa-border-radius-s);
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
    background-color: var(--wa-color-neutral-fill-quiet);
    border-radius: var(--wa-border-radius-s);
  }

  .name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  a > wa-icon,
  wa-badge,
  .count {
    flex-shrink: 0;
  }

  .count {
    color: var(--wa-color-text-quiet);
    font-size: var(--wa-font-size-s);
  }
</style>
