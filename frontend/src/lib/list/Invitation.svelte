<script lang="ts">
  import { browserSupportsWebAuthn } from "@simplewebauthn/browser";
  import { createMutation, createQuery } from "@tanstack/svelte-query";

  import { navigate, route } from "../../router";
  import { ApiError } from "../api";
  import { messages } from "../i18n";
  import "@awesome.me/webawesome/dist/components/button/button.js";

  import { errorMessage } from "../i18n/errors";
  import { queryClient } from "../query";
  import CreateAccountButton from "../user/CreateAccountButton.svelte";
  import LoginButton from "../user/LoginButton.svelte";
  import { session } from "../user/session";
  import { acceptInvitation, getInvitation } from "./api";
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
  <h1>{$messages.invitations.title}</h1>
  {#if unavailable}
    <p>{$messages.errors.invitationUnavailable}</p>
    <wa-button onclick={cancel}>{$messages.invitations.goBack}</wa-button>
  {:else if invitation.data}
    <p>
      {$messages.invitations.inviteToJoin.beforeName}<strong>{invitation.data.name}</strong
      >{$messages.invitations.inviteToJoin.afterName}
    </p>
    <div>
      {#if $session}
        <wa-button
          variant="brand"
          onclick={() => $session && acceptance.mutate({ token, account: $session.id })}
        >
          {$messages.invitations.accept}
        </wa-button>
      {:else}
        <p>{$messages.invitations.signInToAccept}</p>
        {#if browserSupportsWebAuthn()}
          <LoginButton bind:pending />
          <CreateAccountButton bind:pending />
        {:else}
          <p>{$messages.auth.passkeysUnsupported}</p>
        {/if}
      {/if}
      <wa-button appearance="plain" onclick={cancel}>{$messages.common.cancel}</wa-button>
    </div>
  {:else if invitation.isPending}
    <p role="status">{$messages.invitations.loading}</p>
  {/if}
  {#if error && !unavailable}
    <p role="alert">
      {errorMessage(
        error,
        $messages,
        acceptance.error ? $messages.errors.acceptInvitation : $messages.errors.loadInvitation,
      )}
    </p>
    {#if !invitation.data}<wa-button onclick={() => invitation.refetch()}
        >{$messages.common.tryAgain}</wa-button
      >{/if}
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
