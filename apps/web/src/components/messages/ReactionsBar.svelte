<script lang="ts">
  import EmojiPicker from "./EmojiPicker.svelte";
  import type { ReactionSummary } from "../../lib/types";

  let {
    messageId,
    reactions = [],
    pending = false,
    error = "",
    disabled = false,
    onToggle,
  }: {
    messageId: string;
    reactions: ReactionSummary[];
    pending?: boolean;
    error?: string;
    disabled?: boolean;
    onToggle: (emoji: string) => void;
  } = $props();

  let groupedEntries = $derived(
    [...reactions].sort(
      (a, b) => b.count - a.count || a.emoji.localeCompare(b.emoji),
    ),
  );

  let showPicker = $state(false);
  let pickerWrapRef = $state<HTMLDivElement>();
  let addButtonRef = $state<HTMLButtonElement>();
  let pickerId = $derived(`reaction-picker-${messageId}`);

  function togglePicker() {
    if (disabled || pending) return;
    showPicker = !showPicker;
  }

  function chooseReaction(emoji: string) {
    if (disabled || pending) return;
    onToggle(emoji);
    showPicker = false;
  }

  function closePicker() {
    showPicker = false;
    addButtonRef?.focus();
  }

  function handleClickOutside(e: MouseEvent) {
    if (pickerWrapRef && !pickerWrapRef.contains(e.target as Node)) {
      showPicker = false;
    }
  }

  $effect(() => {
    if (showPicker) {
      document.addEventListener("click", handleClickOutside);
      return () => document.removeEventListener("click", handleClickOutside);
    }
  });

  $effect(() => {
    if (disabled || pending) showPicker = false;
  });
</script>

{#if groupedEntries.length > 0 || error}
  <div class="reactions-bar">
    {#each groupedEntries as { emoji, count, reacted_by_me }}
      <button
        class="reaction-btn"
        class:me={reacted_by_me}
        onclick={() => onToggle(emoji)}
        disabled={disabled || pending}
        aria-pressed={reacted_by_me}
        aria-label="{emoji} — {count} reaction{count !== 1 ? 's' : ''}"
        title="{emoji}"
      >
        <span class="reaction-emoji">{emoji}</span>
        {#if count > 1}
          <span class="reaction-count">{count}</span>
        {/if}
      </button>
    {/each}

    {#if groupedEntries.length > 0 && !disabled}
      <div class="picker-wrapper" bind:this={pickerWrapRef}>
        <button
          bind:this={addButtonRef}
          class="reaction-btn add-btn"
          onclick={togglePicker}
          aria-label="Add another reaction"
          aria-controls={pickerId}
          aria-expanded={showPicker}
          disabled={disabled || pending}
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <circle cx="12" cy="12" r="9"/>
            <path d="M8 14s1.5 2 4 2 4-2 4-2M9 9h.01M15 9h.01"/>
          </svg>
          <svg width="10" height="10" viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <path d="M8 3v10M3 8h10" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        </button>

        {#if showPicker}
          <EmojiPicker
            id={pickerId}
            disabled={disabled || pending}
            onPick={chooseReaction}
            onEscape={closePicker}
          />
        {/if}
      </div>
    {/if}
    {#if error}<span class="reaction-error" role="status">{error}</span>{/if}
  </div>
{/if}

<style>
  .reactions-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    align-items: center;
    margin-top: 5px;
  }

  .reaction-btn {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 8px;
    border: 1px solid var(--line-strong, rgba(16, 21, 29, 0.17));
    border-radius: 999px;
    background: var(--panel, #fff);
    cursor: pointer;
    font-size: 12.5px;
    line-height: 1.4;
    transition: background 0.1s, border-color 0.1s;
    color: var(--muted, #666);
  }

  .reaction-btn:hover {
    background: color-mix(in srgb, var(--accent, #5865f2) 10%, transparent);
    border-color: color-mix(in srgb, var(--accent, #5865f2) 30%, transparent);
  }

  .reaction-btn:disabled {
    cursor: wait;
    opacity: 0.55;
  }

  .reaction-btn.me {
    background: var(--accent-soft, rgba(0, 128, 196, 0.13));
    border-color: var(--accent, #0080c4);
    color: var(--text-strong, #10151d);
  }

  .reaction-emoji {
    font-size: 14px;
    line-height: 1;
  }

  .reaction-count {
    font-size: 11.5px;
    font-weight: 700;
    color: var(--muted, #666);
  }

  .reaction-btn.me .reaction-count {
    color: var(--accent, #5865f2);
  }

  /* The add-chip only appears next to existing chips (Slack model): dashed,
     quiet, and never rendered under a message without reactions. */
  .add-btn {
    padding: 2px 7px;
    border-style: dashed;
    background: transparent;
    color: var(--muted-2, #8b94a3);
  }

  .add-btn:hover {
    color: var(--muted, #666);
  }

  .picker-wrapper {
    position: relative;
  }

  .reaction-error {
    color: var(--danger, #b42318);
    font-size: 12px;
  }
</style>
