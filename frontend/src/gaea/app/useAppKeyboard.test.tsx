import { describe, expect, it, vi } from "vitest";
import { act, cleanup, renderHook } from "@testing-library/react";
import { useAppKeyboard } from "./useAppKeyboard";
import { initialState } from "../lib/store";

function pressEscape() {
  act(() => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
  });
}

function params(overrides: Partial<Parameters<typeof useAppKeyboard>[0]> = {}) {
  return {
    state: initialState,
    closeTopmost: vi.fn(() => true),
    anyOpen: false,
    cancel: vi.fn(() => undefined),
    workspacePanelOpen: false,
    previewFile: null,
    toggleFocus: vi.fn(),
    closeFilePreview: vi.fn(),
    newSessionAndReset: vi.fn(async () => {}),
    openHistory: vi.fn(async () => {}),
    toggleSidebar: vi.fn(),
    toggleWorkspacePanel: vi.fn(),
    setPaletteOpen: vi.fn(),
    ...overrides,
  };
}

describe("useAppKeyboard Esc ladder", () => {
  it("idle outside input enters the guarded backtrack point without stopping", () => {
    const p = params();
    renderHook(() => useAppKeyboard(p));
    pressEscape();
    expect(p.cancel).not.toHaveBeenCalled();
    cleanup();
  });

  it("running outside input stops the active turn", () => {
    const p = params({ state: { ...initialState, running: true } });
    renderHook(() => useAppKeyboard(p));
    pressEscape();
    expect(p.cancel).toHaveBeenCalledTimes(1);
    cleanup();
  });

  it("overlay closes before turn stop", () => {
    const p = params({ anyOpen: true, state: { ...initialState, running: true } });
    renderHook(() => useAppKeyboard(p));
    pressEscape();
    expect(p.closeTopmost).toHaveBeenCalledTimes(1);
    expect(p.cancel).not.toHaveBeenCalled();
    cleanup();
  });

  it("approval modal owns Esc; global handler does not double-close", () => {
    const p = params({
      anyOpen: true,
      state: {
        ...initialState,
        running: true,
        approval: { id: "a1", tool: "bash", subject: "rm" },
      },
    });
    renderHook(() => useAppKeyboard(p));
    pressEscape();
    expect(p.closeTopmost).not.toHaveBeenCalled();
    expect(p.cancel).not.toHaveBeenCalled();
    cleanup();
  });
});
