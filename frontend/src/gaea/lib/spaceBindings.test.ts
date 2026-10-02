// spaceBindings.test.ts — S2.3 bridge 分面分类表（docs/gaea-space-shell-design.md §7）
// FE4-04：总数数量锁（历史口径 435→…→565 的单行魔法数断言）已退役——AppBindings 键 ↔
// facets 键的双向相等由 spaceBindings.ts 的 satisfies Record<keyof AppBindings, BindingSpace>
// + _NoStrayFacet/_NoMissingFacet 编译期钉死（多余键 TS2353、缺键 TS1360，实验核实），
// 运行期数字只会在每次增删时手工磨改；防漏保证改由下方两条推导式用例承担（键都解析到
// 真实绑定 + legacy 清单互补镜像）。以下为退役锁的历史批次记录（考古用）：
// 批次记录：v4.323 提示词工坊 +5（PromptTemplateList/Get/Save/Reset/Preview，t6 首刀
// 模板可编辑覆盖层，CreatePage「提示词工坊」面板，play）→ 数量锁 523。前批：7.3-1 任务收件箱 +4（GaeaTaskInboxList/Save/SetStatus/Delete，双空间
// 首页挂点+四入口接线，shared——隔离由 space 参数承担）→ v4.319 技能调用计数 +1（SkillStats，work）→ 数量锁 518。
// 前批：sin 书源 t2 +6（SinBookSourceSearch/Toc/Download/DownloadCancel/BooksList/
// BookDelete）+ t5 +1：SinBookSourceBookExportEpub（EPUB 导出，play）+ ImportNovelBookEx 转正（sin 送小说导入消费，play）→ 数量锁 489。
// 前批：书源取书 +4（NovelBookSourceSearch/Toc/Import/ImportCancel，书源→拆书导入
// t1，NovelB 门面，play 数据面；t3 失败章补下 +1：NovelBookSourceImportChapters，v4.284）→ 481。前批：平台质量评审 +2（NovelReviewPlatforms/
// NovelChapterReview，v4.282 oh-story 蒸馏 T1，NovelB 门面，play 数据面）→ 472。
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { bindingNames } from "./bindingNames";
import { legacyBindings } from "./legacyBindings";
import { gaeaToGaea } from "./bridge/mappings";
import { MOCK_ONLY_NAMES } from "./bridge/drift";
import {
  GAEA_METHOD_FACETS,
  WORK_BINDING_NAMES,
  PLAY_BINDING_NAMES,
  SHARED_BINDING_NAMES,
  UNKNOWN_BINDING_SPACE,
  bindingSpaceOf,
  isBindingAllowedInSpace,
  isSharedBinding,
  resetUnknownBindingWarningsForTest,
} from "./spaceBindings";

// 未登记名告警去重是模块级状态：每个用例前复位，避免用例间互相掩盖告警。
// 同时把留痕通道捕获下来（不打印），供「未登记名告警一次」与「已登记名零告警」
// 双向断言使用。
const warnCalls: string[] = [];
beforeEach(() => {
  resetUnknownBindingWarningsForTest();
  warnCalls.length = 0;
  vi.spyOn(console, "warn").mockImplementation((...a: unknown[]) => {
    warnCalls.push(a.map(String).join(" "));
  });
});
afterEach(() => {
  vi.restoreAllMocks();
});

