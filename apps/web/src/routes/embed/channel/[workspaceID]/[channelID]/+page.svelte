<script lang="ts">
  import EmbedChannelView from "../../../../../components/embed/EmbedChannelView.svelte";

  let { params } = $props();
</script>

{#key `${params.workspaceID}:${params.channelID}`}
  <EmbedChannelView
    workspaceRouteID={params.workspaceID}
    channelRouteID={params.channelID}
  />
{/key}
