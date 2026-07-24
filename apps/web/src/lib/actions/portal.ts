/**
 * Render an element under document.body.
 *
 * Virtua item wrappers carry `contain: layout style`, which turns each item
 * into a containing block even for `position: fixed` descendants. Overlays
 * that must cover the viewport (e.g. the touch action sheet) escape by
 * teleporting to the body.
 */
export function portal(node: HTMLElement) {
  document.body.appendChild(node);
  return {
    destroy() {
      node.remove();
    },
  };
}
