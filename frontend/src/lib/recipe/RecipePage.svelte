<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import WaDropdownItem from "@awesome.me/webawesome/dist/components/dropdown-item/dropdown-item.js";
  import type { WaSelectEvent } from "@awesome.me/webawesome/dist/events/select.js";
  import "@awesome.me/webawesome/dist/components/dropdown/dropdown.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import { createMutation, createQuery } from "@tanstack/svelte-query";

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
  let deleteDialog = $state<WaDialog>();
  const saved = (recipe: Recipe) => {
    if (account !== $session?.id) {
      return;
    }
    queryClient.setQueryData(recipeKey(account, recipe.id), recipe);
    void queryClient.invalidateQueries({ queryKey: recipesKey(account) });
    if (isNew) {
      void navigate("/recipes/:recipeId", { params: { recipeId: recipe.id }, replace: true });
    } else {
      editing = false;
    }
  };
  const remove = createMutation(() => ({
    mutationFn: async () => {
      await deleteRecipe(id);
      if (account !== $session?.id) {
        return;
      }
      void queryClient.invalidateQueries({ queryKey: recipesKey(account) });
      await navigate("/recipes", { replace: true });
      queryClient.removeQueries({ queryKey: recipeKey(account, id), exact: true });
    },
  }));

  const selectAction = (event: WaSelectEvent) => {
    const action = (event.detail.item as WaDropdownItem).value;
    if (action === "edit") {
      editing = true;
    }
    if (action === "delete" && deleteDialog) {
      remove.reset();
      deleteDialog.open = true;
    }
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
        <wa-dropdown placement="bottom-end" onwa-select={selectAction}>
          <wa-button slot="trigger" appearance="plain" disabled={remove.isPending}>
            <wa-icon name="ellipsis-vertical" label={$messages.recipes.actionsFor(detail.data.name)}
            ></wa-icon>
          </wa-button>
          <wa-dropdown-item value="edit">
            <wa-icon slot="icon" name="pen"></wa-icon>
            {$messages.recipes.edit}
          </wa-dropdown-item>
          <wa-dropdown-item value="delete" variant="danger">
            <wa-icon slot="icon" name="trash-can" variant="solid"></wa-icon>
            {$messages.recipes.delete}
          </wa-dropdown-item>
        </wa-dropdown>
      {/if}
    {/snippet}
  </PageHeader>
  <div class="recipe-content">
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
        {/if}
      {/if}
    </section>
  </div>
</article>

{#if !isNew && detail.data}
  <wa-dialog
    bind:this={deleteDialog}
    label={$messages.recipes.confirmDelete}
    onwa-hide={(event: Event) => {
      if (remove.isPending) event.preventDefault();
    }}
  >
    <p>{$messages.recipes.deleteDescription(detail.data.name)}</p>
    {#if remove.isError}
      <p role="alert">{$messages.recipes.deleteFailed}</p>
    {/if}
    <wa-button slot="footer" data-dialog="close" disabled={remove.isPending}>
      {$messages.common.cancel}
    </wa-button>
    <wa-button
      slot="footer"
      variant="danger"
      loading={remove.isPending}
      onclick={() => remove.mutate()}
    >
      {$messages.recipes.delete}
    </wa-button>
  </wa-dialog>
{/if}

<style>
  article {
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
