<script lang="ts">
  import { browserSupportsWebAuthn } from "@simplewebauthn/browser";
  import { createMutation, createQuery } from "@tanstack/svelte-query";
  import "@awesome.me/webawesome/dist/components/button/button.js";

  import { navigate, route } from "../../router";
  import { queryClient } from "../query";
  import CreateAccountButton from "../user/CreateAccountButton.svelte";
  import LoginButton from "../user/LoginButton.svelte";
  import { session } from "../user/session";
  import { acceptInvitation, ApiError, errorMessage, getInvitation } from "./api";
  import { listKey, summariesKey } from "./queries";

  const token = $derived(route.params.token ?? "");
  let pending = $state(false);
  const invitation = createQuery(() => ({
    queryKey: ["invitation", token, $session?.id],
    queryFn: ({ signal }) => getInvitation(token, signal),
    retry: false,
  }));
  const acceptance = createMutation(() => ({
    mutationFn: ({ token }: { token: string; account: string }) => acceptInvitation(token),
    onSuccess: async (list, { account, token: acceptedToken }) => {
      if (account !== $session?.id || acceptedToken !== token) return;
      queryClient.setQueryData(listKey(account, list.id), list);
      void queryClient.invalidateQueries({ queryKey: summariesKey(account) });
      await navigate("/lists/:id", { params: { id: list.id } });
    },
  }));
  const error = $derived(acceptance.error ?? invitation.error);
  const unavailable = $derived(error instanceof ApiError && error.status === 404);
  const cancel = () => navigate($session ? "/lists" : "/login");
</script>

<section>
  <h1>Invitation</h1>
  {#if unavailable}
    <p>This invitation is no longer available.</p>
    <wa-button onclick={cancel}>Go back</wa-button>
  {:else if invitation.data}
    <p>You've been invited to join <strong>{invitation.data.name}</strong>.</p>
    <div>
      {#if $session}
        <wa-button
          variant="brand"
          onclick={() => $session && acceptance.mutate({ token, account: $session.id })}
          disabled={acceptance.isPending}
          loading={acceptance.isPending}
        >
          Accept invitation
        </wa-button>
      {:else}
        <p>To accept, log in or create an account.</p>
        {#if browserSupportsWebAuthn()}
          <LoginButton bind:pending />
          <CreateAccountButton bind:pending />
        {:else}
          <p>This browser does not support passkeys.</p>
        {/if}
      {/if}
      <wa-button appearance="plain" onclick={cancel} disabled={pending || acceptance.isPending}
        >Cancel</wa-button
      >
    </div>
  {:else if invitation.isPending}
    <p role="status">Loading invitation…</p>
  {/if}
  {#if error && !unavailable}
    <p role="alert">{errorMessage(error)}</p>
    {#if !invitation.data}<wa-button onclick={() => invitation.refetch()}>Try again</wa-button>{/if}
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
