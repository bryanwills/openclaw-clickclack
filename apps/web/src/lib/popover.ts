/**
 * Decide whether a popover anchored to `anchor` should open upward.
 *
 * Below-placed popovers inside a scroll container (e.g. `.messages-scroll`)
 * get clipped at the container's bottom edge and end up unreachable under
 * the composer. Flip upward when the space between the anchor and the
 * nearest scrollable ancestor's bottom edge can't fit the popover.
 */
export function shouldOpenUpward(anchor: HTMLElement | undefined | null, needed: number): boolean {
  if (!anchor) return false;
  let boundary = window.innerHeight;
  let node: HTMLElement | null = anchor.parentElement;
  while (node) {
    const overflowY = getComputedStyle(node).overflowY;
    if (overflowY === "auto" || overflowY === "scroll" || overflowY === "hidden") {
      boundary = node.getBoundingClientRect().bottom;
      break;
    }
    node = node.parentElement;
  }
  const rect = anchor.getBoundingClientRect();
  return boundary - rect.bottom < needed && rect.top > needed;
}
