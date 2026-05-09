<script lang="ts">
  type Props = {
    channelName: string;
    status: string;
    onChannelName: (value: string) => void;
    onClose: () => void;
    onCreate: () => void;
  };

  let { channelName, status, onChannelName, onClose, onCreate }: Props = $props();
</script>

<div class="modal-scrim" role="presentation">
  <button class="modal-backdrop" type="button" aria-label="Close channel dialog" onclick={onClose}></button>
  <section class="profile-modal create-modal" aria-label="Create channel">
    <header>
      <div>
        <p>Channels</p>
        <h2>Create channel</h2>
      </div>
      <button type="button" aria-label="Close channel dialog" onclick={onClose}>×</button>
    </header>
    <form
      class="profile-form"
      onsubmit={(event) => {
        event.preventDefault();
        onCreate();
      }}
    >
      <label class="field">
        <span>Channel name</span>
        <input
          value={channelName}
          aria-label="Channel name"
          placeholder="product-launch"
          autocomplete="off"
          oninput={(event) => onChannelName(event.currentTarget.value)}
        />
      </label>
      {#if status}<p class="profile-status">{status}</p>{/if}
      <div class="profile-actions">
        <button type="button" class="ghost-action" onclick={onClose}>Cancel</button>
        <button type="submit" class="primary-action">Create channel</button>
      </div>
    </form>
  </section>
</div>
