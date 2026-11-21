<script lang="ts">
  import { tick } from "svelte";

  import { messages } from "../i18n";
  import "@awesome.me/webawesome/dist/components/button/button.js";

  import { departmentOrder } from "./classifier";
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
      if (items.length === 0) {
        continue;
      }
      result.push({
        key: `heading:${deptId}`,
        kind: "heading",
        name: $messages.departments[deptId],
        checked: false,
      });
      result.push(...items.map((item): Row => ({ key: item.id, kind: "item", item })));
    }
    if (checkedItems.length > 0) {
      result.push({
        key: "heading:checked",
        kind: "heading",
        name: $messages.items.checked,
        checked: true,
      });
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

  const move = (node: HTMLElement, { from }: { from: DOMRect; to: DOMRect }) => {
    if (animateMovement) {
      // Measure the destination without an in-progress transition, then let CSS
      // transition from the previous visual position to the new layout position.
      node.style.transition = "none";
      node.style.translate = "none";
      const to = node.getBoundingClientRect();
      node.style.translate = `${from.left - to.left}px ${from.top - to.top}px`;
      node.getBoundingClientRect();
      node.style.transition = "";
      node.style.translate = "";
    }
    // CSS owns playback; Svelte only measures and reconciles the keyed rows.
    return { duration: 0 };
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
              {$messages.items.clear}
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
    transition: translate var(--wa-transition-normal) var(--wa-transition-easing);
    min-height: 3rem;
    margin: var(--wa-space-2xs);
  }

  @media (prefers-reduced-motion: reduce) {
    li {
      transition: none;
    }
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
    color: var(--wa-color-text-quiet);
  }

  wa-button {
    -webkit-tap-highlight-color: transparent;
  }
</style>
