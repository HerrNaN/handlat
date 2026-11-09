<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import WaInput from "@awesome.me/webawesome/dist/components/input/input.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/dropdown/dropdown.js";
  import "@awesome.me/webawesome/dist/components/dropdown-item/dropdown-item.js";
  import type { WaSelectEvent } from "@awesome.me/webawesome/dist/events/select.js";
  import { createMutation } from "@tanstack/svelte-query";
  import { tick } from "svelte";

  import { session } from "../user/session";
  import { errorMessage, type ListSummary } from "./api";
  import { leaveListMutation, renameListMutation } from "./queries";
  import ShareList from "./ShareList.svelte";

  const {
    list,
    onleave,
  }: {
    list: Pick<ListSummary, "id" | "name" | "memberCount">;
    onleave?: () => void;
  } = $props();
  const rename = createMutation(() => renameListMutation());
  const departure = createMutation(() => leaveListMutation());
  let renameDialog = $state<WaDialog>();
  let leaveDialog = $state<WaDialog>();
  let nameInput = $state<WaInput>();
  let name = $state("");
  const shared = $derived(list.memberCount > 1);

  const openRename = async () => {
    rename.reset();
    name = list.name;
    await tick();
    if (renameDialog) renameDialog.open = true;
  };
  const saveName = async (event: SubmitEvent) => {
    event.preventDefault();
    if (!$session || rename.isPending || !name.trim()) return;
    try {
      await rename.mutateAsync({ account: $session.id, listID: list.id, name: name.trim() });
      if (renameDialog) renameDialog.open = false;
    } catch {
      // Keep the dialog open so the user can retry.
    }
  };
  const confirmLeave = () => {
    departure.reset();
    if (leaveDialog) leaveDialog.open = true;
  };
  const leave = async () => {
    if (!$session || departure.isPending) return;
    try {
      await departure.mutateAsync({ account: $session.id, listID: list.id });
      if (leaveDialog) leaveDialog.open = false;
      onleave?.();
    } catch {
      // Keep the dialog open so the user can retry.
    }
  };
</script>

<ShareList listID={list.id} name={list.name}>
  {#snippet children(share, sharing)}
    <wa-dropdown
      placement="bottom-end"
      onwa-select={(event: WaSelectEvent) => {
        const action = event.detail.item.getAttribute("value");
        if (action === "rename") void openRename();
        if (action === "share") void share();
        if (action === "leave") confirmLeave();
      }}
    >
      <wa-button slot="trigger" appearance="plain" disabled={sharing || departure.isPending}>
        <wa-icon name="ellipsis-vertical" label={`Actions for ${list.name}`}></wa-icon>
      </wa-button>
      <wa-dropdown-item value="rename">Rename</wa-dropdown-item>
      <wa-dropdown-item value="share">Share</wa-dropdown-item>
      <wa-dropdown-item value="leave" variant="danger">
        {shared ? "Leave list" : "Delete list"}
      </wa-dropdown-item>
    </wa-dropdown>
  {/snippet}
</ShareList>

<wa-dialog
  bind:this={renameDialog}
  label="Rename list"
  onwa-after-show={() => {
    nameInput?.focus();
    nameInput?.select();
  }}
  onwa-hide={(event: Event) => {
    if (rename.isPending) event.preventDefault();
  }}
>
  <form onsubmit={saveName}>
    <wa-input
      bind:this={nameInput}
      label="List name"
      value={name}
      oninput={() => {
        name = nameInput?.value ?? "";
      }}
      required
      disabled={rename.isPending}
    ></wa-input>
    {#if rename.error}<p role="alert">{errorMessage(rename.error)}</p>{/if}
    <footer>
      <wa-button data-dialog="close" disabled={rename.isPending}>Cancel</wa-button>
      <wa-button
        type="submit"
        variant="brand"
        disabled={rename.isPending || !name.trim()}
        loading={rename.isPending}>Save</wa-button
      >
    </footer>
  </form>
</wa-dialog>

<wa-dialog
  bind:this={leaveDialog}
  label={shared ? "Leave list?" : "Delete list?"}
  onwa-hide={(event: Event) => {
    if (departure.isPending) event.preventDefault();
  }}
>
  {#if shared}
    <p>Leave “{list.name}”? Other members will keep the list.</p>
  {:else}
    <p>
      You're the only member. Leaving will delete “{list.name}” and all its items. This cannot be
      undone.
    </p>
  {/if}
  {#if departure.error}<p role="alert">{errorMessage(departure.error)}</p>{/if}
  <wa-button slot="footer" data-dialog="close" disabled={departure.isPending}>Cancel</wa-button>
  <wa-button
    slot="footer"
    variant="danger"
    onclick={leave}
    disabled={departure.isPending}
    loading={departure.isPending}>{shared ? "Leave list" : "Delete list"}</wa-button
  >
</wa-dialog>

<style>
  footer {
    display: flex;
    justify-content: flex-end;
    gap: var(--wa-space-s);
    margin-block-start: var(--wa-space-l);
  }
</style>
