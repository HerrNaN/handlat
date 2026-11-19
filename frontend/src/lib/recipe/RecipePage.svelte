<script lang="ts">
  import { createMutation, createQuery } from "@tanstack/svelte-query";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  import { navigate } from "../../router";
  import { ApiError } from "../api";
  import PageHeader from "../components/PageHeader.svelte";
  import { messages } from "../i18n";
  import { queryClient } from "../query";
  import { session } from "../user/session";
  import { deleteRecipe, getRecipe, getRecipes, recipeKey, recipesKey, type Recipe } from "./api";
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

  let editing = $state(false);
  const saved = (recipe: Recipe) => {
    if (account !== $session?.id) return;
    queryClient.setQueryData(recipeKey(account, recipe.id), recipe);
    void queryClient.invalidateQueries({ queryKey: recipesKey(account) });
    if (isNew)
      void navigate("/recipes/:recipeId", { params: { recipeId: recipe.id }, replace: true });
    else editing = false;
  };
  const remove = createMutation(() => ({
    mutationFn: async () => {
      await deleteRecipe(id);
      if (account !== $session?.id) return;
      void queryClient.invalidateQueries({ queryKey: recipesKey(account) });
      await navigate("/recipes", { replace: true });
      queryClient.removeQueries({ queryKey: recipeKey(account, id), exact: true });
    },
  }));

  const confirmDelete = () => {
    if (detail.data && window.confirm($messages.recipes.confirmDelete(detail.data.name))) {
      remove.mutate();
    }
  };
</script>

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
      <wa-button disabled={remove.isPending} onclick={() => (editing = true)}>
        {$messages.recipes.edit}
      </wa-button>
    {/if}
  {/snippet}
</PageHeader>

<section>
  {#if isNew}
    {#if collection.isPending}
      <p role="status">{$messages.recipes.loading}</p>
    {:else if collection.isError}
      <p role="alert">{$messages.recipes.loadFailed}</p>
      <wa-button onclick={() => collection.refetch()}>{$messages.common.tryAgain}</wa-button>
    {:else if collection.data?.enabled}
      <RecipeEditor onSave={saved} onCancel={() => navigate("/recipes")} />
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
    {#if editing}
      <RecipeEditor recipe={detail.data} onSave={saved} onCancel={() => (editing = false)} />
    {:else}
      <RecipeIngredients recipe={detail.data} {account} />
      <footer>
        <wa-button
          variant="danger"
          appearance="plain"
          loading={remove.isPending}
          onclick={confirmDelete}
        >
          {$messages.recipes.delete}
        </wa-button>
        {#if remove.isError}
          <p role="alert">{$messages.recipes.deleteFailed}</p>
        {/if}
      </footer>
    {/if}
  {/if}
</section>

<style>
  section {
    max-inline-size: 40rem;
    margin-inline: auto;
    padding: var(--wa-space-l);
  }

  footer {
    border-top: 1px solid var(--wa-color-surface-border);
    padding-block-start: var(--wa-space-m);
    margin-block-start: var(--wa-space-xl);
  }
</style>
