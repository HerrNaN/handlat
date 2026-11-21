<script lang="ts">
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import { createQuery } from "@tanstack/svelte-query";

  import { navigate, route } from "../../router";
  import { ApiError } from "../api";
  import PageHeader from "../components/PageHeader.svelte";
  import { messages } from "../i18n";
  import { queryClient } from "../query";
  import { session } from "../user/session";
  import { getRecipe, getRecipes, recipeKey, recipesKey, type Recipe } from "./api";
  import RecipeActions from "./RecipeActions.svelte";
  import RecipeEditor from "./RecipeEditor.svelte";
  import RecipeIngredients from "./RecipeIngredients.svelte";

  let { id, account }: { id: string; account: string } = $props();
  const isNew = $derived(id === "new");
  const collection = createQuery(() => ({
    queryKey: recipesKey(account),
    queryFn: ({ signal }) => getRecipes(signal),
    enabled: isNew,
  }));
  const detail = createQuery(() => ({
    queryKey: recipeKey(account, id),
    queryFn: ({ signal }) => getRecipe(id, signal),
    enabled: !isNew,
  }));

  const editing = $derived(!isNew && route.search.edit === true);
  const closeEditor = () =>
    navigate("/recipes/:recipeId", { params: { recipeId: id }, replace: true });
  const saved = (recipe: Recipe) => {
    if (account !== $session?.id) {
      return;
    }
    queryClient.setQueryData(recipeKey(account, recipe.id), recipe);
    void queryClient.invalidateQueries({ queryKey: recipesKey(account) });
    void navigate("/recipes/:recipeId", { params: { recipeId: recipe.id }, replace: true });
  };
</script>

<article>
  <PageHeader
    title={isNew
      ? $messages.recipes.newRecipe
      : editing
        ? $messages.recipes.edit
        : (detail.data?.name ?? $messages.nav.recipes)}
  >
    {#snippet leading()}
      <wa-button appearance="plain" onclick={() => navigate("/recipes")}>
        <wa-icon name="arrow-left" label={$messages.common.back}></wa-icon>
      </wa-button>
    {/snippet}
    {#snippet actions()}
      {#if !isNew && detail.data && !editing}
        <RecipeActions
          recipe={detail.data}
          {account}
          onEdit={() => {
            void navigate("/recipes/:recipeId", {
              params: { recipeId: id },
              search: { edit: true },
              replace: true,
            });
          }}
          onDeleted={() => navigate("/recipes", { replace: true })}
        />
      {/if}
    {/snippet}
  </PageHeader>
  {#if isNew && collection.isSuccess && collection.data.enabled}
    <RecipeEditor onSave={saved} onCancel={() => navigate("/recipes")} />
  {:else if editing && detail.isSuccess}
    <RecipeEditor recipe={detail.data} onSave={saved} onCancel={closeEditor} />
  {:else}
    <div class="recipe-content">
      <section>
        {#if isNew}
          {#if collection.isPending}
            <p role="status">{$messages.recipes.loading}</p>
          {:else if collection.isError}
            <p role="alert">{$messages.recipes.loadFailed}</p>
            <wa-button onclick={() => collection.refetch()}>{$messages.common.tryAgain}</wa-button>
          {:else}
            <p>{$messages.recipes.disclosure}</p>
            <wa-button href="/recipes">{$messages.nav.recipes}</wa-button>
          {/if}
        {:else if detail.isPending}
          <p role="status">{$messages.recipes.loading}</p>
        {:else if detail.error instanceof ApiError && detail.error.status === 404}
          <p role="alert">{$messages.recipes.unavailable}</p>
        {:else if detail.isError}
          <p role="alert">{$messages.recipes.loadFailed}</p>
          <wa-button onclick={() => detail.refetch()}>{$messages.common.tryAgain}</wa-button>
        {:else if detail.data}
          <RecipeIngredients recipe={detail.data} {account} />
        {/if}
      </section>
    </div>
  {/if}
</article>

<style>
  article {
    --ingredient-row-height: 2rem;

    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .recipe-content {
    flex: 1;
    min-block-size: 0;
    overflow-y: auto;
  }

  section {
    max-inline-size: 40rem;
    margin-inline: auto;
    padding: var(--wa-space-l);
  }
</style>
