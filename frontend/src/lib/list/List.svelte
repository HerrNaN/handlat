<script lang="ts">
    import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";
    import AddItem from "./AddItem.svelte";
    import ItemList from "./ItemList.svelte";
    import { createMutation, createQuery, useMutationState } from "@tanstack/svelte-query";
    import { addItemMutation, checkItemMutation, clearCheckedMutation, deleteItemMutation, dismissListErrors, listQuery, renameListMutation, updateItemMutation } from "./queries";
    import { session } from "../user/session";
    import { type Item } from "./list";
    import EditItem from "./EditItem.svelte";
    import { route, navigate } from "../../router";
    import Input from "../components/Input.svelte";
    import { errorMessage } from "./api";

    const target = $derived({ account: $session?.id ?? "", listID: route.params.id ?? "" });
    const query = createQuery(() => listQuery($session?.id, target.listID));
    const rename = createMutation(() => renameListMutation());
    const addition = createMutation(() => addItemMutation());
    const update = createMutation(() => updateItemMutation());
    const check = createMutation(() => checkItemMutation());
    const remove = createMutation(() => deleteItemMutation());
    const clear = createMutation(() => clearCheckedMutation());
    const mutations = [rename, addition, update, check, remove, clear];
    const writes = useMutationState({
        filters: { mutationKey: ["private-lists"] },
        select: (mutation) => ({
            status: mutation.state.status,
            error: mutation.state.error,
            variables: mutation.state.variables as { account: string; listID?: string; itemID?: string } | undefined,
        }),
    });
    const listWrites = $derived(writes.filter((write) => write.variables?.account === target.account && write.variables.listID === target.listID));
    const pending = $derived(listWrites.filter((write) => write.status === "pending"));
    const pendingItems = $derived(new Set(pending.map((write) => write.variables?.itemID).filter((id) => id !== undefined)));
    const error = $derived(listWrites.findLast((write) => write.status === "error")?.error ?? query.error);
    const refresh = async () => {
        if (pending.length || query.isFetching) return;
        mutations.forEach((mutation) => mutation.reset());
        dismissListErrors();
        await query.refetch();
    };
    const saveName = async (name: string) => {
        try {
            await rename.mutateAsync({ ...target, name });
            return true;
        } catch {
            return false;
        }
    };

    let editItemDialog = $state<WaDialog>();
    let itemToEditID = $state("");
    const itemToEdit = $derived(query.data?.items.find((item) => item.id === itemToEditID));
    const onClickEditItem = (item: Item) => {
        if (editItemDialog) editItemDialog.open = true;
        itemToEditID = item.id;
    };

</script>

<article>
    <header>
        <wa-button
            appearance="plain"
            aria-label="Back"
            onclick={() => navigate("/lists")}
        >
            <wa-icon name="arrow-left"></wa-icon>
        </wa-button>
        <h2>
            {#if query.data}
                <Input value={query.data.name} onsave={saveName} disabled={rename.isPending} />
            {/if}
        </h2>
    </header>
    {#if query.isFetching && !query.data}
        <p role="status">Loading…</p>
    {/if}
    {#if error}
        <aside role="alert">
            <p>{errorMessage(error)}</p>
            <wa-button onclick={refresh}>Refresh</wa-button>
        </aside>
    {/if}
    {#if query.data}
    {#key query.data.id}
    <div class="list-area">
        <ItemList items={query.data.items} {onClickEditItem} {pendingItems} clearing={clear.isPending}
            onCheck={(itemID, checked) => check.mutate({ ...target, itemID, checked })}
            onDelete={(itemID) => remove.mutateAsync({ ...target, itemID })}
            onClear={() => clear.mutate({ ...target })} />
        <AddItem busy={addition.isPending} {error} onAdd={(item) => addition.mutateAsync({ ...target, item })} />
    </div>
    {/key}
    {/if}
</article>

<wa-dialog bind:this={editItemDialog} label="Edit Item" light-dismiss>
    {#if error}
        <aside role="alert">
            <p>{errorMessage(error)}</p>
            <wa-button onclick={refresh}>Refresh</wa-button>
        </aside>
    {/if}
    {#if itemToEdit}
        <EditItem item={itemToEdit} busy={pendingItems.has(itemToEdit.id)} onUpdate={(itemID, changes) => update.mutate({ ...target, itemID, changes })} />
    {/if}
</wa-dialog>

<style>
    aside, p[role="status"], aside p { margin: var(--wa-space-s) var(--wa-space-m); }

    article {
        height: 100%;
        display: flex;
        flex-direction: column;
        background-color: var(--wa-color-surface-default);
        overflow-y: hidden;
    }

    .list-area {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow-y: auto;
        background-color: var(--wa-color-surface-lowered);
    }

    header {
        height: 2rem;
        z-index: 1;
        position: sticky;
        top: 0;
        display: flex;
        align-items: center;
        padding: var(--wa-space-m);
        background-color: var(--wa-color-surface-raised);
        box-shadow: 0 var(--wa-shadow-offset-x-m) var(--wa-shadow-offset-y-m)
            var(--wa-color-shadow);
    }

    h2 {
        margin: 0;
        margin-inline-start: var(--wa-space-xs);
        font-size: var(--wa-font-size-xl);
        font-weight: var(--wa-font-weight-normal);
        color: var(--wa-color-text-normal);
    }
</style>
