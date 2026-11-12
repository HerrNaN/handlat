<script lang="ts">
  import WaDrawer from "@awesome.me/webawesome/dist/components/drawer/drawer.js";
  import WaInput from "@awesome.me/webawesome/dist/components/input/input.js";
  import { onDestroy } from "svelte";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  import { observeKeyboardDismissal } from "../utils/keyboard";
  import { errorMessage, type NewItem } from "./api";
  import { classify, departments } from "./classifier";
  import { parseItemInput } from "./quantity";

  let input = $state<WaInput>();
  let drawer = $state<WaDrawer>();
  let stopObserving: (() => void) | undefined;

  const stop = () => {
    stopObserving?.();
    stopObserving = undefined;
  };
  const show = () => {
    stop();
    stopObserving = observeKeyboardDismissal(() => {
      if (drawer) drawer.open = false;
    });
  };
  const open = async () => {
    if (!drawer || !input) return;

    // The drawer finds its focus target by attribute; Svelte only sets the input property.
    input.setAttribute("autofocus", "");
    drawer.open = true;
    await drawer.updateComplete;
    if (drawer.open) input.focus();
  };
  onDestroy(stop);
  let { error, onAdd }: { error?: unknown; onAdd: (item: NewItem) => Promise<unknown> } = $props();

  const handleAdd = async () => {
    if (!input) return;

    const text = input.value?.trim();
    if (text) {
      // Clear before saving so the next item can be entered without losing focus.
      input.value = "";
      try {
        const item = parseItemInput(text);
        await onAdd({
          id: crypto.randomUUID(),
          ...item,
          department: classify(item.text, departments).id,
        });
      } catch {
        // The parent displays the error. Leave any newer input untouched.
      }
    }
  };

  const handleKeydown = (e: KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleAdd();
    }
  };
</script>

<div class="input-wrapper">
  <wa-drawer
    bind:this={drawer}
    placement="bottom"
    light-dismiss
    without-header
    onwa-show={show}
    onwa-hide={stop}
  >
    <wa-input
      bind:this={input}
      enterkeyhint="go"
      placeholder="t.ex. 2 kg potatis"
      onkeydown={handleKeydown}
    ></wa-input>
    {#if error}
      <p role="alert">{errorMessage(error)}</p>
    {/if}
  </wa-drawer>
  <wa-button size="l" variant="brand" onclick={open}>
    <wa-icon name="plus" label="Add item"></wa-icon>
  </wa-button>
</div>

<style>
  .input-wrapper {
    margin-block-start: auto;
    align-self: flex-end;
    flex-shrink: 0;
    position: sticky;
    bottom: 0;
    display: flex;
    justify-content: flex-end;
    align-items: flex-end;
  }

  wa-button {
    margin: 1rem;
  }

  wa-drawer {
    --size: min-content;
    --show-duration: 0;

    :global(&[open] + wa-button) {
      display: none;
    }
  }

  wa-input {
    width: 100%;
  }
</style>
