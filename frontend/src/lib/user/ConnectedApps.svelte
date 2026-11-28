<script lang="ts">
  import { createMutation, createQuery } from "@tanstack/svelte-query";
  import z from "zod";

  import { readApiError } from "../api";
  import { messages } from "../i18n";
  import { queryClient } from "../query";
  import "@awesome.me/webawesome/dist/components/button/button.js";

  import "@awesome.me/webawesome/dist/styles/utilities/visually-hidden.css";

  let { account }: { account: string } = $props();
  const url = "/api/connected-apps";
  const statusSchema = z.object({
    enabled: z.boolean(),
    apps: z.array(z.object({ id: z.string(), name: z.string(), clientId: z.url() })),
  });
  const key = $derived(["connected-apps", account]);

  const connections = createQuery(() => ({
    queryKey: key,
    queryFn: async ({ signal }) => {
      const response = await fetch(url, { signal, cache: "no-store" });
      if (!response.ok) {
        throw await readApiError(response);
      }
      return statusSchema.parse(await response.json());
    },
  }));

  const disconnect = createMutation(() => ({
    mutationFn: async (id: string) => {
      const response = await fetch(`${url}/${encodeURIComponent(id)}`, { method: "DELETE" });
      if (!response.ok) {
        throw await readApiError(response);
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: key });
    },
  }));
</script>

{#if connections.data?.enabled || connections.data?.apps.length || connections.isError}
  <section aria-labelledby="connected-apps-heading">
    <h2 id="connected-apps-heading">{$messages.connectedApps.title}</h2>
    {#if connections.isError}
      <p role="alert">{$messages.connectedApps.loadFailed}</p>
    {:else if connections.data}
      {#if connections.data.enabled}
        <p>{$messages.connectedApps.serverUrl} <code>{location.origin}/mcp</code></p>
      {:else}
        <p role="status">{$messages.connectedApps.paused}</p>
      {/if}
      {#if connections.data.apps.length}
        <ul>
          {#each connections.data.apps as app (app.id)}
            <li>
              <div>
                <strong>{app.name}</strong>
                <small dir="ltr">{new URL(app.clientId).host}</small>
              </div>
              <wa-button
                size="s"
                onclick={() => disconnect.mutate(app.id)}
                disabled={disconnect.isPending}
                loading={disconnect.isPending && disconnect.variables === app.id}
              >
                <span aria-hidden="true">{$messages.connectedApps.disconnect}</span>
                <span class="wa-visually-hidden-force">
                  {$messages.connectedApps.disconnectApp(app.name)}
                </span>
              </wa-button>
            </li>
          {/each}
        </ul>
      {:else}
        <p>{$messages.connectedApps.empty}</p>
      {/if}
    {/if}
    {#if disconnect.isError}
      <p role="alert">{$messages.connectedApps.disconnectFailed}</p>
    {/if}
  </section>
{/if}

<style>
  h2 {
    margin: 0 0 var(--wa-space-m);
  }

  ul {
    padding: 0;
    list-style: none;
  }

  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--wa-space-s);
    border-block-start: 1px solid var(--wa-color-surface-border);
    padding-block: var(--wa-space-s);
  }

  strong,
  small {
    display: block;
  }

  small,
  strong,
  code {
    overflow-wrap: anywhere;
  }
</style>
