<script lang="ts">
    import { onMount } from "svelte";
    import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";
    import AddItem from "./AddItem.svelte";
    import ItemList from "./ItemList.svelte";
    import { listStore } from "./store";
    import { type Item } from "./list";
    import EditItem from "./EditItem.svelte";
    import { route, navigate } from "../../router";
    import Input from "../components/Input.svelte";
    import ListStatus from "./ListStatus.svelte";

    const { activeList, status, changeListName, setActiveList } = listStore;

    let editItemDialog = $state<WaDialog>();
    let itemToEditID = $state("");
    const itemToEdit = $derived($activeList.items.find((item) => item.id === itemToEditID));
    const onClickEditItem = (item: Item) => {
        if (editItemDialog) editItemDialog.open = true;
        itemToEditID = item.id;
    };

    $effect(() => {
        const id = route.params.id;
        if (id) void setActiveList(id);
    });

    onMount(() => {
        const refresh = () => { void listStore.refresh(); };
        window.addEventListener("focus", refresh);
        return () => window.removeEventListener("focus", refresh);
    });
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
            {#if $activeList.id}
                <Input value={$activeList.name} onsave={changeListName} disabled={$status.loading || $status.saving} />
            {/if}
        </h2>
    </header>
    <ListStatus />
    {#if $activeList.id}
    <div class="list-area">
        <ItemList items={$activeList.items} {onClickEditItem} />
        <AddItem />
    </div>
    {/if}
</article>

<wa-dialog bind:this={editItemDialog} label="Edit Item" light-dismiss>
    <ListStatus />
    {#if itemToEdit}
        <EditItem item={itemToEdit} />
    {/if}
</wa-dialog>

<style>
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
