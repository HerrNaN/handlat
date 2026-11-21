<script lang="ts">
  import { route } from "../../router";
  import { session } from "../user/session";
  import RecipePage from "./RecipePage.svelte";
</script>

{#if $session && route.params.recipeId}
  {#key `${$session.id}:${route.params.recipeId}`}
    <RecipePage id={route.params.recipeId} account={$session.id} />
  {/key}
{/if}
