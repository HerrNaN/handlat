<script lang="ts">
  import { QueryClientProvider } from "@tanstack/svelte-query";

  import "./router";
  import { Router } from "sv-router";

  import { queryClient } from "./lib/query";
  import { observer } from "./main";

  const onerror = (error: unknown) => console.error(error);
</script>

<svelte:boundary {onerror}>
  <QueryClientProvider client={queryClient}>
    <Router />
  </QueryClientProvider>
</svelte:boundary>
