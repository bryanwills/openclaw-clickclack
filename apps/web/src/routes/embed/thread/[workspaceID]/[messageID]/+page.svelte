<script lang="ts">
  import EmbedThreadView from "../../../../../components/embed/EmbedThreadView.svelte";

  let { params } = $props();
</script>

{#key `${params.workspaceID}:${params.messageID}`}
  <EmbedThreadView
    workspaceRouteID={params.workspaceID}
    messageRouteID={params.messageID}
  />
{/key}
