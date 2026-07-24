import { expect, test, type Locator, type Page } from "@playwright/test";
import { randomUUID } from "node:crypto";
import { waitForAppReady } from "./app-ready";

async function openReactionChannel(page: Page) {
  const suffix = randomUUID().replaceAll("-", "").slice(0, 12);
  const workspaceResponse = await page.request.post("/api/workspaces", {
    data: { name: `Reaction Proof ${suffix}` },
  });
  expect(workspaceResponse.ok()).toBe(true);
  const { workspace } = (await workspaceResponse.json()) as {
    workspace: { id: string; route_id: string };
  };
  const channelResponse = await page.request.post(`/api/workspaces/${workspace.id}/channels`, {
    data: { name: `reaction-proof-${suffix}`, kind: "public" },
  });
  expect(channelResponse.ok()).toBe(true);
  const { channel } = (await channelResponse.json()) as {
    channel: { id: string; route_id: string; name: string };
  };

  await page.goto(`/app/${workspace.route_id}/${channel.route_id}`);
  await waitForAppReady(page);
  await expect(page.getByRole("heading", { name: `#${channel.name}` })).toBeVisible();
  return { suffix, workspace, channel };
}

async function sendMessage(page: Page, body: string): Promise<Locator> {
  await page.getByLabel("Message body").fill(body);
  await page.getByRole("button", { name: "Send" }).click();
  const row = page.locator(".message-row:not(.is-pending)", { hasText: body });
  await expect(row).toBeVisible();
  return row;
}

async function pickReaction(scope: Locator, emoji: string) {
  await scope.getByRole("button", { name: "Add reaction" }).click();
  await scope
    .getByRole("group", { name: "Choose a reaction" })
    .getByRole("button", { name: `React with ${emoji}` })
    .click();
}

test("reaction mutations are accessible, authoritative, persistent, and realtime", async ({
  page,
}) => {
  const { suffix } = await openReactionChannel(page);
  const row = await sendMessage(page, `Reaction behavior proof ${suffix}`);
  const messageID = await row.getAttribute("data-message-id");
  expect(messageID).toBeTruthy();

  const addButton = row.getByRole("button", { name: "Add reaction" });
  await addButton.click();
  const picker = row.getByRole("group", { name: "Choose a reaction" });
  await expect(picker).toBeVisible();
  await expect(picker.getByRole("button", { name: "React with 👍" })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(picker).toHaveCount(0);
  await expect(addButton).toBeFocused();

  const addResponsePromise = page.waitForResponse(
    (response) =>
      response.request().method() === "POST" &&
      response.url().endsWith(`/api/messages/${messageID}/reactions`),
  );
  await pickReaction(row, "👍");
  const addResponse = await addResponsePromise;
  expect(addResponse.ok()).toBe(true);
  const payload = (await addResponse.json()) as {
    event: { type: string; payload: { emoji?: string; count?: number } };
    reactions: Array<{ emoji: string; count: number; reacted_by_me: boolean }>;
  };
  expect(payload.event.type).toBe("reaction.added");
  expect(payload.event.payload).toMatchObject({ emoji: "👍", count: 1 });
  expect(payload.reactions).toEqual([{ emoji: "👍", count: 1, reacted_by_me: true }]);

  const reaction = row.getByRole("button", { name: "👍 — 1 reaction" });
  await expect(reaction).toBeVisible();
  await expect(reaction).toHaveAttribute("aria-pressed", "true");
  if (process.env.REACTION_PROOF_PATH) {
    await row.scrollIntoViewIfNeeded();
    await page.screenshot({ path: process.env.REACTION_PROOF_PATH, fullPage: true });
  }

  await page.reload();
  await waitForAppReady(page);
  const persistedRow = page.locator(`[data-message-id="${messageID}"]`);
  await expect(persistedRow.getByRole("button", { name: "👍 — 1 reaction" })).toBeVisible();

  let messageRefreshes = 0;
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (request.method() === "GET" && url.pathname === `/api/messages/${messageID}`) {
      messageRefreshes += 1;
    }
  });
  const removeResponse = await page.request.delete(
    `/api/messages/${messageID}/reactions/${encodeURIComponent("👍")}`,
  );
  expect(removeResponse.ok()).toBe(true);
  await expect(persistedRow.getByRole("button", { name: "👍 — 1 reaction" })).toHaveCount(0);
  expect(messageRefreshes).toBe(0);
});

