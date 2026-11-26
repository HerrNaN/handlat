<script lang="ts">
  import { createQuery } from "@tanstack/svelte-query";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/callout/callout.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  import { navigate, p } from "../../router";
  import PageHeader from "../components/PageHeader.svelte";
  import { messages } from "../i18n";
  import EmptyState from "../list/EmptyState.svelte";
  import ListSkeleton from "../list/ListSkeleton.svelte";
  import { session } from "../user/session";
  import { recipeCollectionQuery } from "./api";
  import RecipeActions from "./RecipeActions.svelte";

  const newRecipeURL = p("/recipes/:recipeId", { params: { recipeId: "new" } });
  const collection = createQuery(() => recipeCollectionQuery($session?.id));
</script>

<article>
  <PageHeader
    title={collection.data?.enabled ? $messages.recipes.collection : $messages.nav.recipes}
  >
    {#snippet actions()}
      {#if collection.data?.enabled}
        <wa-button href={newRecipeURL} appearance="plain">
          <wa-icon name="plus" label={$messages.recipes.newRecipe}></wa-icon>
        </wa-button>
      {/if}
    {/snippet}
  </PageHeader>

  <div class="content">
    {#if collection.isPending}
      <ListSkeleton overview label={$messages.recipes.loading} />
    {:else if collection.isError}
      <aside>
        <wa-callout variant="danger" role="alert">
          <wa-icon slot="icon" name="circle-exclamation"></wa-icon>
          {$messages.recipes.loadFailed}
        </wa-callout>
        <wa-button onclick={() => collection.refetch()}>{$messages.common.tryAgain}</wa-button>
      </aside>
    {:else if collection.data.enabled && collection.data.recipes.length === 0}
      <EmptyState
        icon="kitchen-set"
        heading={$messages.recipes.emptyTitle}
        description={$messages.recipes.empty}
      >
        <wa-button href={newRecipeURL} appearance="outlined">
          <wa-icon slot="start" name="plus"></wa-icon>
          {$messages.recipes.create}
        </wa-button>
      </EmptyState>
    {:else if collection.data.enabled}
      <ul>
        {#each collection.data.recipes as recipe (recipe.id)}
          <li>
            <a href={p("/recipes/:recipeId", { params: { recipeId: recipe.id } })}>
              <wa-icon name="kitchen-set" aria-hidden="true"></wa-icon>
              <span>{recipe.name}</span>
            </a>
            {#if $session}
              <RecipeActions
                {recipe}
                account={$session.id}
                onEdit={() => {
                  void navigate("/recipes/:recipeId", {
                    params: { recipeId: recipe.id },
                    search: { edit: true },
                  });
                }}
              />
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</article>

<style>
  article {
    height: 100%;
    display: flex;
    flex-direction: column;
    background-color: var(--wa-color-surface-default);
    overflow: hidden;
  }

  .content {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow-y: auto;
  }

  aside {
    margin: var(--wa-space-s) var(--wa-space-m);
    display: grid;
    gap: var(--wa-space-s);
    justify-items: start;
  }

  wa-callout {
    width: 100%;
  }

  ul {
    list-style: none;
    margin: 0;
    padding: var(--wa-space-2xs);
    display: flex;
    flex-direction: column;
    gap: var(--wa-space-2xs);
  }

  li {
    display: flex;
    align-items: center;
    padding-inline-end: var(--wa-space-xs);
    background-color: var(--wa-color-surface-raised);
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
    border-radius: var(--wa-border-radius-s);
  }

  a > wa-icon {
    flex-shrink: 0;
  }

  a > span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
