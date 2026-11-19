<script lang="ts">
  import { createMutation, createQuery } from "@tanstack/svelte-query";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  import { p } from "../../router";
  import PageHeader from "../components/PageHeader.svelte";
  import { messages } from "../i18n";
  import { queryClient } from "../query";
  import { session } from "../user/session";
  import { enableRecipes, getRecipes, recipesKey } from "./api";

  const collection = createQuery(() => ({
    queryKey: recipesKey($session?.id),
    queryFn: ({ signal }) => getRecipes(signal),
    enabled: Boolean($session),
  }));
  const optIn = createMutation(() => ({
    mutationFn: async (account: string) => {
      await enableRecipes();
      if (account === $session?.id) {
        await queryClient.invalidateQueries({ queryKey: recipesKey(account) });
      }
    },
  }));
</script>

<PageHeader title={collection.data?.enabled ? $messages.recipes.collection : $messages.nav.recipes}>
  {#snippet actions()}
    {#if collection.data?.enabled}
      <wa-button href={p("/recipes/:recipeId", { params: { recipeId: "new" } })}>
        <wa-icon slot="start" name="plus"></wa-icon>
        {$messages.recipes.newRecipe}
      </wa-button>
    {/if}
  {/snippet}
</PageHeader>

<section>
  {#if collection.isPending}
    <p role="status">{$messages.recipes.loading}</p>
  {:else if collection.isError}
    <p role="alert">{$messages.recipes.loadFailed}</p>
    <wa-button onclick={() => collection.refetch()}>{$messages.common.tryAgain}</wa-button>
  {:else if collection.data && !collection.data.enabled}
    <h2>{$messages.recipes.experimental}</h2>
    <p>{$messages.recipes.disclosure}</p>
    <wa-button
      variant="brand"
      loading={optIn.isPending}
      onclick={() => $session && optIn.mutate($session.id)}
    >
      {$messages.recipes.enable}
    </wa-button>
    {#if optIn.isError}
      <p role="alert">{$messages.recipes.enableFailed}</p>
    {/if}
  {:else if collection.data?.recipes.length}
    <ul>
      {#each collection.data.recipes as recipe (recipe.id)}
        <li>
          <a href={p("/recipes/:recipeId", { params: { recipeId: recipe.id } })}>
            <span>{recipe.name}</span>
            <wa-icon name="chevron-right" aria-hidden="true"></wa-icon>
          </a>
        </li>
      {/each}
    </ul>
  {:else}
    <p>{$messages.recipes.empty}</p>
  {/if}
</section>

<style>
  section {
    max-inline-size: 40rem;
    margin-inline: auto;
    padding: var(--wa-space-l);
  }

  ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  a {
    min-block-size: 3rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--wa-space-m);
    padding-block: var(--wa-space-xs);
    border-bottom: 1px solid var(--wa-color-surface-border);
    color: var(--wa-color-text-normal);
    overflow-wrap: anywhere;
  }
</style>
