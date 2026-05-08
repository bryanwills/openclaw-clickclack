<script lang="ts">
  import { avatarHue, avatarInitial, dmAvatarUser, dmTitle } from "../../lib/chat/people";
  import type { DirectConversation } from "../../lib/types";

  type Props = {
    conversations: DirectConversation[];
    currentUserID?: string;
    selectedDirectID: string;
    directMemberID: string;
    onSelectDirect: (conversationID: string) => void;
    onDirectMemberID: (value: string) => void;
    onCreateDirect: () => void;
  };

  let {
    conversations,
    currentUserID,
    selectedDirectID,
    directMemberID,
    onSelectDirect,
    onDirectMemberID,
    onCreateDirect,
  }: Props = $props();
</script>

<section class="nav-section">
  <div class="section-title">
    <span class="caret" aria-hidden="true">▾</span>
    <span class="label">Direct messages</span>
  </div>
  <div class="nav-list">
    {#each conversations as conversation (conversation.id)}
      {@const dmUser = dmAvatarUser(conversation, currentUserID)}
      <button
        class="nav-item dm"
        class:active={conversation.id === selectedDirectID}
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
        <span class="presence-dot" aria-hidden="true"></span>
      </button>
    {/each}
    {#if conversations.length === 0}
      <p class="nav-empty">No direct messages yet</p>
    {/if}
  </div>
  <form
    class="inline-create"
    onsubmit={(event) => {
      event.preventDefault();
      onCreateDirect();
    }}
  >
    <input
      value={directMemberID}
      placeholder="user id"
      aria-label="DM member user ID"
      oninput={(event) => onDirectMemberID(event.currentTarget.value)}
    />
    <button type="submit" class="ghost" aria-label="Start DM">＋</button>
  </form>
</section>