test("desktop message actions stay clear of message content", async ({ page }) => {
  const { suffix } = await openReactionChannel(page);
  const previousRow = await sendMessage(page, `Desktop action neighbor ${suffix}`);
  const row = await sendMessage(
    page,
    `Desktop action placement ${suffix} keeps the first line readable.`,
  );
  await row.hover();

  const geometry = await row.evaluate(
    (element, previousMessageID) => {
      const actions = element.querySelector<HTMLElement>(".message-actions");
      const content = element.querySelector<HTMLElement>(".markdown");
      const previous = document.querySelector<HTMLElement>(
        `[data-message-id="${CSS.escape(previousMessageID)}"]`,
      );
      if (!actions || !content) throw new Error("message actions missing");
      if (!previous) throw new Error("previous message missing");

      const rowRect = element.getBoundingClientRect();
      const actionsRect = actions.getBoundingClientRect();
      const textRects = [content, previous].flatMap((container) => {
        const rects: DOMRect[] = [];
        const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT);
        while (walker.nextNode()) {
          const node = walker.currentNode;
          if (!node.textContent?.trim()) continue;
          const range = document.createRange();
          range.selectNodeContents(node);
          rects.push(...range.getClientRects());
        }
        return rects;
      });

      return {
        actionsTop: actionsRect.top,
        actionsBottom: actionsRect.bottom,
        rowTop: rowRect.top,
        rowBottom: rowRect.bottom,
        overlapsText: textRects.some(
          (rect) =>
            actionsRect.left < rect.right &&
            actionsRect.right > rect.left &&
            actionsRect.top < rect.bottom &&
            actionsRect.bottom > rect.top,
        ),
      };
    },
    (await previousRow.getAttribute("data-message-id"))!,
  );

  expect(geometry.actionsTop).toBeGreaterThanOrEqual(geometry.rowTop - 0.5);
  expect(geometry.actionsBottom).toBeLessThanOrEqual(geometry.rowBottom + 0.5);
  expect(geometry.overlapsText).toBe(false);
});

test("right-edge message action tooltips stay inside the message viewport", async ({ page }) => {
  const { suffix } = await openReactionChannel(page);
  const row = await sendMessage(page, `Desktop tooltip placement ${suffix}`);
  await row.hover();
  const trigger = row.getByRole("button", { name: "More actions" });
  await trigger.hover();

  const geometry = await trigger.evaluate((element) => {
    const scroller = element.closest<HTMLElement>(".messages-scroll");
    if (!scroller) throw new Error("message scroller missing");

    const triggerRect = element.getBoundingClientRect();
    const scrollerRect = scroller.getBoundingClientRect();
    const tooltipStyle = getComputedStyle(element, "::before");
    const tooltipWidth =
      Number.parseFloat(tooltipStyle.width) +
      Number.parseFloat(tooltipStyle.paddingLeft) +
      Number.parseFloat(tooltipStyle.paddingRight) +
      Number.parseFloat(tooltipStyle.borderLeftWidth) +
      Number.parseFloat(tooltipStyle.borderRightWidth);
    const tooltipRight = triggerRect.right - Number.parseFloat(tooltipStyle.right);
    return {
      tooltipLeft: tooltipRight - tooltipWidth,
      tooltipRight,
      scrollerLeft: scrollerRect.left,
      scrollerRight: scrollerRect.right,
    };
  });

  expect(geometry.tooltipLeft).toBeGreaterThanOrEqual(geometry.scrollerLeft - 0.5);
  expect(geometry.tooltipRight).toBeLessThanOrEqual(geometry.scrollerRight + 0.5);
});

