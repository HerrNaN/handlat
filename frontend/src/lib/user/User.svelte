<script lang="ts">
    import { p } from "../../router";
    import { session, clearSession } from "./session";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";

    let error = $state("");
    const logout = async () => {
        try {
            const response = await fetch("/api/auth/logout", { method: "POST" });
            if (!response.ok) throw new Error("Logout failed");
            clearSession();
            location.assign("/sign-in");
        } catch {
            error = "Could not sign out. Check your connection and try again.";
        }
    };

</script>

<section>
    <h1>Profile</h1>
    {#if error}<p role="alert">{error}</p>{/if}
    {#if $session}
        <p>You are signed in as: <span class="username">{$session.name}</span>.</p>
        <wa-button appearance="outlined" onclick={logout}>
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
