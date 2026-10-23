<script lang="ts">
    import { type Snippet } from "svelte";
    import { navigate, route } from "./router";

    import "@awesome.me/webawesome/dist/components/icon/icon.js";
    import "@awesome.me/webawesome/dist/components/tab-group/tab-group.js";
    import "@awesome.me/webawesome/dist/components/tab-panel/tab-panel.js";
    import "@awesome.me/webawesome/dist/components/tab/tab.js";
    import "@awesome.me/webawesome/dist/components/button/button.js";
    import { navigateToDefaultList } from "./lib/list/navigation";
    import { session } from "./lib/user/session";

    let { children }: { children?: Snippet } = $props();

    $effect(() => {
        if ($session === null && route.pathname !== "/sign-in" && route.pathname !== "/sign-up") {
            void navigate("/sign-in", { replace: true });
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
            <li>
                <wa-button
                    href="/recipes"
                    aria-label="Recipes (soon!)"
                    appearance="plain"
                    disabled
                >
                    <wa-icon name="kitchen-set"></wa-icon>
                </wa-button>
            </li>
            <li class:active={route.pathname.startsWith("/lists")}>
                <wa-button
                    onclick={navigateToDefaultList}
                    aria-label="Lists"
                    appearance="plain"
                    disabled={!$session}
                >
                    <wa-icon name="list-check"></wa-icon>
                </wa-button>
            </li>
            <li class:active={route.pathname.startsWith("/user")}>
                <wa-button
                    onclick={() => navigate("/user")}
                    aria-label="User"
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
                        color: var(--wa-color-brand);
                        padding: 1rem 2rem;
                        border-radius: var(--wa-border-radius-l);
                        background-color: var(--wa-color-brand-fill-normal);
                    }
                }
            }
        }
    }
</style>
