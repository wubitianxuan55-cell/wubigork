// spaceBindings.test.ts — S2.3 bridge 分面分类表（docs/gaea-space-shell-design.md §7）
import { describe, expect, it } from "vitest";
import {
  GAEA_METHOD_FACETS,
  WORK_BINDING_NAMES,
  PLAY_BINDING_NAMES,
  SHARED_BINDING_NAMES,
  bindingSpaceOf,
  isBindingAllowedInSpace,
} from "./spaceBindings";

describe("spaceBindings 分类表（S2.3 bridge 分面）", () => {
  it("AppBindings 全部方法被显式分类（satisfies 编译期已兜底；此处锁数量）", () => {
    const total = Object.keys(GAEA_METHOD_FACETS).length;
    expect(total).toBe(280); // 与 keyof AppBindings 一致（satisfies 编译期钉死；v4.28 + PptxOutline/GaeaBrowserObserve；v4.64 + SubagentFollowUp；v4.66 + PromoteSubagent；v4.78 + TaskKill；v4.80 + ContextNodeDetail；v4.86 + GaeaGit*7；v4.94 + SubagentContextView；v4.99 直调转正 + ImageHubAssets/ChapterArtList）
  });

  it("work/play/shared 三数组两两无交集且之和 + independent = 总数", () => {
    const work = new Set<string>(WORK_BINDING_NAMES);
    const play = new Set<string>(PLAY_BINDING_NAMES);
    const shared = new Set<string>(SHARED_BINDING_NAMES);
    for (const n of work) {
      expect(play.has(n)).toBe(false);
      expect(shared.has(n)).toBe(false);
    }
    for (const n of play) {
      expect(shared.has(n)).toBe(false);
    }
    const independent = Object.keys(GAEA_METHOD_FACETS).filter(
      (k) => GAEA_METHOD_FACETS[k as keyof typeof GAEA_METHOD_FACETS] === "independent",
    );
    expect(work.size + play.size + shared.size + independent.length).toBe(
      Object.keys(GAEA_METHOD_FACETS).length,
    );
  });

  it("SPACE_BINDINGS 解析与抽查：办公/任务→work，轻语→play，空间/模型→shared", () => {
    expect(bindingSpaceOf("XlsxPlanEdit")).toBe("work");
    expect(bindingSpaceOf("TaskList")).toBe("work");
    expect(bindingSpaceOf("WhisperMemories")).toBe("play");
    expect(bindingSpaceOf("GaeaSpaceActivate")).toBe("shared");
    expect(bindingSpaceOf("GaeaSpaceList")).toBe("shared");
    expect(bindingSpaceOf("UnifiedSearch")).toBe("shared"); // scope 参数隔离（S1.2-C）
    expect(bindingSpaceOf("StartProgrammingWeb")).toBe("independent");
  });

  it("isBindingAllowedInSpace：shared/independent 两空间可达，work/play 仅所属空间", () => {
    expect(isBindingAllowedInSpace("XlsxPlanEdit", "work")).toBe(true);
    expect(isBindingAllowedInSpace("XlsxPlanEdit", "play")).toBe(false);
    expect(isBindingAllowedInSpace("WhisperMemories", "play")).toBe(true);
    expect(isBindingAllowedInSpace("WhisperMemories", "work")).toBe(false);
    expect(isBindingAllowedInSpace("GaeaSpaceList", "work")).toBe(true);
    expect(isBindingAllowedInSpace("GaeaSpaceList", "play")).toBe(true);
    expect(isBindingAllowedInSpace("StartProgrammingWeb", "work")).toBe(true);
    expect(isBindingAllowedInSpace("StartProgrammingWeb", "play")).toBe(true);
    // 未知名方法兜底 work（合法调用面已由编译期覆盖）
    expect(isBindingAllowedInSpace("NoSuchMethod", "play")).toBe(false);
  });
});
