import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { observeKeyboardDismissal } from "./keyboard";

let viewport: EventTarget & { height: number; width: number; scale: number };
let coarse = true;

beforeEach(() => {
  viewport = Object.assign(new EventTarget(), { height: 800, width: 400, scale: 1 });
  coarse = true;
  vi.useFakeTimers();
  vi.stubGlobal("window", {
    visualViewport: viewport,
    requestAnimationFrame: (callback: () => void) => setTimeout(callback, 16),
    cancelAnimationFrame: (id: number) => clearTimeout(id),
    matchMedia: () => ({ matches: coarse }),
  });
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const resize = (height: number) => {
  viewport.height = height;
  viewport.dispatchEvent(new Event("resize"));
  vi.runOnlyPendingTimers();
};
describe("observeKeyboardDismissal", () => {
  it("detects dismissal after a keyboard-sized shrink and removes listeners on cleanup", () => {
    const dismissed = vi.fn();
    const stop = observeKeyboardDismissal(dismissed);
    resize(740); // Browser toolbar changes alone must not arm dismissal.
    resize(800);
    expect(dismissed).not.toHaveBeenCalled();
    resize(500);
    resize(650); // Keyboard animation or a keyboard layout change.
    expect(dismissed).not.toHaveBeenCalled();
    resize(760); // Allow a slightly different toolbar height after dismissal.
    expect(dismissed).toHaveBeenCalledTimes(1);
    stop();
    resize(500);
    resize(800);
    expect(dismissed).toHaveBeenCalledTimes(1);
  });

  it("discards viewport comparisons across rotation and zoom", () => {
    const dismissed = vi.fn();
    const stop = observeKeyboardDismissal(dismissed);
    resize(500);
    viewport.width = 800;
    resize(400);
    resize(500);
    viewport.scale = 2;
    resize(250);
    viewport.scale = 1;
    resize(500);
    expect(dismissed).not.toHaveBeenCalled();
    stop();
  });

  it("leaves the editor open without a software keyboard and on desktop resize", () => {
    coarse = false;
    const dismissed = vi.fn();
    const stop = observeKeyboardDismissal(dismissed);
    resize(400);
    resize(800);
    expect(dismissed).not.toHaveBeenCalled();
    stop();
  });
});
