<script lang="ts">
  import type { Channel } from "../../lib/types";

  type Props = {
    channels: Channel[];
    selectedChannelID: string;
    selectedDirectID: string;
    channelName: string;
    onSelectChannel: (channelID: string) => void;
    onChannelName: (value: string) => void;
    onCreateChannel: () => void;
  };

  let {
    channels,
    selectedChannelID,
    selectedDirectID,
    channelName,
    onSelectChannel,
    onChannelName,
    onCreateChannel,
  }: Props = $props();
</script>

<section class="nav-section">
  <div class="section-title">
    <span class="caret" aria-hidden="true">▾</span>
    <span class="label">Channels</span>
  </div>
  <div class="nav-list">
    {#each channels as channel (channel.id)}
      <button
        class="nav-item channel"
        class:active={channel.id === selectedChannelID && !selectedDirectID}
        onclick={() => onSelectChannel(channel.id)}
      >
        <span class="hash">#</span> <span class="nav-label">{channel.name}</span>
      </button>
    {/each}
    {#if channels.length === 0}
      <p class="nav-empty">No channels yet</p>
    {/if}
  </div>
  <form
    class="inline-create"
    onsubmit={(event) => {
      event.preventDefault();
      onCreateChannel();
    }}
  >
    <input
      value={channelName}
      placeholder="add-channel"
      aria-label="New channel name"
      oninput={(event) => onChannelName(event.currentTarget.value)}
    />
    <button type="submit" class="ghost" aria-label="Create channel">＋</button>
  </form>
</section>
