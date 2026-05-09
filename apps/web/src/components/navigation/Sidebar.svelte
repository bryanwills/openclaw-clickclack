<script lang="ts">
  import {
    avatarHue,
    avatarInitial,
    directConversationForUser,
    handleLabel,
  } from "../../lib/chat/people";
  import type { Channel, DirectConversation, User } from "../../lib/types";
  import ChannelList from "./ChannelList.svelte";
  import DirectMessageList from "./DirectMessageList.svelte";

  type Props = {
    workspaceName?: string;
    status: string;
    connected: boolean;
    sidebarCollapsed: boolean;
    channels: Channel[];
    directConversations: DirectConversation[];
    recentPeople: User[];
    currentUser: User | null;
    selectedChannelID: string;
    selectedDirectID: string;
    selectedProfile: User | null;
    onToggleCollapse: () => void;
    onSelectChannel: (channelID: string) => void;
    onCreateChannel: () => void;
    onSelectDirect: (conversationID: string) => void;
    onCreateDirect: () => void;
    onOpenProfile: (profile: User) => void;
    onOpenSettings: () => void;
  };

  let {
    workspaceName,
    status,
    connected,
    sidebarCollapsed,
    channels,
    directConversations,
    recentPeople,
    currentUser,
    selectedChannelID,
    selectedDirectID,
    selectedProfile,
    onToggleCollapse,
    onSelectChannel,
    onCreateChannel,
    onSelectDirect,
    onCreateDirect,
    onOpenProfile,
    onOpenSettings,
  }: Props = $props();
</script>

<aside class="sidebar" aria-label="Channels and DMs">
  <header class="workspace-header">
    <div class="workspace-name">
      <strong>{workspaceName || "Pick a workspace"}</strong>
      <span class="presence" class:online={connected}>{connected ? "Connected" : status}</span>
    </div>
    <button
      type="button"
      class="sidebar-collapse"
      aria-label={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
      title={sidebarCollapsed ? "Expand sidebar" : "Collapse sidebar"}
      onclick={onToggleCollapse}
    >
      <svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true">
        <path
          fill="none"
          stroke="currentColor"
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d={sidebarCollapsed ? "m9 6 6 6-6 6" : "m15 6-6 6 6 6"}
        />
      </svg>
    </button>
  </header>

  <div class="sidebar-scroll">
    <ChannelList
      {channels}
      {selectedChannelID}
      {selectedDirectID}
      {onSelectChannel}
      {onCreateChannel}
    />

    <DirectMessageList
      conversations={directConversations}
      currentUserID={currentUser?.id}
      {selectedDirectID}
      {onSelectDirect}
      {onCreateDirect}
    />

    <section class="nav-section">
      <div class="section-title">
        <span class="caret" aria-hidden="true">▾</span>
        <span class="label">People</span>
      </div>
      <div class="nav-list">
        {#each recentPeople as person (person.id)}
          {@const conversation = directConversationForUser(directConversations, person.id)}
          <button
            class="nav-item dm"
            class:active={conversation?.id === selectedDirectID || selectedProfile?.id === person.id}
            onclick={() => {
              if (conversation) onSelectDirect(conversation.id);
              else onOpenProfile(person);
            }}
          >
            <span class="dm-avatar" style="--hue: {avatarHue(person.id)}deg">
              {#if person.avatar_url}
                <img src={person.avatar_url} alt="" loading="lazy" />
              {:else}
                {avatarInitial(person.display_name)}
              {/if}
            </span>
            <span class="nav-label">{person.display_name}</span>
            <span class="presence-dot active" aria-hidden="true"></span>
          </button>
        {/each}
        {#if recentPeople.length === 0}
          <p class="nav-empty">People appear here as you chat</p>
        {/if}
      </div>
    </section>
  </div>

  {#if currentUser}
    <button
      class="user-card"
      type="button"
      onclick={onOpenSettings}
      oncontextmenu={(event) => {
        event.preventDefault();
        onOpenSettings();
      }}
      aria-label={`Account settings for ${currentUser.display_name} ${handleLabel(currentUser.handle)}`}
    >
      <span class="dm-avatar" style="--hue: {avatarHue(currentUser.id)}deg">
        {#if currentUser.avatar_url}
          <img src={currentUser.avatar_url} alt="" loading="lazy" />
        {:else}
          {avatarInitial(currentUser.display_name)}
        {/if}
      </span>
      <div class="user-meta">
        <strong>{currentUser.display_name}</strong>
        <span>{currentUser.handle ? handleLabel(currentUser.handle) : connected ? "Active" : "Reconnecting…"}</span>
      </div>
      <span class="presence-dot active" aria-hidden="true"></span>
    </button>
  {/if}
</aside>
