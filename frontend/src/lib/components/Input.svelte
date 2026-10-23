<script lang="ts">
    import WaInput from "@awesome.me/webawesome/dist/components/input/input.js";
    import type { WaInputProps } from "@awesome.me/webawesome/dist/custom-elements-jsx.d.ts";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import "@awesome.me/webawesome/dist/components/icon/icon.js";

    type Props = {
        value: string;
        onsave: (value: string) => Promise<boolean>;
    } & WaInputProps;

    const { value, onsave, ...waProps }: Props = $props();

    let editing = $state(false);
    let input = $state<WaInput>();
    let saving = $state(false);

    const commitChange = async () => {
        if (!editing || saving) return;
        if (input && input.value !== null) {
            if (input.value === value) { editing = false; return; }
            saving = true;
            if (await onsave(input.value)) editing = false;
            saving = false;
        }
    };
</script>

<div>
    {#if editing}
        <wa-input
            bind:this={input}
            {...waProps}
            onchange={() => commitChange()}
            onblur={() => commitChange()}
            {value}
            disabled={saving || waProps.disabled}
        ></wa-input>
    {:else}
        {value}
        <wa-button
            appearance="plain"
            disabled={waProps.disabled}
            onclick={() => {
                editing = true;
                setTimeout(() => {
                    if (input) input.focus();
                });
            }}
        >
            <wa-icon name="pen" label="Edit list name"></wa-icon>
        </wa-button>
    {/if}
</div>

<style>
    div {
        display: flex;
        align-items: center;

        gap: var(--wa-space-s);
    }
</style>
