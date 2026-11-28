<script lang="ts">
  import { browserSupportsWebAuthn, startAuthentication } from "@simplewebauthn/browser";
  import { createMutation, createQuery } from "@tanstack/svelte-query";
  import z from "zod";

  import { ApiError, post, readApiError } from "../api";
  import { messages } from "../i18n";
  import { passkeyErrorMessage } from "../i18n/errors";
  import "@awesome.me/webawesome/dist/components/button/button.js";

  const request = new URLSearchParams(location.search).get("request") ?? "";
  const path = `/api/mcp/approvals/${encodeURIComponent(request)}`;
  const clientSchema = z.object({
    name: z.string(),
    domain: z.string(),
    callbackHost: z.string(),
    localCallback: z.boolean(),
    renewable: z.boolean(),
  });
  const redirectSchema = z.object({ redirect: z.url() });

  const client = createQuery(() => ({
    queryKey: ["mcp-approval", request],
    enabled: /^[A-Z2-7]{26}$/.test(request),
    retry: false,
    queryFn: async ({ signal }) => {
      const response = await fetch(path, { signal, cache: "no-store" });
      if (!response.ok) {
        throw await readApiError(response);
      }
      return clientSchema.parse(await response.json());
    },
  }));

  const approve = createMutation(() => ({
    mutationFn: async () => {
      const begin = await post(`${path}/passkey/begin`);
      const credential = await startAuthentication({ optionsJSON: await begin.json() });
      const response = await post(`${path}/passkey/finish`, credential);
      return redirectSchema.parse(await response.json()).redirect;
    },
    onSuccess: (redirect) => location.assign(redirect),
  }));

  const decline = createMutation(() => ({
    mutationFn: async () => {
      const response = await post(`${path}/deny`);
      return redirectSchema.parse(await response.json()).redirect;
    },
    onSuccess: (redirect) => location.assign(redirect),
  }));

  const pending = $derived(approve.isPending || decline.isPending);
  const approvalError = $derived(
    approve.error instanceof ApiError && approve.error.code === "passkey_account_mismatch"
      ? $messages.connectedApps.accountMismatch
      : passkeyErrorMessage(approve.error, $messages, $messages.connectedApps.approveFailed),
  );
</script>

<section aria-labelledby="connect-title">
  <h1 id="connect-title">{$messages.connectedApps.approveTitle}</h1>
  {#if client.isLoading}
    <p role="status">{$messages.connectedApps.loading}</p>
  {:else if client.data}
    <p class="domain" dir="ltr">{client.data.domain}</p>
    <p>{$messages.connectedApps.approveDescription(client.data.name)}</p>
    <p>
      {client.data.localCallback
        ? $messages.connectedApps.localCallback
        : $messages.connectedApps.callbackDestination}
      <strong class="domain" dir="ltr">{client.data.callbackHost}</strong>.
    </p>
    <p>
      {client.data.renewable
        ? $messages.connectedApps.approveDuration
        : $messages.connectedApps.approveLimitedDuration}
    </p>
    {#if !browserSupportsWebAuthn()}
      <p role="alert">{$messages.auth.passkeysUnsupported}</p>
    {/if}
    {#if approve.isError}
      <p role="alert">{approvalError}</p>
    {/if}
    {#if decline.isError}
      <p role="alert">{$messages.connectedApps.declineFailed}</p>
    {/if}
    <div class="actions">
      <wa-button onclick={() => decline.mutate()} disabled={pending} loading={decline.isPending}>
        {$messages.connectedApps.decline}
      </wa-button>
      <wa-button
        variant="brand"
        onclick={() => approve.mutate()}
        disabled={pending || !browserSupportsWebAuthn()}
        loading={approve.isPending}
      >
        {$messages.connectedApps.approve}
      </wa-button>
    </div>
  {:else}
    <p role="alert">{$messages.connectedApps.expired}</p>
  {/if}
</section>

<style>
  section {
    box-sizing: border-box;
    max-width: 34rem;
    margin-inline: auto;
    padding: var(--wa-space-xl);
  }

  .domain {
    font-weight: var(--wa-font-weight-bold);
    overflow-wrap: anywhere;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: var(--wa-space-s);
    margin-block-start: var(--wa-space-xl);
  }
</style>