describe("spaceBindings 分类表（S2.3 bridge 分面）", () => {
  // FE4-04 数量锁退役：原「expect(Object.keys(...).length).toBe(565)」魔法数断言
  // 已删（单行超长批次注释随之归档到文件头）。键集相等的两种漂移方向都有编译期
  // 红外加运行期推导红：多余键 = satisfies TS2353 + _NoStrayFacet TS2344；缺键 =
  // satisfies TS1360 + _NoMissingFacet TS2344（FE4-04 实验钉死）。
  it("键集由 tsc satisfies 钉死（AppBindings ↔ facets 双向）；此处锁每个键都解析到真实绑定", () => {
    // 推导式断言（非魔法数）：每个键经 gaeaToGaea 解析（无映射键按同名直调）
    // 后必须是真实 Go 绑定，或显式 mock-only。Go 侧删除绑定并再生
    // bindingNames.ts 而本表未同步时 orphaned 非空 → 红（「Go 删了清单没删」
    // 方向的运行期镜像，与 satisfies 的多余键红互补）。
    const bindingSet = new Set<string>(bindingNames);
    const mockOnly = new Set<string>(MOCK_ONLY_NAMES);
    const orphaned = Object.keys(GAEA_METHOD_FACETS).filter((k) => {
      if (k in gaeaToGaea) {
        return !bindingSet.has(gaeaToGaea[k as keyof typeof gaeaToGaea]);
      }
      return !bindingSet.has(k) && !mockOnly.has(k);
    });
    expect(orphaned).toEqual([]);
  });

  it("legacy 清单与认领集互补：drift.ts 四把编译期锁的运行期镜像（FE4-04）", () => {
    // 认领集 = facets 键经 gaeaToGaea 映射（与 drift.ts AppBindingTarget 同式）。
    const bindingSet = new Set<string>(bindingNames);
    const targets = new Set<string>(
      Object.keys(GAEA_METHOD_FACETS).map((k) =>
        k in gaeaToGaea ? gaeaToGaea[k as keyof typeof gaeaToGaea] : k,
      ),
    );
    const legacy = new Set<string>(legacyBindings);
    // 覆盖（镜像方向二 _CheckAppBindingsCoversAll）：Go 绑定名必须被认领或挂
    // legacy 清单——Go 新增绑定后只再生 bindingNames.ts、漏再生
    // legacyBindings.ts 时 uncovered 非空 → 红。
    const uncovered = bindingNames.filter((n) => !targets.has(n) && !legacy.has(n));
    expect(uncovered).toEqual([]);
    // 不过期（镜像锁三 _CheckLegacyNoStale）：legacy 名必须仍是真实 Go 绑定
    // ——Go 删除绑定后未再生 legacyBindings.ts（或被手改）→ 红。
    const stale = [...legacy].filter((n) => !bindingSet.has(n));
    expect(stale).toEqual([]);
    // 不重叠（镜像锁四 _CheckLegacyNoOverlap）：被认领名不得同时挂 legacy 清单
    // （重叠会架空方向一；手写时代实测 10 名在册重叠）→ 红。
    const overlap = [...legacy].filter((n) => targets.has(n));
    expect(overlap).toEqual([]);
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
    // v4.439 起 independent 分面为空集（编程板块删除）；仍按字符串比较以防未来回归
    const independent = Object.keys(GAEA_METHOD_FACETS).filter(
      (k) => String(GAEA_METHOD_FACETS[k as keyof typeof GAEA_METHOD_FACETS]) === "independent",
    );
    expect(independent).toHaveLength(0);
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
    // v4.439 编程板块删除后 independent 分面归零：已移除方法不再出现在分面表
    expect(GAEA_METHOD_FACETS).not.toHaveProperty("StartProgrammingWeb");
    const facets: string[] = Object.values(GAEA_METHOD_FACETS);
    expect(facets).not.toContain("independent");
  });

  it("isBindingAllowedInSpace：shared 两空间可达，work/play 仅所属空间", () => {
    expect(isBindingAllowedInSpace("XlsxPlanEdit", "work")).toBe(true);
    expect(isBindingAllowedInSpace("XlsxPlanEdit", "play")).toBe(false);
    expect(isBindingAllowedInSpace("WhisperMemories", "play")).toBe(true);
    expect(isBindingAllowedInSpace("WhisperMemories", "work")).toBe(false);
    expect(isBindingAllowedInSpace("GaeaSpaceList", "work")).toBe(true);
    expect(isBindingAllowedInSpace("GaeaSpaceList", "play")).toBe(true);
    // 未知名方法：fail-closed（FE3-06）——未登记 ≠ work，两空间一律拒绝
    expect(isBindingAllowedInSpace("NoSuchMethod", "play")).toBe(false);
    expect(isBindingAllowedInSpace("NoSuchMethod", "work")).toBe(false);
  });

  // ── FE3-06：未登记名 fail-closed（原兜底 work，是门面代理唯一运行时防线）──
  it("bindingSpaceOf：已登记名原值返回，未登记名返回哨兵 unknown（不再兜底 work）", () => {
    expect(bindingSpaceOf("XlsxPlanEdit")).toBe("work");
    expect(bindingSpaceOf("WhisperMemories")).toBe("play");
    expect(bindingSpaceOf("GaeaSpaceList")).toBe("shared");
    // 未登记：哨兵，不是 work（改坏锚点：兜底 `?? "work"` → 本断言 FAIL）
    expect(bindingSpaceOf("NoSuchMethod")).toBe(UNKNOWN_BINDING_SPACE);
    expect(bindingSpaceOf("NoSuchMethod")).not.toBe("work");
    // Go 侧真实存在的 184 个未登记绑定之一（audit 名单样本）：同样必须判 unknown
    expect(bindingSpaceOf("AddCustomEngine")).toBe(UNKNOWN_BINDING_SPACE);
    expect(bindingSpaceOf("ApplyBranch")).toBe(UNKNOWN_BINDING_SPACE);
  });

  it("isBindingAllowedInSpace：未登记名在 work/play 两空间都被拒（未知≠白名单）", () => {
    for (const name of ["NoSuchMethod", "AddCustomEngine", "BrainSearch", "ChatGeneral", "notEvenCamelCase"]) {
      expect(isBindingAllowedInSpace(name, "work")).toBe(false);
      expect(isBindingAllowedInSpace(name, "play")).toBe(false);
    }
  });

  it("isSharedBinding：未登记名不得进 sharedApp 门面", () => {
    expect(isSharedBinding("GaeaSpaceList")).toBe(true);
    expect(isSharedBinding("XlsxPlanEdit")).toBe(false);
    expect(isSharedBinding("NoSuchMethod")).toBe(false);
    expect(isSharedBinding("AddCustomEngine")).toBe(false);
  });

  it("未登记名留痕：console.warn 一次（同名字只告警一次）", () => {
    bindingSpaceOf("NoSuchMethodForWarnTest");
    bindingSpaceOf("NoSuchMethodForWarnTest");
    isBindingAllowedInSpace("NoSuchMethodForWarnTest", "work");
    expect(warnCalls.filter((w) => w.includes("NoSuchMethodForWarnTest"))).toHaveLength(1);
    expect(warnCalls[0]).toContain("未登记");
  });

  // 反证：已登记名的门控路径零告警（否则「留痕」会退化为满屏噪声，日志失效）。
  it("已登记名零告警：work/play/shared 三门面解析都不打 warn", () => {
    expect(bindingSpaceOf("XlsxPlanEdit")).toBe("work");
    expect(isBindingAllowedInSpace("WhisperMemories", "play")).toBe(true);
    expect(isBindingAllowedInSpace("GaeaSpaceList", "work")).toBe(true);
    expect(isSharedBinding("GaeaSpaceList")).toBe(true);
    expect(warnCalls).toEqual([]);
  });
});
