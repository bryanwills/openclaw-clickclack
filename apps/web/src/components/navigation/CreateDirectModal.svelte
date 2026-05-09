<script lang="ts">
  import { avatarHue, avatarInitial, handleLabel } from "../../lib/chat/people";
  import type { User } from "../../lib/types";

  type Props = {
    people: User[];
    currentUserID?: string;
    memberID: string;
    onMemberID: (value: string) => void;
    onClose: () => void;
    onStart: (memberID: string) => void;
  };

  let { people, currentUserID, memberID, onMemberID, onClose, onStart }: Props = $props();

  let query = $derived(memberID.trim().toLowerCase());
  let choices = $derived(people
    .filter((person) => person.id !== currentUserID)
    .filter((person) => {
      if (!query) return true;
      return (
        person.display_name.toLowerCase().includes(query) ||
        person.handle?.toLowerCase().includes(query) ||
        person.id.toLowerCase().includes(query)
      );
    }));
</script>

<div class="modal-scrim" role="presentation">
  <button class="modal-backdrop" type="button" aria-label="Close direct message dialog" onclick={onClose}></button>
  <section class="profile-modal create-modal" aria-label="Start direct message">
    <header>
      <div>
        <p>Direct messages</p>
        <h2>Start a DM</h2>
      </div>
      <button type="button" aria-label="Close direct message dialog" onclick={onClose}>×</button>
    </header>
    <form
      class="profile-form"
      onsubmit={(event) => {
        event.preventDefault();
        onStart(memberID);
      }}
    >
      <label class="field">
        <span>Find a person</span>
        <input
          value={memberID}
          aria-label="Find a person"
          placeholder="Name, handle, or user id"
          autocomplete="off"
          oninput={(event) => onMemberID(event.currentTarget.value)}
        />
      </label>

      <div class="person-picker" aria-label="People">
        {#each choices as person (person.id)}
          <button type="button" class="person-choice" onclick={() => onStart(person.id)}>
            <span class="dm-avatar" style="--hue: {avatarHue(person.id)}deg">
              {#if person.avatar_url}
                <img src={person.avatar_url} alt="" loading="lazy" />
              {:else}
                {avatarInitial(person.display_name)}
              {/if}
            </span>
            <span>
              <strong>{person.display_name}</strong>
              <small>{handleLabel(person.handle) || person.id}</small>
            </span>
          </button>
        {/each}
        {#if choices.length === 0}
          <div class="person-empty">No matching people yet</div>
        {/if}
      </div>

      <div class="profile-actions">
        <button type="button" class="ghost-action" onclick={onClose}>Cancel</button>
        <button type="submit" class="primary-action">Start DM</button>
      </div>
    </form>
  </section>
</div>
