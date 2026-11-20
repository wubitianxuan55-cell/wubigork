import { describe, expect, it, vi } from "vitest";
import type { Action } from "./controller_state";
import { bindOfficeContextRefresh } from "./controller";

describe("office context event refresh", () => {
  it("rebuilt office controller refreshes ContextUsage", async () => {
    let handler: ((payload: unknown) => void) | undefined;
    const dispatch = vi.fn<(a: Action) => void>();
    const loadContext = vi.fn(async () => ({
      used: 120_000,
      window: 1_000_000,
      compactRatio: 0.8,
      compactLeft: 680_000,
    }));

    bindOfficeContextRefresh(
      dispatch,
      (_channel, next) => {
        handler = next;
        return () => {};
      },
      loadContext,
    );

    handler?.({ window: 1_000_000 });
    await vi.waitFor(() => {
      expect(loadContext).toHaveBeenCalledTimes(1);
      expect(dispatch).toHaveBeenCalledWith({
        type: "context",
        context: {
          used: 120_000,
          window: 1_000_000,
          compactRatio: 0.8,
          compactLeft: 680_000,
        },
      });
    });
  });
});
