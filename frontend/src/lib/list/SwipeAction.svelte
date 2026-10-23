<script lang="ts" module>
    const THRESHOLD = 60;
    const DEAD_ZONE = 10;
</script>

<script lang="ts">
    import type { Snippet } from "svelte";

    type Props = {
        ontrigger: () => void;
        disabled?: boolean;
        underlay?: Snippet;
        children: Snippet;
    };

    let {
        ontrigger,
        disabled = false,
        underlay,
        children,
    }: Props = $props();

    let isDragging = $state(false);
    let startX = 0;
    let currentX = $state(0);
    let contentEl = $state<HTMLElement | null>(null);

    export const reset = () => {
        isDragging = false;
        currentX = 0;
    };

    const handlePointerDown = (e: PointerEvent) => {
        if (disabled) return;
        startX = e.clientX;
        isDragging = true;
    };

    const handlePointerMove = (e: PointerEvent) => {
        if (!isDragging || !contentEl) return;
        const delta = startX - e.clientX;
        if (Math.abs(delta) <= DEAD_ZONE) {
            currentX = 0;
            return;
        }
        // Capture only after a drag starts. Capturing on pointerdown retargets
        // ordinary button clicks to this wrapper and prevents item toggles.
        (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
        const adjusted = delta > 0 ? delta - DEAD_ZONE : delta + DEAD_ZONE;
        currentX = -Math.max(0, Math.min(adjusted, contentEl.offsetWidth));
    };

    const handlePointerUp = () => {
        if (!isDragging) return;
        isDragging = false;
        const width = contentEl?.offsetWidth ?? 0;

        if (Math.abs(currentX) > THRESHOLD) {
            const targetX = -width;
            const willTransition = currentX !== targetX;
            currentX = targetX;
            if (!willTransition) {
                ontrigger();
            }
        } else {
            currentX = 0;
        }
    };

    const handleTransitionEnd = (e: TransitionEvent) => {
        if (e.propertyName !== "transform") return;
        if (currentX < 0) {
            ontrigger();
        }
    };

    const handlePointerCancel = () => {
        isDragging = false;
        currentX = 0;
    };
</script>

<div
    class="action"
    role="none"
    onpointerdown={handlePointerDown}
    onpointermove={handlePointerMove}
    onpointerup={handlePointerUp}
    onpointercancel={handlePointerCancel}
>
    <div class="underlay">
        {@render underlay?.()}
    </div>

    <div
        class="content"
        class:snapping={!isDragging}
        class:off-screen={!isDragging && currentX < 0}
        style="transform: translateX({currentX}px)"
        bind:this={contentEl}
        ontransitionend={handleTransitionEnd}
    >
        {@render children()}
    </div>
</div>

<style>
    .action {
        position: relative;
        width: 100%;
        height: 100%;
        touch-action: pan-y;
        user-select: none;
    }

    .underlay {
        position: absolute;
        inset: 0;
        display: flex;
        align-items: center;
    }

    .content {
        position: relative;
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;

        &.snapping {
            transition: transform var(--wa-transition-normal) ease;
        }

        &.off-screen {
            pointer-events: none;
        }
    }
</style>
