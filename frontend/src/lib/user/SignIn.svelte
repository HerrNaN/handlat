<script lang="ts">
    import { browserSupportsWebAuthn, startAuthentication } from "@simplewebauthn/browser";
    import { navigate } from "../../router";
    import { refreshSession } from "./session";
    import Link from "../components/Link.svelte";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/callout/callout.js";

    let pending = $state(false);
    let error = $state("");

    const signIn = async () => {
        pending = true;
        error = "";
        try {
            const begin = await fetch("/api/auth/login/begin", { method: "POST" });
            if (!begin.ok) throw new Error("Could not start sign-in");

            const credential = await startAuthentication({ optionsJSON: await begin.json() });
            const finish = await fetch("/api/auth/login/finish", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(credential),
            });
            if (!finish.ok) throw new Error(await finish.text());

            await refreshSession();
            await navigate("/user");
        } catch (cause) {
            error = cause instanceof Error ? cause.message : "Sign-in failed";
            console.error("Sign-in failed:", cause)
        } finally {
            pending = false;
        }
    };
</script>

<section>
    <h1>Sign in</h1>
    <p>Sign in to your account using your passkey.</p>
    {#if browserSupportsWebAuthn()}
        <wa-button onclick={signIn} disabled={pending} variant="brand">
            {pending ? "Signing in…" : "Sign in with passkey"}
        </wa-button>
        <p class="signup">
            Don't have an account?
            <Link to="/sign-up">Sign up</Link>
        </p>
    {:else}
        <p>This browser does not support passkeys.</p>
    {/if}
    {#if error}
        <wa-callout variant="danger">
            <wa-icon slot="icon" name="circle-exclamation"></wa-icon>
            {error}
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
