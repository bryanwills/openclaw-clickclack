<script lang="ts">
  import type { Channel } from "../../lib/types";

  type Props = {
    expanded: boolean;
    channels: Channel[];
    selectedChannelID: string;
    selectedDirectID: string;
    hrefForChannel: (channelID: string) => string;
    onSelectChannel: (channelID: string) => void;
    onCreateChannel: () => void;
    onToggle: () => void;
  };

  let {
    expanded,
    channels,
    selectedChannelID,
    selectedDirectID,
    hrefForChannel,
    onSelectChannel,
    onCreateChannel,
    onToggle,
  }: Props = $props();

  let visibleChannels = $derived(
    expanded
      ? channels
      : channels.filter(
          (channel) =>
            (channel.id === selectedChannelID && !selectedDirectID) ||
            (channel.unread_count || 0) > 0,
        ),
  );

  function shouldHandleClientNavigation(event: MouseEvent): boolean {
    return event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey;
  }
</script>

<section class="nav-section" class:collapsed={!expanded}>
  <div class="section-title">
    <button type="button" class="section-toggle" aria-expanded={expanded} aria-controls="sidebar-channels-list" onclick={onToggle}>
      <span class="caret" aria-hidden="true">▾</span>
      <span class="label">Channels</span>
    </button>
    <button
      type="button"
      class="add-button"
      aria-label="Create channel"
      title="Create channel"
      onclick={onCreateChannel}
    >＋</button>
  </div>
  <div
    class="nav-list"
    id="sidebar-channels-list"
    hidden={!expanded && visibleChannels.length === 0}
  >
    {#each visibleChannels as channel (channel.id)}
      {@const unread = channel.unread_count || 0}
      <a
        href={hrefForChannel(channel.id)}
        class="nav-item channel"
        class:active={channel.id === selectedChannelID && !selectedDirectID}
        class:has-unread={unread > 0 && !(channel.id === selectedChannelID && !selectedDirectID)}
        onclick={(event) => {
          if (!shouldHandleClientNavigation(event)) return;
          event.preventDefault();
          onSelectChannel(channel.id);
        }}
      >
        <span class="hash">#</span> <span class="nav-label">{channel.name}</span>
        {#if unread > 0 && !(channel.id === selectedChannelID && !selectedDirectID)}
          <span class="unread-badge" aria-label={`${unread} unread`}>{unread > 99 ? "99+" : unread}</span>
        {/if}
      </a>
    {/each}
    {#if expanded && channels.length === 0}
      <p class="nav-empty">No channels yet</p>
    {/if}
  </div>
</section>
