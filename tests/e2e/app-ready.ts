import { expect, type Page } from "@playwright/test";

export async function waitForAppReady(page: Page) {
  // /app can connect realtime before canonical route application finishes.
  // Wait until boot has completed so interactions survive any route remount.
  await expect(page.locator('.shell[data-app-ready="true"]')).toBeVisible();
}
