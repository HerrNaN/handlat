<script lang="ts">
    import { observer } from "./main";
    import "./router";
    import { Router } from "sv-router";

    const onerror = (error: unknown) => console.error(error)

</script>

<svelte:boundary {onerror}>
    <Router />
</svelte:boundary>
