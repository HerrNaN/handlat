<script lang="ts">
    import { browserSupportsWebAuthn, startAuthentication } from "@simplewebauthn/browser";
    import { createMutation } from "@tanstack/svelte-query";
    import { navigate } from "../../router";
    import { clearSession, refreshSession } from "./session";
    import { post } from "../api";
    import Link from "../components/Link.svelte";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/callout/callout.js";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";

    const signIn = createMutation(() => ({
        mutationKey: ["auth", "sign-in"],
        mutationFn: async () => {
            const begin = await post("/api/auth/login/begin");

            const credential = await startAuthentication({ optionsJSON: await begin.json() });
            await post("/api/auth/login/finish", credential);
        },
        onSuccess: async () => {
            clearSession();
            await refreshSession();
            await navigate("/user");
        },
    }));
</script>

<section>
    <h1>Sign in</h1>
    <p>Sign in to your account using your passkey.</p>
    {#if browserSupportsWebAuthn()}
        <wa-button onclick={() => signIn.mutate()} disabled={signIn.isPending} variant="brand">
            {signIn.isPending ? "Signing in…" : "Sign in with passkey"}
        </wa-button>
        <p class="signup">
            Don't have an account?
            <Link to="/sign-up">Sign up</Link>
        </p>
    {:else}
        <p>This browser does not support passkeys.</p>
    {/if}
    {#if signIn.isError}
        <wa-callout variant="danger">
            <wa-icon slot="icon" name="circle-exclamation"></wa-icon>
            {signIn.error instanceof Error ? signIn.error.message : "Sign-in failed"}
        </wa-callout>
    {/if}
</section>

<style>
    section {
        display: flex;
        flex-direction: column;

        padding: var(--wa-space-l);
    }

    p.signup {
        display: flex;
        align-items: center;
    }
</style>
