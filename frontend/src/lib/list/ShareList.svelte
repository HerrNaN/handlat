<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import type { Snippet } from "svelte";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  import { p } from "../../router";
  import { createInvitation, errorMessage } from "./api";

  const {
    listID,
    name,
    children,
  }: {
    listID: string;
    name: string;
    children: Snippet<[() => Promise<void>, boolean]>;
  } = $props();
  let dialog = $state<WaDialog>();
  let pending = $state(false);
  let link = $state("");
  let error = $state("");
  let copied = $state(false);

  const shareData = () => ({
    title: name,
    text: `You're invited to join “${name}” on Handlat. This invitation is valid for 24 hours.`,
    url: link,
  });
  const shareNative = async () => {
    try {
      await navigator.share(shareData());
      if (dialog) dialog.open = false;
    } catch (cause) {
      if (cause instanceof Error && cause.name === "AbortError") return;
      error = "Unable to open sharing. Copy the link instead.";
    }
  };

  const share = async () => {
    if (pending) return;
    pending = true;
    link = "";
    error = "";
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
      error = errorMessage(cause);
    } finally {
      pending = false;
    }
    if (dialog) dialog.open = true;
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(link);
      copied = true;
      error = "";
    } catch {
      error = "Select and copy the invitation link above.";
    }
  };
</script>

{@render children(share, pending)}

<wa-dialog bind:this={dialog} label="Share list" light-dismiss>
  {#if link}
    <p>
      Send this invitation to one person. It expires 24 hours after creation and can be accepted
      once.
    </p>
    <input
      aria-label="Invitation link"
      value={link}
      readonly
      onclick={(event) => event.currentTarget.select()}
    />
    {#if typeof navigator.share === "function"}<wa-button onclick={shareNative}
        >Share invitation</wa-button
      >{/if}
    <wa-button onclick={copy}>Copy link</wa-button>
    {#if copied}<p role="status">Link copied.</p>{/if}
  {/if}
  {#if error}<p role="alert">{error}</p>{/if}
  <wa-button slot="footer" data-dialog="close">Done</wa-button>
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
