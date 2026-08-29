// MemoryHubPage 三脑检索 scope 传递测试（S1.2-C）：
// runSearch → app.UnifiedSearch(q, scope, 8)；默认 scope=当前生效空间（work），
// SpaceChip 激活广播后随 play，切「全部」传 ""（旧行为，显式选择才跨空间）。
// GraphView（3D 图谱）在 jsdom 下替换为桩，聚焦 hub strip 检索交互。
import { describe, expect, it, beforeEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor, act } from "@testing-library/react";
import MemoryHubPage from "./MemoryHubPage";
import { noteSpaceActivated } from "../gaea/lib/useSpaceScope";
import type { SpaceActiveView } from "../gaea/lib/types";

const h = vi.hoisted(() => {
  let unifiedCalls: unknown[][] = [];
  let spaceImpl: (() => Promise<SpaceActiveView>) | undefined;
  return {
    recordUnified: (args: unknown[]) => {
      unifiedCalls.push(args);
    },
    unifiedCalls: () => unifiedCalls,
    resetUnifiedCalls: () => {
      unifiedCalls = [];
    },
    setSpaceImpl: (fn: (() => Promise<SpaceActiveView>) | undefined) => {
      spaceImpl = fn;
    },
    spaceImpl: () => spaceImpl,
  };
});

vi.mock("../gaea/lib/bridge", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../gaea/lib/bridge")>();
  return {
    ...actual,
    app: new Proxy(actual.app, {
      get(target, prop, receiver) {
        if (prop === "GaeaSpaceActive") {
          const override = h.spaceImpl();
          if (override) return override;
        }
        if (prop === "UnifiedSearch") {
          const orig = Reflect.get(target, prop, receiver) as (...a: unknown[]) => unknown;
          return (...args: unknown[]) => {
            h.recordUnified(args);
            return orig(...args);
          };
        }
        return Reflect.get(target, prop, receiver);
      },
    }),
  };
});

vi.mock("../gaea/components/memoryhub/GraphView", () => ({
  GraphView: () => null,
}));

describe("MemoryHubPage 三脑检索 scope（S1.2-C）", () => {
  beforeEach(() => {
    // GaeaSpaceActive 即时返回 work（真实 mock 带 80ms delay），保持确定性。
    h.setSpaceImpl(async () => ({
      space: "work",
      modeOn: true,
      exportsDir: ".gaea/exports",
      workDir: ".gaea/work",
    }));
    h.resetUnifiedCalls();
  });

  it("默认 scope=当前生效空间（work）：UnifiedSearch 收到 (q, \"work\", 8)", async () => {
    render(<MemoryHubPage />);
    fireEvent.change(screen.getByPlaceholderText("三脑检索 · 工作区资料"), {
      target: { value: "振动锤" },
    });
    fireEvent.click(screen.getByRole("button", { name: "检索" }));

    await waitFor(() => expect(h.unifiedCalls().length).toBeGreaterThan(0));
    const [q, scope, topN] = h.unifiedCalls()[0];
    expect(q).toBe("振动锤");
    expect(scope).toBe("work");
    expect(topN).toBe(8);
  });

  it("切「全部」后 UnifiedSearch 收到 scope=\"\"（旧行为，显式选择才跨空间）", async () => {
    render(<MemoryHubPage />);
    fireEvent.click(screen.getByRole("radio", { name: "全部" }));
    fireEvent.change(screen.getByPlaceholderText("三脑检索 · 工作区资料"), {
      target: { value: "振动锤" },
    });
    fireEvent.click(screen.getByRole("button", { name: "检索" }));

    await waitFor(() => expect(h.unifiedCalls().length).toBeGreaterThan(0));
    const [q, scope, topN] = h.unifiedCalls()[0];
    expect(q).toBe("振动锤");
    expect(scope).toBe("");
    expect(topN).toBe(8);
  });

  it("SpaceChip 激活 play 后（noteSpaceActivated 广播）默认 scope 随之变为 play", async () => {
    render(<MemoryHubPage />);
    act(() => {
      noteSpaceActivated({
        space: "play",
        modeOn: true,
        exportsDir: ".gaea/play/exports",
        workDir: ".gaea/play/work",
      });
    });
    fireEvent.change(screen.getByPlaceholderText("三脑检索 · 工作区资料"), {
      target: { value: "游戏" },
    });
    fireEvent.click(screen.getByRole("button", { name: "检索" }));

    await waitFor(() => expect(h.unifiedCalls().length).toBeGreaterThan(0));
    const [, scope] = h.unifiedCalls()[0];
    expect(scope).toBe("play");
  });
});
