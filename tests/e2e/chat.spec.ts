import { expect, test } from "@playwright/test";
import { execFileSync } from "node:child_process";

test("sends messages, searches, uploads, opens a thread, and creates a DM", async ({ page }) => {
  const consoleMessages: string[] = [];
  page.on("console", (message) => consoleMessages.push(`${message.type()}: ${message.text()}`));
  page.on("pageerror", (error) => consoleMessages.push(`pageerror: ${error.message}`));
  const workspacesResponse = await page.request.get("/api/workspaces");
  const workspaces = (await workspacesResponse.json()) as { workspaces: { id: string }[] };
  const workspaceId = workspaces.workspaces[0].id;
  const secondUserId = execFileSync(
    "go",
    [
      "run",
      "./apps/api/cmd/clickclack",
      "admin",
      "user",
      "create",
      "--data",
      "./data/e2e",
      "--workspace",
      workspaceId,
      "--name",
      "Second User",
      "--email",
      "second@example.com",
    ],
    { cwd: process.cwd(), encoding: "utf8" },
  ).trim();

  await page.goto("/");

  await page.getByRole("button", { name: "# general" }).click();
  await expect(page.getByRole("heading", { name: "#general" })).toBeVisible();

  await page.getByLabel("Message body").fill("hello **playwright**");
  await page.getByRole("button", { name: "Send" }).click();
  await expect(
    page.locator(".markdown").filter({ hasText: "hello playwright" }),
    consoleMessages.join("\n"),
  ).toBeVisible({
    timeout: 5_000,
  });

  await page.getByLabel("Search messages").fill("playwright");
  await page.getByRole("button", { name: "Search" }).click();
  await expect(page.getByLabel("Search results").getByText("hello **playwright**")).toBeVisible();

  await page.getByLabel("Upload file").setInputFiles({
    name: "note.txt",
    mimeType: "text/plain",
    buffer: Buffer.from("uploaded from playwright"),
  });
  await expect(page.getByText("note.txt")).toBeVisible();
  await page.getByLabel("Message body").fill("message with upload");
  await page.getByRole("button", { name: "Send" }).click();
  await expect(page.locator(".markdown").filter({ hasText: "message with upload" })).toBeVisible();

  await page.getByRole("button", { name: "Open thread" }).first().click();
  await expect(page.getByText("Thread", { exact: true })).toBeVisible();

  await page.getByLabel("Reply body").fill("thread _reply_");
  await page.getByRole("button", { name: "Reply" }).click();
  await expect(page.locator(".reply .markdown").filter({ hasText: "thread reply" })).toBeVisible();

  await page.reload();
  await expect(page.locator(".markdown").filter({ hasText: "hello playwright" })).toBeVisible();

  await page.getByLabel("DM member user ID").fill(secondUserId);
  await page.getByLabel("DM member user ID").press("Enter");
  await expect(page.getByRole("heading", { name: /Second User/ })).toBeVisible();
  await page.getByLabel("Message body").fill("private playwright");
  await page.getByRole("button", { name: "Send" }).click();
  await expect(page.locator(".markdown").filter({ hasText: "private playwright" })).toBeVisible();
});
