import { expect, test } from "@playwright/test";
import { waitForAppReady } from "./app-ready";

// Appearance prefs are device-local: the settings modal writes localStorage
// and data attributes on <html>; base.css maps those to color-scheme and
// board token overrides. These tests drive the real UI and assert the
// attribute + persistence contract.

async function openAppearanceSettings(page: import("@playwright/test").Page) {
  await page.goto("/app");
  await waitForAppReady(page);
  const accountSettings = page.getByRole("button", { name: /account settings/i });
  const modal = page.getByRole("dialog", { name: "Account settings" });
  const appearanceHeading = modal.getByRole("heading", { name: "Appearance" });
  const mobileNavigation = page.getByRole("button", { name: "Toggle navigation" });
  if (
    (await mobileNavigation.isVisible()) &&
    (await mobileNavigation.getAttribute("aria-expanded")) !== "true"
  ) {
    await mobileNavigation.click();
    await expect(mobileNavigation).toHaveAttribute("aria-expanded", "true");
  }
  await expect(accountSettings).toBeVisible();
  await accountSettings.click();
  await expect(modal.getByRole("heading", { name: "Profile settings" })).toBeVisible();
  await modal.getByRole("button", { name: "Appearance" }).click();
  await expect(appearanceHeading).toBeVisible();
}

test("forced color mode applies instantly and survives reload", async ({ page }) => {
  await openAppearanceSettings(page);

  const html = page.locator("html");
  await expect(html).not.toHaveAttribute("data-color-mode");

  await page.getByRole("radio", { name: "Dark" }).click();
  await expect(html).toHaveAttribute("data-color-mode", "dark");
  // color-scheme pins to dark, so light-dark() resolves the dark background.
  await expect
    .poll(() => page.evaluate(() => getComputedStyle(document.documentElement).colorScheme))
    .toBe("dark");

  await page.reload();
  // The app.html boot script applies the stored mode before hydration.
  await expect(html).toHaveAttribute("data-color-mode", "dark");

  await openAppearanceSettings(page);
  await page.getByRole("radio", { name: "System" }).click();
  await expect(html).not.toHaveAttribute("data-color-mode");
});

test("board theme retunes the app palette and survives reload", async ({ page }) => {
  await openAppearanceSettings(page);

  const html = page.locator("html");
  const accentOf = () =>
    page.evaluate(() =>
      getComputedStyle(document.documentElement).getPropertyValue("--accent").trim(),
    );
  const signalAccent = await accentOf();

  await page.getByRole("radio", { name: /^Ember/ }).click();
  await expect(html).toHaveAttribute("data-board", "ember");
  await expect.poll(accentOf).not.toBe(signalAccent);

  await page.reload();
  await expect(html).toHaveAttribute("data-board", "ember");

  await openAppearanceSettings(page);
  await page.getByRole("radio", { name: /^Signal/ }).click();
  await expect(html).not.toHaveAttribute("data-board");
  await expect.poll(accentOf).toBe(signalAccent);
});

test("message layout switches to outlined chains and survives reload", async ({ page }) => {
  await openAppearanceSettings(page);

  const html = page.locator("html");
  await expect(html).not.toHaveAttribute("data-message-layout");
  await expect(page.getByRole("radio", { name: /^Standard/ })).toHaveAttribute(
    "aria-checked",
    "true",
  );

  await page.getByRole("radio", { name: /^Outlined chains/ }).click();
  await expect(html).toHaveAttribute("data-message-layout", "outlined");
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("clickclack:message-layout:v1")))
    .toBe("outlined");

  await page.reload();
  await expect(html).toHaveAttribute("data-message-layout", "outlined");

  await openAppearanceSettings(page);
  await page.getByRole("radio", { name: /^Standard/ }).click();
  await expect(html).not.toHaveAttribute("data-message-layout");
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("clickclack:message-layout:v1")))
    .toBeNull();
});

