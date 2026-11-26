<script lang="ts">
  import { type Snippet } from "svelte";

  import { messages } from "./lib/i18n";
  import { navigateToDefaultList } from "./lib/list/navigation";

  import "@awesome.me/webawesome/dist/components/icon/icon.js";
  import "@awesome.me/webawesome/dist/components/tab-group/tab-group.js";
  import "@awesome.me/webawesome/dist/components/tab-panel/tab-panel.js";
  import "@awesome.me/webawesome/dist/components/tab/tab.js";
  import "@awesome.me/webawesome/dist/components/button/button.js";
  import "@awesome.me/webawesome/dist/components/popover/popover.js";

  import { session } from "./lib/user/session";
  import { navigate, route } from "./router";

  let { children }: { children?: Snippet } = $props();
  const recipesEnabled = $derived($session?.recipesEnabled === true);

  $effect(() => {
    if (
      $session === null &&
      route.pathname !== "/login" &&
      !route.pathname.startsWith("/invitations/")
    ) {
      void navigate("/login", { replace: true });
    }
  });
</script>

<main>
  <div class="wrapper">
    {#if children}
      {@render children()}
    {/if}
  </div>

  <nav>
    <ul>
      <li class:active={recipesEnabled && route.pathname.startsWith("/recipes")}>
        {#if recipesEnabled}
          <wa-button onclick={() => navigate("/recipes")} appearance="plain">
            <wa-icon name="kitchen-set" label={$messages.nav.recipes}></wa-icon>
          </wa-button>
        {:else}
          <wa-button id="recipes-nav" appearance="plain">
            <wa-icon name="kitchen-set" label={$messages.nav.recipesInfo}></wa-icon>
          </wa-button>
          <wa-popover for="recipes-nav" placement="top-start">
            <strong>
              <wa-icon name="flask" aria-hidden="true"></wa-icon>
              {$messages.nav.recipesExperimental}
            </strong>
            <span>{$messages.nav.recipesEnableInProfile}</span>
          </wa-popover>
        {/if}
      </li>
      <li class:active={route.pathname.startsWith("/lists")}>
        <wa-button onclick={navigateToDefaultList} appearance="plain" disabled={!$session}>
          <wa-icon name="list-check" label={$messages.lists.title}></wa-icon>
        </wa-button>
      </li>
      <li class:active={route.pathname.startsWith("/user")}>
        <wa-button
          onclick={() => navigate("/user")}
          aria-label={$messages.nav.user}
          appearance="plain"
          disabled={!$session}
        >
          <wa-icon name="user"></wa-icon>
        </wa-button>
      </li>
    </ul>
  </nav>
</main>

<style>
  :global(body) {
    margin: 0;
    -webkit-tap-highlight-color: transparent;
  }

  main {
    height: 100dvh;
    max-height: 100svh;
    display: flex;
    flex-direction: column;
  }

  .wrapper {
    flex: 1;
    overflow-y: auto;
  }

  nav {
    height: 4rem;

    #recipes-nav wa-icon {
      opacity: 0.5;
    }

    wa-popover {
      &::part(body) {
        padding: var(--wa-space-s) var(--wa-space-m);
      }

      strong {
        display: flex;
        align-items: center;
        gap: var(--wa-space-xs);
      }
    }

    wa-button {
      width: 100%;
      height: 100%;

      &::part(base) {
        width: 100%;
        height: 100%;
        background-color: transparent;
      }
    }

    ul {
      width: 100%;
      height: 100%;
      padding: 0;
      margin: 0;
      list-style: none;
      display: flex;

      li {
        width: 100%;
        height: 100%;
        flex: 1;
        display: flex;
        justify-content: center;
        align-items: center;

        &.active {
          wa-icon {
            color: var(--wa-color-brand-on-normal);
            padding: var(--wa-space-m) var(--wa-space-xl);
            border-radius: var(--wa-border-radius-l);
            background-color: var(--wa-color-brand-fill-normal);
          }
        }
      }
    }
  }
</style>
