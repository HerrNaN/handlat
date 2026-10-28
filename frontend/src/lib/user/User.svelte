<script lang="ts">
    import { p } from "../../router";
    import { session, clearSession } from "./session";
    import { post } from "../api";
    import { createMutation } from "@tanstack/svelte-query";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";

    const logout = createMutation(() => ({
        mutationKey: ["auth", "logout"],
        mutationFn: () => post("/api/auth/logout"),
        onSuccess: () => {
            clearSession();
            location.assign("/sign-in");
        },
    }));

</script>

<section>
    <h1>Profile</h1>
    {#if logout.isError}<p role="alert">Could not sign out. Check your connection and try again.</p>{/if}
    {#if $session}
        <p>You are signed in as: <span class="username">{$session.name}</span>.</p>
        <wa-button appearance="outlined" onclick={() => logout.mutate()} disabled={logout.isPending}>
            <wa-icon slot="start" name="arrow-right-from-bracket"></wa-icon>
            Log out
        </wa-button>
    {:else}
        <p>You are not signed in.</p>
        <p><a href={p("/sign-in")}>Sign in</a></p>
        <p><a href={p("/sign-up")}>Create an account</a></p>
    {/if}
</section>

<style>
    section {
        padding: var(--wa-space-l);
    }
    .username {
        font-weight: var(--wa-font-weight-bold);
    }
</style>
