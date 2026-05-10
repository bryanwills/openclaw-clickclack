<script lang="ts">
  import { avatarImageSource } from "../../lib/chat/avatars";
  import { avatarHue, avatarInitial } from "../../lib/chat/people";

  type AvatarLoading = "eager" | "lazy";
  type AvatarFetchPriority = "high" | "low" | "auto";

  type Props = {
    id?: string | null;
    name?: string | null;
    src?: string | null;
    class?: string;
    size?: number;
    loading?: AvatarLoading;
    fetchPriority?: AvatarFetchPriority;
    buttonLabel?: string;
    onclick?: (event: MouseEvent) => void;
  };

  let {
    id,
    name,
    src,
    class: className = "avatar",
    size = 40,
    loading = "lazy",
    fetchPriority = "low",
    buttonLabel,
    onclick,
  }: Props = $props();

  let failedSource = $state("");

  const source = $derived(avatarImageSource(src));
  const showImage = $derived(source !== "" && failedSource !== source);
  const hue = $derived(avatarHue(id || name || source || "avatar"));
  const initial = $derived(avatarInitial(name));

  function onImageError() {
    failedSource = source;
  }
</script>

{#if buttonLabel}
  <button
    type="button"
    class={className}
    style="--hue: {hue}deg"
    aria-label={buttonLabel}
    {onclick}
  >
    {#if showImage}
      <img
        src={source}
        alt=""
        width={size}
        height={size}
        {loading}
        decoding="async"
        fetchpriority={fetchPriority}
        onerror={onImageError}
      />
    {:else}
      {initial}
    {/if}
  </button>
{:else}
  <span class={className} style="--hue: {hue}deg">
    {#if showImage}
      <img
        src={source}
        alt=""
        width={size}
        height={size}
        {loading}
        decoding="async"
        fetchpriority={fetchPriority}
        onerror={onImageError}
      />
    {:else}
      {initial}
    {/if}
  </span>
{/if}