test("density switches to compact and survives reload", async ({ page }) => {
  await openAppearanceSettings(page);

  const html = page.locator("html");
  await expect(html).not.toHaveAttribute("data-density");
  await expect(page.getByRole("radio", { name: /^Comfortable/ })).toHaveAttribute(
    "aria-checked",
    "true",
  );

  await page.getByRole("radio", { name: /^Compact/ }).click();
  await expect(html).toHaveAttribute("data-density", "compact");
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("clickclack:density:v1")))
    .toBe("compact");

  await page.evaluate(() => {
    const fixture = document.createElement("article");
    fixture.id = "appearance-density-fixture";
    fixture.className = "message-group";
    fixture.style.cssText = "position:fixed;left:-10000px;top:0;width:600px;visibility:hidden";
    fixture.innerHTML = `
      <span class="avatar"></span>
      <div class="group-body">
        <header><strong>Fixture</strong><time>2:14 PM</time></header>
        <div class="message-row">
          <span class="row-stamp"></span>
          <div class="message-content"><div class="markdown">Fixture message</div></div>
        </div>
      </div>
    `;
    document.body.append(fixture);
  });

  await page.getByRole("radio", { name: /^Outlined chains/ }).click();
  const messageGroup = page.locator("#appearance-density-fixture");
  const messageRow = messageGroup.locator(".message-row");
  await expect
    .poll(() =>
      messageRow.evaluate((element) => {
        const style = getComputedStyle(element);
        return { marginLeft: style.marginLeft, paddingLeft: style.paddingLeft };
      }),
    )
    .toEqual({ marginLeft: "0px", paddingLeft: "0px" });

  await page.setViewportSize({ width: 390, height: 844 });
  await expect
    .poll(() => messageGroup.evaluate((element) => getComputedStyle(element).paddingLeft))
    .toBe("12px");

  // The pre-paint boot script applies the stored density before hydration.
  await page.reload();
  await expect(html).toHaveAttribute("data-density", "compact");

  await openAppearanceSettings(page);
  await page.getByRole("radio", { name: /^Comfortable/ }).click();
  await expect(html).not.toHaveAttribute("data-density");
  await expect
    .poll(() => page.evaluate(() => localStorage.getItem("clickclack:density:v1")))
    .toBeNull();
});

test("appearance choices support radio keyboard navigation", async ({ page }) => {
  await openAppearanceSettings(page);

  const colorModes = page.getByRole("radiogroup", { name: "Color mode" });
  const system = colorModes.getByRole("radio", { name: "System" });
  const light = colorModes.getByRole("radio", { name: "Light" });
  await expect(system).toHaveAttribute("tabindex", "0");
  await expect(light).toHaveAttribute("tabindex", "-1");
  await system.focus();
  await page.keyboard.press("ArrowRight");
  await expect(light).toBeFocused();
  await expect(light).toHaveAttribute("aria-checked", "true");
  await expect(system).toHaveAttribute("tabindex", "-1");

  const boards = page.getByRole("radiogroup", { name: "Board theme" });
  const signal = boards.getByRole("radio", { name: /^Signal/ });
  const iris = boards.getByRole("radio", { name: /^Iris/ });
  await signal.focus();
  await page.keyboard.press("ArrowLeft");
  await expect(iris).toBeFocused();
  await expect(iris).toHaveAttribute("aria-checked", "true");

  const messageLayouts = page.getByRole("radiogroup", { name: "Message layout" });
  const standard = messageLayouts.getByRole("radio", { name: /^Standard/ });
  const outlined = messageLayouts.getByRole("radio", { name: /^Outlined chains/ });
  await standard.focus();
  await page.keyboard.press("End");
  await expect(outlined).toBeFocused();
  await expect(outlined).toHaveAttribute("aria-checked", "true");
  await page.keyboard.press("Home");
  await expect(standard).toBeFocused();
  await expect(standard).toHaveAttribute("aria-checked", "true");
  await page.keyboard.press("ArrowUp");
  await expect(outlined).toBeFocused();
  await expect(outlined).toHaveAttribute("tabindex", "0");

  const densities = page.getByRole("radiogroup", { name: "Density" });
  const comfortable = densities.getByRole("radio", { name: /^Comfortable/ });
  const compact = densities.getByRole("radio", { name: /^Compact/ });
  await comfortable.focus();
  await page.keyboard.press("ArrowDown");
  await expect(compact).toBeFocused();
  await expect(compact).toHaveAttribute("aria-checked", "true");
});
