<script lang="ts">
  import {
    BOARD_THEMES,
    COLOR_MODES,
    DENSITIES,
    MESSAGE_LAYOUTS,
    loadBoardTheme,
    loadColorMode,
    loadDensity,
    loadMessageLayout,
    setBoardTheme,
    setColorMode,
    setDensity,
    setMessageLayout,
    type BoardTheme,
    type ColorMode,
    type Density,
    type MessageLayout,
  } from "../../lib/appearance";
  import type { User } from "../../lib/types";

  let { user }: { user: User } = $props();

  const previewName = $derived(user.display_name || user.handle || "You");
  const previewInitial = $derived((previewName[0] ?? "Y").toUpperCase());

  let colorMode = $state<ColorMode>(loadColorMode());
  let boardTheme = $state<BoardTheme>(loadBoardTheme());
  let messageLayout = $state<MessageLayout>(loadMessageLayout());
  let density = $state<Density>(loadDensity());

  $effect(() => {
    void user.appearance_preferences;
    colorMode = loadColorMode();
    boardTheme = loadBoardTheme();
    messageLayout = loadMessageLayout();
    density = loadDensity();
  });

  function pickMode(mode: ColorMode) {
    colorMode = mode;
    setColorMode(mode);
  }

  function pickBoard(board: BoardTheme) {
    boardTheme = board;
    setBoardTheme(board);
  }

  function pickMessageLayout(layout: MessageLayout) {
    messageLayout = layout;
    setMessageLayout(layout);
  }

  function pickDensity(next: Density) {
    density = next;
    setDensity(next);
  }

  function moveRadioSelection(
    event: KeyboardEvent,
    optionCount: number,
    currentIndex: number,
    select: (index: number) => void,
  ) {
    let nextIndex: number;
    switch (event.key) {
      case "ArrowRight":
      case "ArrowDown":
        nextIndex = (currentIndex + 1) % optionCount;
        break;
      case "ArrowLeft":
      case "ArrowUp":
        nextIndex = (currentIndex - 1 + optionCount) % optionCount;
        break;
      case "Home":
        nextIndex = 0;
        break;
      case "End":
        nextIndex = optionCount - 1;
        break;
      default:
        return;
    }

    event.preventDefault();
    select(nextIndex);
    const radios = (event.currentTarget as HTMLElement).parentElement?.querySelectorAll<HTMLElement>(
      '[role="radio"]',
    );
    radios?.item(nextIndex).focus();
  }
</script>

<header class="settings-page__header">
  <p class="settings-page__eyebrow">Account</p>
  <h2 class="settings-page__h1">Appearance</h2>
  <p class="settings-page__lead">
    Changes apply instantly and follow your account on every device.
  </p>
</header>

