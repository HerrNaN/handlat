<script lang="ts">
  import WaDrawer from "@awesome.me/webawesome/dist/components/drawer/drawer.js";
  import WaInput from "@awesome.me/webawesome/dist/components/input/input.js";
  import WaToast from "@awesome.me/webawesome/dist/components/toast/toast.js";
  import { onDestroy, onMount } from "svelte";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  import { observeKeyboardDismissal } from "../utils/keyboard";
  import { errorMessage, type AddItemResult, type NewItem } from "./api";
  import { classify, departments } from "./classifier";
  import { formatQuantity, parseItemInput } from "./quantity";

  let input = $state<WaInput>();
  let drawer = $state<WaDrawer>();
  let toast = $state<WaToast>();
  let addButton = $state<HTMLElement>();
  let stopObserving: (() => void) | undefined;

  const positionToast = () => {
    const anchor = drawer?.open ? input : addButton;
    if (!toast || !anchor) return;

    const offset = Math.max(0, window.innerHeight - anchor.getBoundingClientRect().top);
    toast.style.setProperty("--toast-bottom-offset", `${offset}px`);
  };

  onMount(() => {
    const listeners = new AbortController();
    const { signal } = listeners;
    window.visualViewport?.addEventListener("resize", positionToast, { signal });
    window.visualViewport?.addEventListener("scroll", positionToast, { signal });
    return () => listeners.abort();
  });

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
  let { error, onAdd }: { error?: unknown; onAdd: (item: NewItem) => Promise<AddItemResult> } =
    $props();

  const handleAdd = async () => {
    if (!input) return;

    const text = input.value?.trim();
    if (text) {
      // Clear before saving so the next item can be entered without losing focus.
      input.value = "";
      try {
        const item = parseItemInput(text);
        const result = await onAdd({
          id: crypto.randomUUID(),
          ...item,
          department: classify(item.text, departments).id,
        });
        const merged = result.items.find((item) => item.id === result.mergedItemId);
        if (merged?.amount) {
          positionToast();
          await toast?.create(
            `Updated ${merged.text} to ${formatQuantity(merged.amount, merged.unit)}.`,
            {
              variant: "success",
            },
          );
        }
      } catch {
        // The parent displays the error. Leave any newer input untouched.
      }
    }
  };

  const handleKeydown = (e: KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      void handleAdd();
    }
  };
</script>

<svelte:window onresize={positionToast} />

<div class="input-wrapper">
  <wa-toast bind:this={toast} placement="bottom-center"></wa-toast>
  <wa-drawer
    bind:this={drawer}
    placement="bottom"
    light-dismiss
    without-header
    onwa-show={show}
    onwa-hide={stop}
    onwa-after-show={positionToast}
    onwa-after-hide={positionToast}
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
  <wa-button bind:this={addButton} size="l" variant="brand" onclick={open}>
    <wa-icon name="plus" label="Add item"></wa-icon>
  </wa-button>
</div>

<style>
  wa-toast {
    inset-block-end: var(--toast-bottom-offset, 0px);
    max-height: calc(100dvh - var(--toast-bottom-offset, 0px));
    overflow-wrap: anywhere;
  }

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
    margin: var(--wa-space-m);
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
