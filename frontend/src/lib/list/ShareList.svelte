<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import type { Snippet } from "svelte";

  import { p } from "../../router";
  import { messages } from "../i18n";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  import { errorMessage } from "../i18n/errors";
  import { createInvitation } from "./api";

  const {
    listID,
    name,
    children,
  }: {
    listID: string;
    name: string;
    children: Snippet<[() => Promise<void>]>;
  } = $props();
  let dialog = $state<WaDialog>();
  let pending = $state(false);
  let link = $state("");
  let invitationError = $state<unknown>();
  let shareError = $state<"shareFailed" | "copyManually">();
  const error = $derived(
    shareError
      ? $messages.invitations[shareError]
      : invitationError
        ? errorMessage(invitationError, $messages, $messages.errors.createInvitation)
        : undefined,
  );
  let copied = $state(false);

  const shareData = () => ({
    title: name,
    text: $messages.invitations.shareText(name),
    url: link,
  });
  const shareNative = async () => {
    try {
      await navigator.share(shareData());
      if (dialog) dialog.open = false;
    } catch (cause) {
      if (cause instanceof Error && cause.name === "AbortError") return;
      shareError = "shareFailed";
    }
  };

  const share = async () => {
    if (pending) return;
    pending = true;
    link = "";
    invitationError = undefined;
    shareError = undefined;
    copied = false;
    try {
      const invite = await createInvitation(listID);
      link = new URL(p("/invitations/:token", { params: { token: invite.token } }), location.origin)
        .href;
      if (navigator.share) {
        try {
          await navigator.share(shareData());
          return;
        } catch (cause) {
          if (cause instanceof Error && cause.name === "AbortError") return;
          // Sharing may require a fresh user gesture after the link request.
          // Offer the copy fallback if the browser cannot open its share sheet.
        }
      }
    } catch (cause) {
      invitationError = cause;
    } finally {
      pending = false;
    }
    if (dialog) dialog.open = true;
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(link);
      copied = true;
      shareError = undefined;
    } catch {
      shareError = "copyManually";
    }
  };
</script>

{@render children(share)}

<wa-dialog bind:this={dialog} label={$messages.invitations.shareList} light-dismiss>
  {#if link}
    <p>{$messages.invitations.shareDescription}</p>
    <input
      aria-label={$messages.invitations.link}
      value={link}
      readonly
      onclick={(event) => event.currentTarget.select()}
    />
    {#if typeof navigator.share === "function"}<wa-button onclick={shareNative}
        >{$messages.invitations.shareInvitation}</wa-button
      >{/if}
    <wa-button onclick={copy}>{$messages.invitations.copyLink}</wa-button>
    {#if copied}<p role="status">{$messages.invitations.copied}</p>{/if}
  {/if}
  {#if error}<p role="alert">{error}</p>{/if}
  <wa-button slot="footer" data-dialog="close">{$messages.common.done}</wa-button>
</wa-dialog>

<style>
  input {
    box-sizing: border-box;
    width: 100%;
    margin-block-end: var(--wa-space-m);
    padding: var(--wa-space-s);
    font: inherit;
  }
</style>
