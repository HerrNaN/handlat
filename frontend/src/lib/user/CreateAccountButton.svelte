<script lang="ts">
  import { startRegistration } from "@simplewebauthn/browser";
  import { createMutation } from "@tanstack/svelte-query";

  import { post } from "../api";
  import { messages } from "../i18n";
  import { passkeyErrorMessage } from "../i18n/errors";
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
    mutationKey: ["auth", "create-account"],
    mutationFn: async () => {
      const begin = await post("/api/auth/register/begin");
      const credential = await startRegistration({ optionsJSON: await begin.json() });
      await post("/api/auth/register/finish", credential);
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

<wa-button
  onclick={authenticate}
  disabled={pending}
  loading={mutation.isPending}
  appearance="outlined"
>
  {mutation.isPending ? $messages.auth.creatingAccount : $messages.auth.createAccount}
</wa-button>
{#if mutation.isError}
  <wa-callout variant="danger" role="alert">
    <wa-icon slot="icon" name="circle-exclamation"></wa-icon>
    {passkeyErrorMessage(mutation.error, $messages, $messages.errors.createAccount)}
  </wa-callout>
{/if}
