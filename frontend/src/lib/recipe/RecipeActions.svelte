<script lang="ts">
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import WaDropdownItem from "@awesome.me/webawesome/dist/components/dropdown-item/dropdown-item.js";
  import "@awesome.me/webawesome/dist/components/dropdown/dropdown.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import type { WaSelectEvent } from "@awesome.me/webawesome/dist/events/select.js";
  import { createMutation } from "@tanstack/svelte-query";

  import { messages } from "../i18n";
  import { queryClient } from "../query";
  import { session } from "../user/session";
  import { deleteRecipe, type getRecipes, type Recipe, recipeKey, recipesKey } from "./api";

  let {
    recipe,
    account,
    onEdit,
    onDeleted,
  }: {
    recipe: Pick<Recipe, "id" | "name">;
    account: string;
    onEdit: () => void;
    onDeleted?: () => Promise<unknown>;
  } = $props();

  let dialog = $state<WaDialog>();
  const remove = createMutation(() => ({
    mutationFn: deleteRecipe,
    onSuccess: async (_data, id) => {
      if (account !== $session?.id) {
        return;
      }

      const indexKey = recipesKey(account);
      queryClient.setQueryData<Awaited<ReturnType<typeof getRecipes>>>(
        indexKey,
        (index) => index && { ...index, recipes: index.recipes.filter((entry) => entry.id !== id) },
      );
      void queryClient.invalidateQueries({ queryKey: indexKey });
      await onDeleted?.();
      queryClient.removeQueries({ queryKey: recipeKey(account, id), exact: true });
    },
  }));

  const selectAction = (event: WaSelectEvent) => {
    const action = (event.detail.item as WaDropdownItem).value;
    if (action === "edit") {
      onEdit();
    }
    if (action === "delete" && dialog) {
      remove.reset();
      dialog.open = true;
    }
  };
</script>

<wa-dropdown placement="bottom-end" onwa-select={selectAction}>
  <wa-button slot="trigger" appearance="plain" disabled={remove.isPending}>
    <wa-icon name="ellipsis-vertical" label={$messages.recipes.actionsFor(recipe.name)}></wa-icon>
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

<wa-dialog
  bind:this={dialog}
  label={$messages.recipes.confirmDelete}
  onwa-hide={(event: Event) => {
    if (remove.isPending) event.preventDefault();
  }}
>
  <p>{$messages.recipes.deleteDescription(recipe.name)}</p>
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
    onclick={() => remove.mutate(recipe.id)}
  >
    {$messages.recipes.delete}
  </wa-button>
</wa-dialog>
