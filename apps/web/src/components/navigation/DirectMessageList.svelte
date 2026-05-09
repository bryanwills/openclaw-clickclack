<script lang="ts">
  import { avatarHue, avatarInitial, dmAvatarUser, dmTitle } from "../../lib/chat/people";
  import type { DirectConversation } from "../../lib/types";

  type Props = {
    conversations: DirectConversation[];
    currentUserID?: string;
    selectedDirectID: string;
    onSelectDirect: (conversationID: string) => void;
    onCreateDirect: () => void;
  };

  let {
    conversations,
    currentUserID,
    selectedDirectID,
    onSelectDirect,
    onCreateDirect,
  }: Props = $props();
</script>

<section class="nav-section">
  <div class="section-title">
    <span class="caret" aria-hidden="true">▾</span>
    <span class="label">Direct messages</span>
    <button
      type="button"
      class="add-button"
      aria-label="Start direct message"
      title="Start direct message"
      onclick={onCreateDirect}
    >＋</button>
  </div>
  <div class="nav-list">
    {#each conversations as conversation (conversation.id)}
      {@const dmUser = dmAvatarUser(conversation, currentUserID)}
      {@const unread = conversation.unread_count || 0}
      {@const isActive = conversation.id === selectedDirectID}
      <button
        class="nav-item dm"
        class:active={isActive}
        class:has-unread={unread > 0 && !isActive}
        onclick={() => onSelectDirect(conversation.id)}
      >
        <span class="dm-avatar" style="--hue: {avatarHue(dmUser?.id || conversation.id)}deg">
          {#if dmUser?.avatar_url}
            <img src={dmUser.avatar_url} alt="" loading="lazy" />
          {:else}
            {avatarInitial(dmUser?.display_name)}
          {/if}
        </span>
        <span class="nav-label">{dmTitle(conversation, currentUserID)}</span>
        {#if unread > 0 && !isActive}
          <span class="unread-badge" aria-label={`${unread} unread`}>{unread > 99 ? "99+" : unread}</span>
        {:else}
          <span class="presence-dot" aria-hidden="true"></span>
        {/if}
      </button>
    {/each}
    {#if conversations.length === 0}
      <p class="nav-empty">No direct messages yet</p>
    {/if}
  </div>
</section>
