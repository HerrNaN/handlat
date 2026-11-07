<script lang="ts">
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import type { PathParams } from "sv-router";
  import type { Snippet } from "svelte";

  import { navigate, type p } from "../../router";

  type RouterPath = Parameters<typeof p>[0];

  type Props = {
    [Path in RouterPath]: {
      children?: Snippet;
      to: Path;
    } & (PathParams<Path> extends never ? { params?: never } : { params: PathParams<Path> });
  }[RouterPath];

  const { children, to, params }: Props = $props();

  const onclick = () => {
    if (params) {
      return navigate(to, { params });
    }

    return navigate(to);
  };
</script>

<wa-button {onclick}>
  {@render children?.()}
</wa-button>

<style>
  wa-button {
    &::part(base) {
      background-color: transparent;
      color: var(--wa-color-text-link);
    }

    &::part(label) {
      text-decoration: underline dotted;
      text-underline-offset: 3px;
    }

    &:hover::part(label) {
      text-decoration-style: solid;
    }
  }
</style>