test("touch long-press opens a message action sheet", async ({ browser, page }) => {
  const { suffix, workspace, channel } = await openReactionChannel(page);
  const body = `Touch action layout ${suffix}`;
  await sendMessage(page, body);

  const mobileContext = await browser.newContext({
    baseURL: new URL(page.url()).origin,
    hasTouch: true,
    isMobile: true,
    viewport: { width: 390, height: 844 },
  });
  const mobilePage = await mobileContext.newPage();
  await mobilePage.goto(`/app/${workspace.route_id}/${channel.route_id}`);
  await waitForAppReady(mobilePage);
  const row = mobilePage.locator(".message-row:not(.is-pending)", { hasText: body });
  await expect(row).toBeVisible();

  const trigger = row.getByRole("button", { name: "More actions" });
  expect(await trigger.getAttribute("data-tooltip")).toBeNull();
  expect(await trigger.getAttribute("class")).not.toContain("tooltip");

  // Touch hides persistent controls without removing the accessible sheet trigger.
  for (const width of [390, 320]) {
    await mobilePage.setViewportSize({ width, height: 844 });
    await row.scrollIntoViewIfNeeded();
    const geometry = await row.evaluate((element) => {
      const persistentActions = [
        ...element.querySelectorAll<HTMLElement>(".message-actions > :not(.message-more)"),
      ];
      const trigger = element.querySelector<HTMLElement>(".message-actions-trigger");
      return {
        persistentActionsHidden: persistentActions.every(
          (action) => getComputedStyle(action).display === "none",
        ),
        triggerVisuallyHidden:
          Boolean(trigger) &&
          trigger!.getBoundingClientRect().width <= 1 &&
          trigger!.getBoundingClientRect().height <= 1,
        scrollWidth: document.documentElement.scrollWidth,
        viewportWidth: window.innerWidth,
      };
    });
    expect(geometry.persistentActionsHidden).toBe(true);
    expect(geometry.triggerVisuallyHidden).toBe(true);
    expect(geometry.scrollWidth).toBeLessThanOrEqual(geometry.viewportWidth);
  }

  // Keyboard and assistive input can open the same modal without a gesture.
  const sheet = mobilePage.getByRole("dialog", { name: "Message actions" });
  await trigger.focus();
  expect(
    await row
      .locator(".message-content")
      .evaluate((element) => getComputedStyle(element).paddingRight),
  ).toBe("44px");
  await mobilePage.keyboard.press("Enter");
  await expect(sheet).toBeVisible();
  await expect(trigger).toHaveAttribute("aria-expanded", "true");
  await expect(sheet.getByRole("button", { name: "React with 👍" })).toBeFocused();
  await mobilePage.keyboard.press("Shift+Tab");
  await expect(sheet.getByRole("button", { name: "Delete message" })).toBeFocused();
  await mobilePage.keyboard.press("Tab");
  await expect(sheet.getByRole("button", { name: "React with 👍" })).toBeFocused();
  await mobilePage.keyboard.press("Escape");
  await expect(sheet).toBeHidden();
  await expect(trigger).toBeFocused();
  await expect(trigger).toHaveAttribute("aria-expanded", "false");

  // A keyboard reaction restores the focusable entry point after the modal closes.
  await mobilePage.keyboard.press("Enter");
  await sheet.getByRole("button", { name: "React with ✅" }).click();
  await expect(sheet).toBeHidden();
  await expect(trigger).toBeFocused();
  await expect(row.getByRole("button", { name: "✅ — 1 reaction" })).toBeVisible();

  // Long-press (click held past the 450ms threshold) opens the bottom sheet.
  const content = row.locator(".message-content");
  const selectionStyle = await content.evaluate((element) => ({
    userSelect: getComputedStyle(element).userSelect,
    touchCallout: getComputedStyle(element).getPropertyValue("-webkit-touch-callout"),
  }));
  expect(selectionStyle.userSelect).not.toBe("none");
  expect(selectionStyle.touchCallout).not.toBe("none");
  await content.click({ delay: 600 });
  await expect(sheet).toBeVisible();
  await expect(sheet.getByRole("button", { name: "Open thread" })).toBeVisible();
  await expect(sheet.getByRole("button", { name: "Reply" })).toBeVisible();
  await expect(sheet.getByRole("button", { name: "Copy text" })).toBeVisible();

  // Escape closes; backdrop closes.
  await mobilePage.keyboard.press("Escape");
  await expect(sheet).toBeHidden();
  await content.click({ delay: 600 });
  await expect(sheet).toBeVisible();
  await mobilePage.getByRole("button", { name: "Close message actions" }).click();
  await expect(sheet).toBeHidden();

  // Reacting from the sheet lands a real reaction chip.
  await content.click({ delay: 600 });
  await sheet.getByRole("button", { name: "React with 👍" }).click();
  await expect(sheet).toBeHidden();
  await expect(row.getByRole("button", { name: "👍 — 1 reaction" })).toBeVisible();

  // A quick tap (no hold) still opens the thread instead of the sheet.
  await content.click();
  await expect(mobilePage.locator(".thread-root .markdown")).toContainText(body);
  await expect(sheet).toBeHidden();

  await mobileContext.close();
});

