<script lang="ts" module>
  export const QUICK_EMOJIS = [
    "👍",
    "❤️",
    "😂",
    "😮",
    "😢",
    "🙏",
    "🎉",
    "🔥",
    "💯",
    "👀",
    "🚀",
    "✅",
  ];
</script>

<script lang="ts">
  import { tick } from "svelte";

  let {
    id,
    placement = "above",
    disabled = false,
    onPick,
    onEscape,
  }: {
    id: string;
    placement?: "above" | "above-right" | "below";
    disabled?: boolean;
    onPick: (emoji: string) => void;
    onEscape: () => void;
  } = $props();

  let gridRef = $state<HTMLDivElement>();

  function handleKeydown(event: KeyboardEvent) {
    if (event.key !== "Escape") return;
    event.preventDefault();
    onEscape();
  }

  $effect(() => {
    void (async () => {
      await tick();
      gridRef?.querySelector<HTMLButtonElement>(".emoji-option")?.focus();
    })();
  });
</script>

<div
  bind:this={gridRef}
  class="emoji-grid"
  class:below={placement === "below"}
  class:above-right={placement === "above-right"}
  {id}
  role="group"
  aria-label="Choose a reaction"
>
  {#each QUICK_EMOJIS as emoji}
    <button
      class="emoji-option"
      onclick={() => onPick(emoji)}
      aria-label={`React with ${emoji}`}
      title={emoji}
      {disabled}
      onkeydown={handleKeydown}
    >
      {emoji}
    </button>
  {/each}
</div>

<style>
  .emoji-grid {
    position: absolute;
    bottom: 100%;
    left: 0;
    margin-bottom: 4px;
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 2px;
    padding: 6px;
    background: var(--panel, #fff);
    border: 1px solid var(--line-strong, #e0e0e0);
    border-radius: var(--radius, 8px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
    z-index: 100;
    min-width: 200px;
  }

  .emoji-grid.below {
    bottom: auto;
    top: calc(100% + 4px);
    left: auto;
    right: 0;
    margin-bottom: 0;
  }

  .emoji-grid.above-right {
    left: auto;
    right: 0;
  }

  .emoji-option {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    padding: 0;
    border: none;
    border-radius: 4px;
    background: transparent;
    cursor: pointer;
    font-size: 18px;
    line-height: 1;
    transition: background 0.1s;
  }

  .emoji-option:hover {
    background: color-mix(in srgb, var(--accent, #5865f2) 12%, transparent);
  }

  .emoji-option:disabled {
    cursor: wait;
    opacity: 0.55;
  }
</style>
