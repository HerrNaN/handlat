<script lang="ts">
  import type { Snippet } from "svelte";
  import "@awesome.me/webawesome/dist/components/icon/icon.js";

  let {
    icon,
    illustration,
    heading,
    description,
    children,
  }: {
    icon?: string;
    illustration?: string;
    heading: string;
    description: string;
    children?: Snippet;
  } = $props();
</script>

<section>
  <div>
    {#if illustration}
      <img src={illustration} alt="" width="320" height="320" />
    {:else}
      <span aria-hidden="true">
        <wa-icon name={icon} variant="solid"></wa-icon>
        {#if children}
          <wa-icon class="badge" name="plus" variant="solid"></wa-icon>
        {/if}
      </span>
    {/if}
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

  img {
    display: block;
    width: 12rem;
    max-width: 100%;
    height: auto;
    margin: 0 auto var(--wa-space-l);
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
