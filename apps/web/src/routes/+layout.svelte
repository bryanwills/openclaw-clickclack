<script lang="ts">
  import { onMount } from "svelte";
  import { initAppearance } from "$lib/appearance";
  import "../styles/index.css";

  let { children } = $props();

  onMount(() => {
    initAppearance();
  });
</script>

{@render children()}
