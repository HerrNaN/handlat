<script lang="ts">
    import { observer } from "./main";
    import "./router";
    import { Router } from "sv-router";
    import { QueryClientProvider } from "@tanstack/svelte-query";
    import { queryClient } from "./lib/query";

    const onerror = (error: unknown) => console.error(error)

</script>

<svelte:boundary {onerror}>
    <QueryClientProvider client={queryClient}>
        <Router />
    </QueryClientProvider>
</svelte:boundary>
