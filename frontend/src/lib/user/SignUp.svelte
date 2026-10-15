<script lang="ts">
    import { browserSupportsWebAuthn, startRegistration } from "@simplewebauthn/browser";
    import { navigate } from "../../router";
    import { refreshSession } from "./session";
    import Link from "../components/Link.svelte";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/callout/callout.js";

    let pending = $state(false);
    let error = $state("");

    const signUp = async () => {
        pending = true;
        error = "";
        try {
            const begin = await fetch("/api/auth/register/begin", { method: "POST" });
            if (!begin.ok) throw new Error("Could not start registration");

            const credential = await startRegistration({ optionsJSON: await begin.json() });
            const finish = await fetch("/api/auth/register/finish", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(credential),
            });
            if (!finish.ok) throw new Error(await finish.text());

            await refreshSession();
            await navigate("/user");
        } catch (cause) {
            error = cause instanceof Error ? cause.message : "Registration failed";
            console.error("Registration failed:", cause)
        } finally {
            pending = false;
        }
    };
</script>

<section>
    <h1>Sign up</h1>
    <p>Create an account using a passkey on this device.</p>
    {#if browserSupportsWebAuthn()}
        <wa-button onclick={signUp} disabled={pending} variant="brand">
            {pending ? "Creating account…" : "Create account with passkey"}
        </wa-button>
        <p class="signin">
            Already have an account?
            <Link to="/sign-in">Sign in</Link>
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

    p.signin {
        display: flex;
        align-items: center;
    }

</style>