test("touch holds work on hybrid devices without hijacking mouse input or inline images", async ({
  browser,
  page,
}) => {
  const { suffix, workspace, channel } = await openReactionChannel(page);
  const body = `Hybrid touch action ${suffix} ![Inline proof](/favicon.svg)`;
  await sendMessage(page, body);

  const hybridContext = await browser.newContext({
    baseURL: new URL(page.url()).origin,
    hasTouch: true,
    viewport: { width: 900, height: 700 },
  });
  const hybridPage = await hybridContext.newPage();
  await hybridPage.goto(`/app/${workspace.route_id}/${channel.route_id}`);
  await waitForAppReady(hybridPage);

  const row = hybridPage.locator(".message-row:not(.is-pending)", {
    hasText: `Hybrid touch action ${suffix}`,
  });
  const content = row.locator(".message-content");
  const sheet = hybridPage.getByRole("dialog", { name: "Message actions" });

  await content.dispatchEvent("pointerdown", {
    pointerId: 1,
    pointerType: "mouse",
    isPrimary: true,
    button: 0,
    clientX: 100,
    clientY: 100,
  });
  await hybridPage.waitForTimeout(500);
  await content.dispatchEvent("pointerup", {
    pointerId: 1,
    pointerType: "mouse",
    isPrimary: true,
    button: 0,
    clientX: 100,
    clientY: 100,
  });
  await expect(sheet).toBeHidden();

  await content.dispatchEvent("pointerdown", {
    pointerId: 2,
    pointerType: "touch",
    isPrimary: true,
    button: 0,
    clientX: 100,
    clientY: 100,
  });
  await hybridPage.waitForTimeout(500);
  await content.dispatchEvent("pointerup", {
    pointerId: 2,
    pointerType: "touch",
    isPrimary: true,
    button: 0,
    clientX: 100,
    clientY: 100,
  });
  await expect(sheet).toBeVisible();
  await hybridPage.keyboard.press("Escape");

  const inlineImage = row.getByRole("img", { name: "Inline proof" });
  await inlineImage.dispatchEvent("pointerdown", {
    pointerId: 3,
    pointerType: "touch",
    isPrimary: true,
    button: 0,
    clientX: 100,
    clientY: 100,
  });
  await hybridPage.waitForTimeout(500);
  await inlineImage.dispatchEvent("pointerup", {
    pointerId: 3,
    pointerType: "touch",
    isPrimary: true,
    button: 0,
    clientX: 100,
    clientY: 100,
  });
  await expect(sheet).toBeHidden();
  await expect(
    hybridPage.getByRole("dialog", { name: "Image viewer: Inline proof" }),
  ).toBeVisible();

  await hybridContext.close();
});

test("a stale copy timer cannot close a reopened touch action sheet", async ({ browser, page }) => {
  const { suffix, workspace, channel } = await openReactionChannel(page);
  const body = `Touch copy lifecycle ${suffix}`;
  await sendMessage(page, body);

  const mobileContext = await browser.newContext({
    baseURL: new URL(page.url()).origin,
    hasTouch: true,
    isMobile: true,
    viewport: { width: 390, height: 844 },
  });
  await mobileContext.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: async () => {} },
    });
  });
  const mobilePage = await mobileContext.newPage();
  await mobilePage.goto(`/app/${workspace.route_id}/${channel.route_id}`);
  await waitForAppReady(mobilePage);

  const row = mobilePage.locator(".message-row:not(.is-pending)", { hasText: body });
  const trigger = row.getByRole("button", { name: "More actions" });
  const sheet = mobilePage.getByRole("dialog", { name: "Message actions" });
  await trigger.focus();
  await mobilePage.keyboard.press("Enter");
  await sheet.getByRole("button", { name: "Copy text" }).click();
  await expect(sheet.getByText("Copied", { exact: true })).toBeVisible();

  await mobilePage.getByRole("button", { name: "Close message actions" }).click();
  await expect(sheet).toBeHidden();
  await trigger.focus();
  await mobilePage.keyboard.press("Enter");
  await expect(sheet).toBeVisible();
  await mobilePage.waitForTimeout(1_000);
  await expect(sheet).toBeVisible();

  await mobileContext.close();
});

