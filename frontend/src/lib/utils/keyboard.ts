/** Observe dismissal after a keyboard has been seen. Call before focusing the input. */
export const observeKeyboardDismissal = (onDismiss: () => void): (() => void) => {
  const viewport = window.visualViewport;
  const listeners = new AbortController();
  let resizeFrame: number | undefined;
  let visible = false;

  const update = (next: boolean) => {
    const dismissed = visible && !next;
    visible = next;
    if (dismissed) onDismiss();
  };

  if (window.matchMedia("(any-pointer: coarse)").matches) {
    const measure = () => ({
      height: viewport?.height ?? window.innerHeight,
      width: viewport?.width ?? window.innerWidth,
      scale: viewport?.scale ?? 1,
    });
    let baseline = measure();
    const resized = () => {
      const current = measure();
      // Rotation and zoom invalidate the comparison, not the user's editing session.
      if (Math.abs(current.width - baseline.width) > 1 || current.scale !== baseline.scale) {
        baseline = current;
        visible = false;
        return;
      }
      baseline.height = Math.max(baseline.height, current.height);
      const shrink = baseline.height - current.height;
      // Ignore browser chrome and wait for a substantial keyboard-sized reduction.
      if (shrink > Math.max(150, baseline.height * 0.2)) update(true);
      // Allow for toolbar changes and rounding when the keyboard closes.
      else if (shrink < 80) update(false);
    };
    (viewport ?? window).addEventListener(
      "resize",
      () => {
        // Safari can update viewport dimensions after dispatching the resize event.
        window.cancelAnimationFrame(resizeFrame ?? 0);
        resizeFrame = window.requestAnimationFrame(resized);
      },
      { signal: listeners.signal },
    );
  }

  return () => {
    listeners.abort();
    window.cancelAnimationFrame(resizeFrame ?? 0);
  };
};
