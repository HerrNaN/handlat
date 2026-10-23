<script lang="ts">
    import { createMutation, createQuery } from "@tanstack/svelte-query";
    import { session } from "../user/session";
    import { createListMutation, summariesQuery } from "./queries";
    import { p, navigate } from "../../router";
    import { errorMessage } from "./api";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";
    import "@awesome.me/webawesome/dist/components/button/button.js";

    const createList = createMutation(() => createListMutation());
    const query = createQuery(() => summariesQuery($session?.id));
    const handleCreateList = () => {
        if (!$session) return;
        createList.mutate({ account: $session.id, name: "Shopping List" }, {
            onSuccess: (list, { account }) => {
                if (account === $session?.id) void navigate("/lists/:id", { params: { id: list.id } });
            },
        });
    };
    const refresh = async () => {
        if (query.isFetching || createList.isPending) return;
        createList.reset();
        await query.refetch();
    };
</script>

<article>
    <header>
        <h2>Lists</h2>
    </header>
    <p>Lists are private and saved to your account. Previous browser-only lists are not imported.</p>
    {#if query.isFetching && !query.data}
        <p role="status">Loading…</p>
    {/if}
    {#if createList.error ?? query.error}
        <aside role="alert">
            <p>{errorMessage(createList.error ?? query.error)}</p>
            <wa-button onclick={refresh}>Refresh</wa-button>
        </aside>
    {/if}
    <wa-button onclick={handleCreateList} disabled={!$session || query.isFetching || createList.isPending}>New list</wa-button>
    {#if !query.isFetching && !query.error && !createList.error && query.data?.length === 0}
        <p>No lists yet. Create your first private list.</p>
    {/if}
    <ul>
        {#each query.data ?? [] as list (list.id)}
            <li>
                <a href={p("/lists/:id", { params: { id: list.id } })}>
                    <wa-icon name="list-check"></wa-icon>
                    <span class="name">{list.name}</span>
                    <span class="count">{list.itemCount}</span>
                    <wa-icon name="chevron-right"></wa-icon>
                </a>
            </li>
        {/each}
    </ul>
</article>

<style>
    aside, p[role="status"], aside p { margin: var(--wa-space-s) var(--wa-space-m); }

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
        box-shadow: 0 var(--wa-shadow-offset-x-m) var(--wa-shadow-offset-y-m)
            var(--wa-color-shadow);
    }

    h2 {
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

    a {
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
