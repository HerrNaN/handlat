<script lang="ts">
  import WaDialog from "@awesome.me/webawesome/dist/components/dialog/dialog.js";
  import WaDropdownItem from "@awesome.me/webawesome/dist/components/dropdown-item/dropdown-item.js";
  import WaInput from "@awesome.me/webawesome/dist/components/input/input.js";
  import type { WaSelectEvent } from "@awesome.me/webawesome/dist/events/select.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/dropdown/dropdown.js";
  import { createMutation } from "@tanstack/svelte-query";
  import { tick } from "svelte";

  import { messages } from "../i18n";
  import { errorMessage } from "../i18n/errors";
  import { session } from "../user/session";
  import type { ListSummary } from "./api";
  import { leaveListMutation, renameListMutation } from "./queries";
  import ShareList from "./ShareList.svelte";

  const {
    list,
    onleave,
    onSetDefault,
  }: {
    list: Pick<ListSummary, "id" | "name" | "memberCount">;
    onleave?: () => void;
    onSetDefault?: () => void;
  } = $props();
  const rename = createMutation(() => renameListMutation());
  const departure = createMutation(() => leaveListMutation());
  let renameDialog = $state<WaDialog>();
  let leaveDialog = $state<WaDialog>();
  let nameInput = $state<WaInput>();
  let name = $state("");
  let saving = $state(false);
  const shared = $derived(list.memberCount > 1);

  const openRename = async () => {
    rename.reset();
    name = list.name;
    await tick();
    if (renameDialog) {
      renameDialog.open = true;
    }
  };
  const saveName = async (event: SubmitEvent) => {
    event.preventDefault();
    if (!$session || saving || !name.trim()) {
      return;
    }
    saving = true;
    try {
      await rename.mutateAsync({ account: $session.id, listID: list.id, name: name.trim() });
    } catch {
      // Keep the dialog open so the user can retry.
      return;
    } finally {
      saving = false;
    }
    if (renameDialog) {
      renameDialog.open = false;
    }
  };
  const confirmLeave = () => {
    departure.reset();
    if (leaveDialog) {
      leaveDialog.open = true;
    }
  };
  const leave = async () => {
    if (!$session || departure.isPending) {
      return;
    }
    try {
      await departure.mutateAsync({ account: $session.id, listID: list.id });
      if (leaveDialog) {
        leaveDialog.open = false;
      }
      onleave?.();
    } catch {
      // Keep the dialog open so the user can retry.
    }
  };
</script>

<ShareList listID={list.id} name={list.name}>
  {#snippet children(share)}
    <wa-dropdown
      placement="bottom-end"
      onwa-select={(event: WaSelectEvent) => {
        const action = (event.detail.item as WaDropdownItem).value;
        if (action === "rename") void openRename();
        if (action === "share") void share();
        if (action === "set-default") onSetDefault?.();
        if (action === "leave") confirmLeave();
      }}
    >
      <wa-button slot="trigger" appearance="plain">
        <wa-icon name="ellipsis-vertical" label={$messages.lists.actionsFor(list.name)}></wa-icon>
      </wa-button>
      <wa-dropdown-item value="rename">
        <wa-icon slot="icon" name="pen"></wa-icon>
        {$messages.lists.rename}
      </wa-dropdown-item>
      <wa-dropdown-item value="share">
        <wa-icon slot="icon" name="share-nodes"></wa-icon>
        {$messages.lists.share}
      </wa-dropdown-item>
      {#if onSetDefault}
        <wa-dropdown-item value="set-default">
          <wa-icon slot="icon" name="star"></wa-icon>
          {$messages.lists.setDefault}
        </wa-dropdown-item>
      {/if}
      <wa-dropdown-item value="leave" variant="danger">
        <wa-icon
          slot="icon"
          name={shared ? "arrow-right-from-bracket" : "trash-can"}
          variant="solid"
        ></wa-icon>
        {shared ? $messages.lists.leave : $messages.lists.delete}
      </wa-dropdown-item>
    </wa-dropdown>
  {/snippet}
</ShareList>

<wa-dialog
  bind:this={renameDialog}
  label={$messages.lists.renameList}
  onwa-after-show={() => {
    nameInput?.focus();
    nameInput?.select();
  }}
>
  <form onsubmit={saveName}>
    <wa-input
      bind:this={nameInput}
      label={$messages.lists.name}
      value={name}
      oninput={() => {
        name = nameInput?.value ?? "";
      }}
      required
    ></wa-input>
    {#if rename.error}<p role="alert">
        {errorMessage(rename.error, $messages, $messages.errors.renameList)}
      </p>{/if}
    <footer>
      <wa-button data-dialog="close">{$messages.common.cancel}</wa-button>
      <wa-button type="submit" variant="brand" disabled={!name.trim()}
        >{$messages.common.save}</wa-button
      >
    </footer>
  </form>
</wa-dialog>

<wa-dialog
  bind:this={leaveDialog}
  label={shared ? $messages.lists.confirmLeave : $messages.lists.confirmDelete}
>
  <p>
    {shared
      ? $messages.lists.leaveDescription(list.name)
      : $messages.lists.deleteDescription(list.name)}
  </p>
  {#if departure.error}<p role="alert">
      {errorMessage(
        departure.error,
        $messages,
        shared ? $messages.errors.leaveList : $messages.errors.deleteList,
      )}
    </p>{/if}
  <wa-button slot="footer" data-dialog="close">{$messages.common.cancel}</wa-button>
  <wa-button slot="footer" variant="danger" onclick={leave}
    >{shared ? $messages.lists.leave : $messages.lists.delete}</wa-button
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
