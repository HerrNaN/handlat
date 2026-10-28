<script lang="ts">
    import { browserSupportsWebAuthn, startRegistration } from "@simplewebauthn/browser";
    import { createMutation } from "@tanstack/svelte-query";
    import { navigate } from "../../router";
    import { clearSession, refreshSession } from "./session";
    import { post } from "../api";
    import Link from "../components/Link.svelte";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/callout/callout.js";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";

    const signUp = createMutation(() => ({
        mutationKey: ["auth", "sign-up"],
        mutationFn: async () => {
            const begin = await post("/api/auth/register/begin");

            const credential = await startRegistration({ optionsJSON: await begin.json() });
            await post("/api/auth/register/finish", credential);
        },
        onSuccess: async () => {
            clearSession();
            await refreshSession();
            await navigate("/user");
        },
    }));
</script>

<section>
    <h1>Sign up</h1>
    <p>Create an account using a passkey on this device.</p>
    {#if browserSupportsWebAuthn()}
        <wa-button onclick={() => signUp.mutate()} disabled={signUp.isPending} variant="brand">
            {signUp.isPending ? "Creating account…" : "Create account with passkey"}
        </wa-button>
        <p class="signin">
            Already have an account?
            <Link to="/sign-in">Sign in</Link>
        </p>
    {:else}
        <p>This browser does not support passkeys.</p>
    {/if}
    {#if signUp.isError}
    <wa-callout variant="danger">
        <wa-icon slot="icon" name="circle-exclamation"></wa-icon>
        {signUp.error instanceof Error ? signUp.error.message : "Registration failed"}
    </wa-callout>
    {/if}
</section>

<style>
    section {
        display: flex;
        flex-direction: column;

        padding: var(--wa-space-l);
    }

    p.signin {
        display: flex;
        align-items: center;
    }

</style>
