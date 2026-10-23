<script lang="ts">
    import { listStore } from "./store";
    import { session } from "../user/session";
    import "@awesome.me/webawesome/dist/components/button/button.js";

    const { status } = listStore;
</script>

{#if $status.loading || $status.saving}
    <p role="status">{$status.saving ? "Saving…" : "Loading…"}</p>
{/if}
{#if $status.error}
    <aside role="alert">
        <p>{$status.error}</p>
        {#if $session}
            <wa-button onclick={() => listStore.refresh()}>Refresh</wa-button>
        {:else}
            <a href="/sign-in">Sign in</a>
        {/if}
    </aside>
{/if}

<style>
    p, aside { margin: var(--wa-space-s) var(--wa-space-m); }
</style>