test("touch action sheets remain usable in short landscape viewports", async ({
  browser,
  page,
}) => {
  const { suffix, workspace, channel } = await openReactionChannel(page);
  const body = `Touch landscape layout ${suffix}`;
  await sendMessage(page, body);

  const mobileContext = await browser.newContext({
    baseURL: new URL(page.url()).origin,
    hasTouch: true,
    isMobile: true,
    viewport: { width: 667, height: 240 },
  });
  const mobilePage = await mobileContext.newPage();
  await mobilePage.goto(`/app/${workspace.route_id}/${channel.route_id}`);
  await waitForAppReady(mobilePage);

  const row = mobilePage.locator(".message-row:not(.is-pending)", { hasText: body });
  await row.locator(".message-content").click({ delay: 600 });
  const sheet = mobilePage.getByRole("dialog", { name: "Message actions" });
  await expect(sheet).toBeVisible();

  const geometry = await sheet.evaluate((element) => {
    const box = element.getBoundingClientRect();
    return {
      top: box.top,
      bottom: box.bottom,
      viewportHeight: window.innerHeight,
      clientHeight: element.clientHeight,
      scrollHeight: element.scrollHeight,
      overflowY: getComputedStyle(element).overflowY,
    };
  });
  expect(geometry.top).toBeGreaterThanOrEqual(0);
  expect(geometry.bottom).toBeLessThanOrEqual(geometry.viewportHeight + 0.5);
  expect(geometry.scrollHeight).toBeGreaterThan(geometry.clientHeight);
  expect(geometry.overflowY).toBe("auto");

  const deleteButton = sheet.getByRole("button", { name: "Delete message" });
  await deleteButton.scrollIntoViewIfNeeded();
  await expect(deleteButton).toBeVisible();

  await mobileContext.close();
});

test("a newer realtime event wins over a delayed mutation response", async ({ page }) => {
  const { suffix } = await openReactionChannel(page);
  const row = await sendMessage(page, `Reaction mutation race ${suffix}`);
  const messageID = await row.getAttribute("data-message-id");
  expect(messageID).toBeTruthy();

  let releaseResponse!: () => void;
  let markCommitted!: () => void;
  const responseGate = new Promise<void>((resolve) => {
    releaseResponse = resolve;
  });
  const committed = new Promise<void>((resolve) => {
    markCommitted = resolve;
  });
  await page.route(`**/api/messages/${messageID}/reactions`, async (route) => {
    const response = await route.fetch();
    markCommitted();
    await responseGate;
    await route.fulfill({ response });
  });

  await pickReaction(row, "👍");
  await committed;
  await expect(row.getByRole("button", { name: "Add reaction" })).toBeDisabled();
  const removal = await page.request.delete(
    `/api/messages/${messageID}/reactions/${encodeURIComponent("👍")}`,
  );
  expect(removal.ok()).toBe(true);
  await expect(row.getByRole("button", { name: "👍 — 1 reaction" })).toHaveCount(0);
  releaseResponse();
  await expect(row.getByRole("button", { name: "Add reaction" })).toBeEnabled();
  await expect(row.getByRole("button", { name: "👍 — 1 reaction" })).toHaveCount(0);
});

