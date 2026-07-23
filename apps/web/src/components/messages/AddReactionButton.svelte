<script lang="ts">
  import EmojiPicker from "./EmojiPicker.svelte";
  import { shouldOpenUpward } from "../../lib/popover";

  let {
    messageId,
    disabled = false,
    pending = false,
    buttonClass = "",
    onToggle,
  }: {
    messageId: string;
    disabled?: boolean;
    pending?: boolean;
    buttonClass?: string;
    onToggle: (emoji: string) => void;
  } = $props();

  let showPicker = $state(false);
  let pickerUp = $state(false);
  let wrapRef = $state<HTMLDivElement>();
  let buttonRef = $state<HTMLButtonElement>();
  let pickerId = $derived(`add-reaction-picker-${messageId}`);

  function togglePicker() {
    if (disabled || pending) return;
    if (!showPicker) pickerUp = shouldOpenUpward(wrapRef, 130);
    showPicker = !showPicker;
  }

  function choose(emoji: string) {
    if (disabled || pending) return;
    onToggle(emoji);
    showPicker = false;
  }

  function closePicker() {
    showPicker = false;
    buttonRef?.focus();
  }

  function handleClickOutside(e: MouseEvent) {
    if (wrapRef && !wrapRef.contains(e.target as Node)) {
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

<div class="add-reaction-wrapper" bind:this={wrapRef}>
  <button
    bind:this={buttonRef}
    type="button"
    class={buttonClass}
    aria-label="Add reaction"
    data-tooltip="Add reaction"
    aria-controls={pickerId}
    aria-expanded={showPicker}
    disabled={disabled || pending}
    onclick={togglePicker}
  >
    <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
      <g fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
        <circle cx="12" cy="12" r="9"/>
        <path d="M8 14s1.5 2 4 2 4-2 4-2M9 9h.01M15 9h.01"/>
      </g>
    </svg>
  </button>
  {#if showPicker}
    <EmojiPicker
      id={pickerId}
      placement={pickerUp ? "above-right" : "below"}
      disabled={disabled || pending}
      onPick={choose}
      onEscape={closePicker}
    />
  {/if}
</div>

<style>
  .add-reaction-wrapper {
    position: relative;
    display: inline-flex;
  }
</style>
