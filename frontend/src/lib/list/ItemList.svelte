<script lang="ts">
  import { tick } from "svelte";
  import { flip } from "svelte/animate";
  import { cubicOut } from "svelte/easing";
  import "@awesome.me/webawesome/dist/components/button/button.js";

  import { departmentOrder, getDepartment } from "./classifier";
  import ItemRow from "./ItemRow.svelte";
  import type { Item } from "./list";

  interface Props {
    items: Item[];
    onClickEditItem: (item: Item) => void;
    onCheck: (itemID: string, checked: boolean) => void;
    onDelete: (itemID: string) => Promise<unknown>;
    onClear: () => void;
  }

  let { items: itemList, onClickEditItem, onCheck, onDelete, onClear }: Props = $props();

  const checkedItems = $derived(
    itemList
      .filter((item): item is Item & { checkedAt: Date } => item.checkedAt !== undefined)
      .sort((a, b) => b.checkedAt.getTime() - a.checkedAt.getTime()),
  );
  type Row =
    | { key: string; kind: "heading"; name: string; checked: boolean }
    | { key: string; kind: "item"; item: Item };

  // One keyed block preserves item identity when moving between sections.
  const rows = $derived.by(() => {
    const result: Row[] = [];
    for (const deptId of departmentOrder) {
      const items = itemList.filter(
        (item) => item.checkedAt === undefined && (item.department ?? "other") === deptId,
      );
      if (items.length === 0) continue;
      result.push({
        key: `heading:${deptId}`,
        kind: "heading",
        name: getDepartment(deptId).name,
        checked: false,
      });
      result.push(...items.map((item): Row => ({ key: item.id, kind: "item", item })));
    }
    if (checkedItems.length > 0) {
      result.push({ key: "heading:checked", kind: "heading", name: "Handlat", checked: true });
      result.push(...checkedItems.map((item): Row => ({ key: item.id, kind: "item", item })));
    }
    return result;
  });

  let list = $state<HTMLUListElement>();
  let previousChecked = new Map<string, boolean>();
  let animateMovement = false;
  $effect.pre(() => {
    const checked = new Map(itemList.map((item) => [item.id, item.checkedAt !== undefined]));
    // Observe data changes, so local optimistic updates, remote updates and rollbacks
    // all animate. Initial loading, additions and removals do not trigger movement.
    animateMovement = [...checked].some(
      ([id, value]) => previousChecked.has(id) && previousChecked.get(id) !== value,
    );
    previousChecked = checked;

    // Moving an existing DOM node can still blur its focused descendant.
    const focused = document.activeElement;
    if (animateMovement && focused instanceof HTMLElement && list?.contains(focused)) {
      void tick().then(() => {
        if (focused.isConnected && document.activeElement === document.body) {
          focused.focus({ preventScroll: true });
        }
      });
    }
  });

  const movementEnds = new WeakMap<HTMLElement, number>();
  const move = (node: HTMLElement, positions: { from: DOMRect; to: DOMRect }) => {
    const now = performance.now();
    const normalDuration = getComputedStyle(node).getPropertyValue("--wa-transition-normal").trim();
    const durationMs = parseFloat(normalDuration) * (normalDuration.endsWith("ms") ? 1 : 1000);
    // A confirming server snapshot can arrive mid-animation. Continue from the
    // current position for the remaining time instead of snapping to the end.
    const duration = window.matchMedia("(prefers-reduced-motion: reduce)").matches
      ? 0
      : animateMovement
        ? durationMs
        : Math.max(0, (movementEnds.get(node) ?? 0) - now);
    movementEnds.set(node, now + duration);
    const animation = flip(node, positions, { duration, easing: cubicOut });
    return { ...animation, css: (t: number, u: number) => `${animation.css?.(t, u)} z-index: 1;` };
  };
</script>

<ul bind:this={list}>
  {#each rows as row (row.key)}
    <li
      animate:move
      class:heading={row.kind === "heading"}
      role={row.kind === "heading" ? "presentation" : undefined}
    >
      {#if row.kind === "heading"}
        <h2>
          {row.name}
          {#if row.checked}
            <wa-button size="s" variant="brand" appearance="plain" onclick={onClear}>
              Clear
            </wa-button>
          {/if}
        </h2>
      {:else}
        <ItemRow
          item={row.item}
          {onCheck}
          {onDelete}
          onEditItem={() => onClickEditItem(row.item)}
        />
      {/if}
    </li>
  {/each}
</ul>

<style>
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  li {
    position: relative;
    min-height: 3rem;
    margin: var(--wa-space-2xs);
  }

  .heading {
    min-height: 0;
    margin: 0;
  }

  h2 {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    margin-block: var(--wa-space-xs);
    padding: 0 var(--wa-space-m);
    font-size: var(--wa-font-size-xs);
    text-transform: uppercase;
    color: var(--wa-color-neutral-on-normal);
  }

  wa-button {
    -webkit-tap-highlight-color: transparent;
  }
</style>
