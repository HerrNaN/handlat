<script lang="ts">
  import type { Snippet } from "svelte";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  let {
    icon,
    heading,
    description,
    children,
  }: {
    icon: string;
    heading: string;
    description: string;
    children?: Snippet;
  } = $props();
</script>

<section>
  <div>
    <span class:groceries={icon === "basket-shopping"} aria-hidden="true">
      <wa-icon name={icon} variant="solid"></wa-icon>
      {#if icon === "basket-shopping"}
        <wa-icon class="apple" name="apple-whole" variant="solid"></wa-icon>
        <wa-icon class="bottle" name="bottle-water" variant="solid"></wa-icon>
        <i></i>
      {/if}
      {#if children}
        <wa-icon class="badge" name="plus" variant="solid"></wa-icon>
      {/if}
    </span>
    <h3>{heading}</h3>
    <p>{description}</p>
    {#if children}
      {@render children()}
    {/if}
  </div>
</section>

<style>
  section {
    flex: 1;
    display: flex;
    padding: var(--wa-space-xl) var(--wa-space-l);
    text-align: center;
  }

  div {
    width: 100%;
    max-width: 20rem;
    margin: auto;
  }

  span {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 6rem;
    height: 6rem;
    border-radius: 50%;
    background: var(--wa-color-neutral-fill-quiet);
    color: var(--wa-color-neutral-on-quiet);
    margin-block-end: var(--wa-space-l);
  }

  wa-icon {
    font-size: 2.5rem;
  }

  .groceries {
    width: 9rem;
    height: 9rem;
  }

  .groceries wa-icon {
    position: absolute;
  }

  .groceries wa-icon:first-child {
    bottom: 0.625rem;
    font-size: 6rem;
    z-index: 1;
  }

  .apple {
    top: 1rem;
    left: 1.4rem;
    font-size: 2.25rem;
    rotate: -12deg;
  }

  .bottle {
    top: 0.75rem;
    right: 1.5rem;
    font-size: 3rem;
    color: var(--wa-color-brand-fill-loud);
    rotate: 12deg;
  }

  i,
  i::before,
  i::after {
    position: absolute;
    width: 0.2rem;
    height: 0.65rem;
    border-radius: 999px;
    background: var(--wa-color-brand-fill-loud);
  }

  i {
    top: 2rem;
    right: 0.25rem;
    rotate: 45deg;
  }

  i::before,
  i::after {
    content: "";
    top: 0.35rem;
  }

  i::before {
    left: -0.65rem;
    rotate: -35deg;
  }

  i::after {
    right: -0.65rem;
    rotate: 35deg;
  }

  .badge {
    position: absolute;
    right: 0;
    bottom: 0;
    width: 1rem;
    min-width: 1rem;
    height: 1rem;
    font-size: 1rem;
    padding: var(--wa-space-xs);
    border-radius: 50%;
    color: var(--wa-color-brand-on-loud);
    background: var(--wa-color-brand-fill-loud);
  }

  h3 {
    margin: 0;
    font-size: var(--wa-font-size-l);
    color: var(--wa-color-text-normal);
  }

  p {
    margin-block: var(--wa-space-s) var(--wa-space-l);
    color: var(--wa-color-text-quiet);
    line-height: var(--wa-line-height-normal);
    text-wrap: pretty;
  }
</style>
