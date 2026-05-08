<script lang="ts">
  import type { Upload } from "../lib/types";

  type Props = {
    upload: Upload;
    url: string;
    onOpenImage?: (url: string, title: string) => void;
  };

  let { upload, url, onOpenImage = () => {} }: Props = $props();

  let videoEl: HTMLVideoElement | null = $state(null);
  let started = $state(false);
  let durationLabel = $state("");

  let isImage = $derived(upload.content_type?.startsWith("image/") ?? false);
  let isVideo = $derived(upload.content_type?.startsWith("video/") ?? false);

  function handlePlay() {
    started = true;
  }

  function handleLoadedMetadata() {
    if (!videoEl || !isFinite(videoEl.duration)) return;
    const total = Math.floor(videoEl.duration);
    const m = Math.floor(total / 60);
    const s = total % 60;
    durationLabel = `${m}:${s.toString().padStart(2, "0")}`;
  }

  function startPlayback() {
    if (!videoEl) return;
    started = true;
    void videoEl.play();
  }

  function formatBytes(size: number) {
    if (size < 1024) return `${size} B`;
    if (size < 1024 * 1024) return `${Math.round(size / 1024)} KB`;
    return `${(size / (1024 * 1024)).toFixed(1)} MB`;
  }
</script>

{#if isImage}
  <div class="media-tile media-tile--image">
    <button
      type="button"
      class="media-tile__open"
      aria-label={`Open image ${upload.filename}`}
      onclick={() => onOpenImage(url, upload.filename)}
    >
      <img src={url} alt={upload.filename} loading="lazy" />
    </button>
    <div class="media-tile__caption">
      <span class="media-tile__name">{upload.filename}</span>
      <a
        class="media-tile__chip"
        href={url}
        download={upload.filename}
        aria-label={`Download ${upload.filename}`}
        onclick={(event) => event.stopPropagation()}
      >
        <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
          <path
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M12 4v12m0 0 4-4m-4 4-4-4M5 20h14"
          />
        </svg>
      </a>
    </div>
  </div>
{:else if isVideo}
  <div class="media-tile media-tile--video" class:is-started={started}>
    <video
      bind:this={videoEl}
      preload="metadata"
      playsinline
      controls={started}
      controlslist="nodownload"
      aria-label={upload.filename}
      onplay={handlePlay}
      onloadedmetadata={handleLoadedMetadata}
    >
      <source src={url} type={upload.content_type} />
    </video>
    {#if !started}
      <button
        type="button"
        class="media-tile__play"
        aria-label={`Play ${upload.filename}`}
        onclick={startPlayback}
      >
        <span class="media-tile__play-icon" aria-hidden="true">
          <svg viewBox="0 0 24 24" width="26" height="26">
            <path fill="currentColor" d="M8 5.5v13l11-6.5z" />
          </svg>
        </span>
      </button>
      {#if durationLabel}
        <span class="media-tile__duration" aria-hidden="true">{durationLabel}</span>
      {/if}
    {/if}
    <div class="media-tile__caption">
      <span class="media-tile__name">{upload.filename}</span>
      <a
        class="media-tile__chip"
        href={url}
        download={upload.filename}
        aria-label={`Download ${upload.filename}`}
        onclick={(event) => event.stopPropagation()}
      >
        <svg viewBox="0 0 24 24" width="14" height="14" aria-hidden="true">
          <path
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M12 4v12m0 0 4-4m-4 4-4-4M5 20h14"
          />
        </svg>
      </a>
    </div>
  </div>
{:else}
  <a class="file-attachment" href={url} target="_blank" rel="noreferrer">
    <span class="file-icon" aria-hidden="true">↧</span>
    <span>
      <strong>{upload.filename}</strong>
      <small>{formatBytes(upload.byte_size)}</small>
    </span>
  </a>
{/if}