test("ambiguous failures recover server state and preserve newer realtime reactions", async ({
  page,
}) => {
  const { suffix } = await openReactionChannel(page);
  const row = await sendMessage(page, `Reaction recovery proof ${suffix}`);
  const messageID = await row.getAttribute("data-message-id");
  expect(messageID).toBeTruthy();

  await page.route(`**/api/messages/${messageID}/reactions`, async (route) => {
    await route.fetch();
    await route.fulfill({
      status: 409,
      contentType: "application/json",
      body: '{"error":"reaction already exists"}',
    });
  });
  await pickReaction(row, "👍");
  await expect(row.getByRole("button", { name: "👍 — 1 reaction" })).toBeVisible();
  await expect(row.getByRole("status")).toHaveCount(0);
  await expect(row.getByRole("button", { name: "Add reaction" })).toBeEnabled();

  await page.unroute(`**/api/messages/${messageID}/reactions`);
  const cleanup = await page.request.delete(
    `/api/messages/${messageID}/reactions/${encodeURIComponent("👍")}`,
  );
  expect(cleanup.ok()).toBe(true);
  await expect(row.getByRole("button", { name: "👍 — 1 reaction" })).toHaveCount(0);

  let releaseFailure!: () => void;
  const failureGate = new Promise<void>((resolve) => {
    releaseFailure = resolve;
  });
  await page.route(`**/api/messages/${messageID}/reactions`, async (route) => {
    await failureGate;
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: '{"error":"failed"}',
    });
  });
  await page.route(`**/api/messages/${messageID}`, async (route) => {
    await route.fulfill({
      status: 500,
      contentType: "application/json",
      body: '{"error":"recovery failed"}',
    });
  });

  await pickReaction(row, "👍");
  await expect(row.getByRole("button", { name: "👍 — 1 reaction" })).toBeVisible();
  const newerReaction = await page.request.post(`/api/messages/${messageID}/reactions`, {
    data: { emoji: "❤️" },
  });
  expect(newerReaction.ok()).toBe(true);
  await expect(row.getByRole("button", { name: "❤️ — 1 reaction" })).toBeVisible();
  releaseFailure();
  await expect(row.getByRole("button", { name: "👍 — 1 reaction" })).toHaveCount(0);
  await expect(row.getByRole("button", { name: "❤️ — 1 reaction" })).toBeVisible();
  await expect(row.getByRole("status")).toContainText("failed");
});

test("a reaction event received during pagination survives the stale page response", async ({
  page,
}) => {
  const { channel } = await openReactionChannel(page);
  const created: Array<{ id: string }> = [];
  for (let index = 0; index < 140; index++) {
    const response = await page.request.post(`/api/channels/${channel.id}/messages`, {
      data: { body: `reaction-history-${String(index).padStart(3, "0")}` },
    });
    expect(response.ok()).toBe(true);
    const data = (await response.json()) as { message: { id: string } };
    created.push(data.message);
  }
  await page.reload();
  await waitForAppReady(page);

  await page.getByLabel("Search messages").fill("reaction-history-010");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  const result = page
    .getByLabel("Search results")
    .locator(".search-result", { hasText: "reaction-history-010" });
  await expect(result).toBeVisible();

  let releasePage!: () => void;
  let markPageCaptured!: () => void;
  const pageGate = new Promise<void>((resolve) => {
    releasePage = resolve;
  });
  const pageCaptured = new Promise<void>((resolve) => {
    markPageCaptured = resolve;
  });
  await page.route(`**/api/channels/${channel.id}/messages?around_seq=*`, async (route) => {
    const response = await route.fetch();
    markPageCaptured();
    await pageGate;
    await route.fulfill({ response });
  });

  await result.click();
  await pageCaptured;
  const reactionResponse = await page.request.post(`/api/messages/${created[10].id}/reactions`, {
    data: { emoji: "👀" },
  });
  expect(reactionResponse.ok()).toBe(true);
  releasePage();

  const target = page.locator(`[data-message-id="${created[10].id}"]`);
  const neighbor = page.locator(`[data-message-id="${created[11].id}"]`);
  await expect(target.getByRole("button", { name: "👀 — 1 reaction" })).toBeVisible();
  await expect(neighbor).toBeVisible();
});

