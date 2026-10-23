<script lang="ts">
    import WaInput from "@awesome.me/webawesome/dist/components/input/input.js";
    import "@awesome.me/webawesome/dist/components/drawer/drawer.js";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";
    import { errorMessage, type NewItem } from "./api";
    import { classify, departments } from "./classifier";

    let input = $state<WaInput>();
    let { busy, error, onAdd }: { busy: boolean; error?: unknown; onAdd: (item: NewItem) => Promise<unknown> } = $props();
    let draft: { text: string; id: string } | undefined;

    const handleAdd = async () => {
        if (!input || busy) return;

        const text = input.value?.trim();
        if (text) {
            if (draft?.text !== text) draft = { text, id: crypto.randomUUID() };
            try {
                await onAdd({ ...draft, department: classify(text, departments).id });
                input.value = "";
                draft = undefined;
            } catch {
                // Keep the draft UUID so retrying a lost response cannot duplicate the item.
            }
        }
    };

    const handleKeydown = (e: KeyboardEvent) => {
        if (e.key === "Enter") {
            handleAdd();
        }
    };

    const id = $props.id();
</script>

<div class="input-wrapper">
    <wa-drawer
        {id}
        placement="bottom"
        light-dismiss
        without-header
        onwa-show={() => setTimeout(() => input?.focus())}
    >
        <wa-input
            bind:this={input}
            enterkeyhint="go"
            placeholder="t.ex. 2 kg potatis"
            disabled={busy}
            onkeydown={handleKeydown}
        ></wa-input>
        {#if error}
            <p role="alert">{errorMessage(error)}</p>
        {/if}
    </wa-drawer>
    <wa-button size="l" variant="brand" data-drawer="open {id}">
        <wa-icon name="plus"></wa-icon>
    </wa-button>
</div>

<style>
    .input-wrapper {
        height: 100%;
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
