<script lang="ts">
  import { browserSupportsWebAuthn } from "@simplewebauthn/browser";

  import { navigate } from "../../router";
  import CreateAccountButton from "./CreateAccountButton.svelte";
  import LoginButton from "./LoginButton.svelte";

  let pending = $state(false);
  const onSuccess = async () => {
    await navigate("/user");
  };
</script>

<section>
  <h1>Welcome to Handlat</h1>
  <p>Use a passkey to log in or create an account.</p>
  {#if browserSupportsWebAuthn()}
    <div>
      <LoginButton bind:pending onsuccess={onSuccess} />
      <CreateAccountButton bind:pending onsuccess={onSuccess} />
    </div>
  {:else}
    <p>This browser does not support passkeys.</p>
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