<div class="settings-form">
  <!-- Live preview: picks apply globally the moment they're made, so the strip
       inherits color mode + board tokens for free; layout and density are
       mirrored locally via data attributes so the sample messages restyle. -->
  <div
    class="appearance-preview"
    data-layout={messageLayout}
    data-density={density}
    aria-hidden="true"
  >
    <div class="appearance-preview__bar">
      <span class="appearance-preview__hash">#</span>
      <span>deploys</span>
      <span class="appearance-preview__topic">Release pipeline chatter</span>
    </div>
    <div class="appearance-preview__chat">
      <div class="appearance-preview__msg">
        <span class="appearance-preview__avatar">{previewInitial}</span>
        <span class="appearance-preview__body">
          <span class="appearance-preview__head">
            <span class="appearance-preview__who">{previewName}</span>
            <span class="appearance-preview__when">2:14 PM</span>
          </span>
          <span class="appearance-preview__text">
            Pipeline's green again — can we ship v0.2.2 to staging?
          </span>
        </span>
      </div>
      <div class="appearance-preview__msg">
        <span class="appearance-preview__avatar is-agent">C</span>
        <span class="appearance-preview__body">
          <span class="appearance-preview__head">
            <span class="appearance-preview__who">Claw</span>
            <span class="bot-chip">bot</span>
            <span class="appearance-preview__when">2:15 PM</span>
          </span>
          <span class="appearance-preview__activity">
            Ran <code>make test && make deploy staging</code> · 34 passed · 12s
          </span>
          <span class="appearance-preview__answer">
            Done — staging is live on v0.2.2. Smoke checks passed; dashboards look clean.
          </span>
          <span class="appearance-preview__reactions">
            <span class="appearance-preview__reaction is-mine">🚀 2</span>
            <span class="appearance-preview__reaction">✅ 1</span>
          </span>
        </span>
      </div>
      <div class="appearance-preview__msg">
        <span class="appearance-preview__avatar">{previewInitial}</span>
        <span class="appearance-preview__body">
          <span class="appearance-preview__head">
            <span class="appearance-preview__who">{previewName}</span>
            <span class="appearance-preview__when">2:16 PM</span>
          </span>
          <span class="appearance-preview__text">Perfect. Promoting to prod after lunch 🌮</span>
        </span>
      </div>
    </div>
  </div>
  <p class="appearance-preview__caption">Live preview — updates as you pick</p>

  <div class="appearance-rows">
    <div class="appearance-row">
      <div class="appearance-row__meta">
        <span class="appearance-row__label">Color mode</span>
        <span class="appearance-row__desc">System follows your OS setting</span>
      </div>
      <div class="appearance-seg" role="radiogroup" aria-label="Color mode">
        {#each COLOR_MODES as mode, index (mode.id)}
          <button
            type="button"
            class="appearance-seg__btn"
            class:is-active={colorMode === mode.id}
            role="radio"
            aria-checked={colorMode === mode.id}
            tabindex={colorMode === mode.id ? 0 : -1}
            onclick={() => pickMode(mode.id)}
            onkeydown={(event) =>
              moveRadioSelection(event, COLOR_MODES.length, index, (nextIndex) =>
                pickMode(COLOR_MODES[nextIndex].id),
              )}
          >
            {mode.label}
          </button>
        {/each}
      </div>
    </div>

    <div class="appearance-row">
      <div class="appearance-row__meta">
        <span class="appearance-row__label">Board theme</span>
        <span class="appearance-row__desc">Palette for boards, accents, and highlights</span>
      </div>
      <div class="appearance-row__control">
        <span class="board-chips__name">
          {BOARD_THEMES.find((board) => board.id === boardTheme)?.label}
        </span>
        <div class="board-chips" role="radiogroup" aria-label="Board theme">
          {#each BOARD_THEMES as board, index (board.id)}
            <button
              type="button"
              class="board-chip"
              class:is-active={boardTheme === board.id}
              role="radio"
              aria-checked={boardTheme === board.id}
              tabindex={boardTheme === board.id ? 0 : -1}
              data-board={board.id}
              aria-label={`${board.label} — ${board.blurb}`}
              title={`${board.label} — ${board.blurb}`}
              onclick={() => pickBoard(board.id)}
              onkeydown={(event) =>
                moveRadioSelection(event, BOARD_THEMES.length, index, (nextIndex) =>
                  pickBoard(BOARD_THEMES[nextIndex].id),
                )}
            ></button>
          {/each}
        </div>
      </div>
    </div>

    <div class="appearance-row">
      <div class="appearance-row__meta">
        <span class="appearance-row__label">Message layout</span>
        <span class="appearance-row__desc">How agent activity attaches to replies</span>
      </div>
      <div class="appearance-seg" role="radiogroup" aria-label="Message layout">
        {#each MESSAGE_LAYOUTS as layout, index (layout.id)}
          <button
            type="button"
            class="appearance-seg__btn"
            class:is-active={messageLayout === layout.id}
            role="radio"
            aria-checked={messageLayout === layout.id}
            tabindex={messageLayout === layout.id ? 0 : -1}
            title={layout.blurb}
            onclick={() => pickMessageLayout(layout.id)}
            onkeydown={(event) =>
              moveRadioSelection(event, MESSAGE_LAYOUTS.length, index, (nextIndex) =>
                pickMessageLayout(MESSAGE_LAYOUTS[nextIndex].id),
              )}
          >
            {layout.label}
          </button>
        {/each}
      </div>
    </div>

    <div class="appearance-row">
      <div class="appearance-row__meta">
        <span class="appearance-row__label">Density</span>
        <span class="appearance-row__desc">Compact fits more messages on screen</span>
      </div>
      <div class="appearance-seg" role="radiogroup" aria-label="Density">
        {#each DENSITIES as option, index (option.id)}
          <button
            type="button"
            class="appearance-seg__btn"
            class:is-active={density === option.id}
            role="radio"
            aria-checked={density === option.id}
            tabindex={density === option.id ? 0 : -1}
            title={option.blurb}
            onclick={() => pickDensity(option.id)}
            onkeydown={(event) =>
              moveRadioSelection(event, DENSITIES.length, index, (nextIndex) =>
                pickDensity(DENSITIES[nextIndex].id),
              )}
          >
            {option.label}
          </button>
        {/each}
      </div>
    </div>
  </div>
</div>
