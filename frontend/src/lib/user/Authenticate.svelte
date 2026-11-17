<script lang="ts">
  import { browserSupportsWebAuthn } from "@simplewebauthn/browser";

  import { navigate } from "../../router";
  import { messages } from "../i18n";
  import CreateAccountButton from "./CreateAccountButton.svelte";
  import LoginButton from "./LoginButton.svelte";

  let pending = $state(false);
  const onSuccess = async () => {
    await navigate("/user");
  };
</script>

<section>
  <h1>{$messages.auth.welcome}</h1>
  <p>{$messages.auth.passkeyPrompt}</p>
  {#if browserSupportsWebAuthn()}
    <div>
      <LoginButton bind:pending onsuccess={onSuccess} />
      <CreateAccountButton bind:pending onsuccess={onSuccess} />
    </div>
  {:else}
    <p>{$messages.auth.passkeysUnsupported}</p>
  {/if}
</section>

<style>
  section {
    padding: var(--wa-space-l);
  }

  div {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: var(--wa-space-m);
  }
</style>