test("thread roots and replies share reaction controls and realtime state", async ({ page }) => {
  const { suffix } = await openReactionChannel(page);
  const rootRow = await sendMessage(page, `Reaction thread root ${suffix}`);
  const rootID = await rootRow.getAttribute("data-message-id");
  expect(rootID).toBeTruthy();
  const replyResponse = await page.request.post(`/api/messages/${rootID}/thread/replies`, {
    data: { body: `Reaction thread reply ${suffix}` },
  });
  expect(replyResponse.ok()).toBe(true);
  const { message: reply } = (await replyResponse.json()) as { message: { id: string } };

  await rootRow.click();
  const thread = page.locator(".thread.open");
  await expect(thread).toBeVisible();
  const threadRoot = thread.locator(".thread-root");
  const threadReply = thread.locator(`[data-message-id="${reply.id}"]`);
  await expect(threadRoot.getByRole("button", { name: "Add reaction" })).toBeVisible();
  await expect(threadReply.getByRole("button", { name: "Add reaction" })).toBeVisible();

  await pickReaction(threadRoot, "🚀");
  await expect(threadRoot.getByRole("button", { name: "🚀 — 1 reaction" })).toBeVisible();
  await expect(rootRow.getByRole("button", { name: "🚀 — 1 reaction" })).toBeVisible();

  const replyReaction = await page.request.post(`/api/messages/${reply.id}/reactions`, {
    data: { emoji: "✅" },
  });
  expect(replyReaction.ok()).toBe(true);
  await expect(threadReply.getByRole("button", { name: "✅ — 1 reaction" })).toBeVisible();
});

test("touch thread messages use accessible action sheets instead of persistent controls", async ({
  browser,
  page,
}) => {
  const { suffix, workspace, channel } = await openReactionChannel(page);
  const rootBody = `Touch thread root ${suffix}`;
  const replyBody = `Touch thread reply ${suffix}`;
  const rootRow = await sendMessage(page, rootBody);
  const rootID = await rootRow.getAttribute("data-message-id");
  expect(rootID).toBeTruthy();
  const replyResponse = await page.request.post(`/api/messages/${rootID}/thread/replies`, {
    data: { body: replyBody },
  });
  expect(replyResponse.ok()).toBe(true);
  const { message: reply } = (await replyResponse.json()) as { message: { id: string } };

  const mobileContext = await browser.newContext({
    baseURL: new URL(page.url()).origin,
    hasTouch: true,
    isMobile: true,
    viewport: { width: 390, height: 844 },
  });
  const mobilePage = await mobileContext.newPage();
  await mobilePage.goto(`/app/${workspace.route_id}/${channel.route_id}`);
  await waitForAppReady(mobilePage);
  await mobilePage.locator(".message-row:not(.is-pending)", { hasText: rootBody }).click();

  const thread = mobilePage.locator(".thread.open");
  const threadRoot = thread.locator(`[data-message-id="${rootID}"]`);
  const threadReply = thread.locator(`[data-message-id="${reply.id}"]`);
  await expect(threadRoot).toBeVisible();
  await expect(threadReply).toBeVisible();
  await expect(threadRoot.getByRole("button", { name: "Add reaction" })).toBeHidden();
  await expect(threadRoot.getByRole("button", { name: "Reply" })).toBeHidden();
  await expect(threadReply.getByRole("button", { name: "Add reaction" })).toBeHidden();

  const rootMore = threadRoot.getByRole("button", { name: "More actions" });
  const sheet = mobilePage.getByRole("dialog", { name: "Message actions" });
  const hiddenGeometry = await rootMore.evaluate((element) => {
    const rect = element.getBoundingClientRect();
    return { width: rect.width, height: rect.height };
  });
  expect(hiddenGeometry.width).toBeLessThanOrEqual(1);
  expect(hiddenGeometry.height).toBeLessThanOrEqual(1);

  await rootMore.focus();
  await mobilePage.keyboard.press("Enter");
  await expect(sheet).toBeVisible();
  await expect(sheet.getByRole("button", { name: "Open thread" })).toHaveCount(0);
  await sheet.getByRole("button", { name: "React with 👍" }).click();
  await expect(sheet).toBeHidden();
  await expect(rootMore).toBeFocused();
  await expect(threadRoot.getByRole("button", { name: "👍 — 1 reaction" })).toBeVisible();

  await threadReply.locator(".markdown").click({ delay: 600 });
  await expect(sheet).toBeVisible();
  await expect(sheet.getByRole("button", { name: "Reply" })).toBeVisible();
  await mobilePage.keyboard.press("Escape");
  await expect(sheet).toBeHidden();

  await mobileContext.close();
});
