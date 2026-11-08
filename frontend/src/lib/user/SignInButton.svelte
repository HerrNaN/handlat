<script lang="ts">
  import { startAuthentication } from "@simplewebauthn/browser";
  import { createMutation } from "@tanstack/svelte-query";

  import { post } from "../api";
  import { clearSession, refreshSession } from "./session";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/callout/callout.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  let {
    pending = $bindable(false),
    onsuccess,
  }: {
    pending?: boolean;
    onsuccess?: () => void | Promise<void>;
  } = $props();

  const mutation = createMutation(() => ({
    mutationKey: ["auth", "sign-in"],
    mutationFn: async () => {
      const begin = await post("/api/auth/login/begin");
      const credential = await startAuthentication({ optionsJSON: await begin.json() });
      await post("/api/auth/login/finish", credential);
    },
    onSuccess: async () => {
      clearSession();
      await refreshSession();
      await onsuccess?.();
    },
    onSettled: () => {
      pending = false;
    },
  }));

  const authenticate = () => {
    if (pending) return;
    pending = true;
    mutation.mutate();
  };
</script>

<wa-button onclick={authenticate} disabled={pending} loading={mutation.isPending} variant="brand">
  {mutation.isPending ? "Logging in…" : "Log in with passkey"}
</wa-button>
{#if mutation.isError}
  <wa-callout variant="danger" role="alert">
    <wa-icon slot="icon" name="circle-exclamation"></wa-icon>
    {mutation.error instanceof Error ? mutation.error.message : "Sign-in failed"}
  </wa-callout>
{/if}
