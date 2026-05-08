<script lang="ts">
  import { avatarHue, avatarInitial } from "../../lib/chat/people";
  import { time } from "../../lib/format";
  import type { SearchResult } from "../../lib/types";

  type Props = {
    results: SearchResult[];
    onClose: () => void;
    onOpenResult: (result: SearchResult) => void;
  };

  let { results, onClose, onOpenResult }: Props = $props();
</script>

{#if results.length > 0}
  <div class="search-results" aria-label="Search results">
    <div class="search-results-head">
      <strong>{results.length} {results.length === 1 ? "result" : "results"}</strong>
      <button type="button" onclick={onClose}>Close</button>
    </div>
    {#each results as result (result.message.id)}
      <button class="search-result" onclick={() => onOpenResult(result)}>
        <span class="dm-avatar" style="--hue: {avatarHue(result.message.author?.id || result.message.author_id || 'x')}deg">
          {#if result.message.author?.avatar_url}
            <img src={result.message.author.avatar_url} alt="" loading="lazy" />
          {:else}
            {avatarInitial(result.message.author?.display_name)}
          {/if}
        </span>
        <div class="search-result-body">
          <div>
            <strong>{result.message.author?.display_name || "Local User"}</strong>
            <time>{time(result.message.created_at)}</time>
          </div>
          <span>{result.message.body}</span>
        </div>
      </button>
    {/each}
  </div>
{/if}
