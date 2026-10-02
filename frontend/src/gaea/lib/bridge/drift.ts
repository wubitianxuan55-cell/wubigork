// drift.ts — 编译期绑定面漂移检查（T6-10.3）。
import { bindingNames } from "../bindingNames";
import { legacyBindings } from "../legacyBindings";
import type { AppBindings } from "./appBindings";
import { gaeaToGaea } from "./mappings";

// ── 编译期绑定面漂移检查（T6-10.3）──────────────────────────────────────
// bindingNames 由 scripts/gen_bindings -names 生成（Go 侧全部导出绑定方法名），
// CI（scripts/check-bindings-drift.ps1）保证它与 Go 侧同步。这里用类型级双向断言
// 校验前端绑定面（AppBindings + gaeaToGaea）与 bindingNames 一致：
//
//   · 方向一 _CheckAppBindingsHasNoStray：AppBindings 声明并经 gaeaToGaea 映射后
//     实际调用的每个 Go 方法名必须真实存在于 bindingNames。Go 侧改名/删除方法，
//     或 gaeaToGaea 映射目标写错（历史上有 6 处：KeepWarm*/PreloadPlan*/AgentMode/
//     SummarizeFile/Subagent*），都会在此报错。
//   · 方向二 _CheckAppBindingsCoversAll：bindingNames 里每个方法名必须被
//     AppBindings 消费（含 gaeaToGaea 映射）或显式排除。Go 侧新增绑定而前端
//     未认领时在此报错（补 AppBindings/gaeaToGaea，或列入下方两个清单并注明理由）。
//
// S2-3「App 绑定面拆分」后 Go 侧方法分为两半：经 AppBindings 消费的 gaea UI
// 绑定面，以及 wailsjsCompat / window.go.app.App.* 直接调用的 legacy 绑定面
// （小说/聊天/语音/绘图/角色库/旧 store 等）。后者不再手写（FE4-04）：由
// scripts/gen_bindings -legacy-ts 生成到 legacyBindings.ts（= Go 全集减 AppBindings
// 认领集），与本文件旧手写联合逐名对齐后退役——手写时代它比真补集多挂 10 个
// 已被认领的名字（GaeaSemanticIndexStatus/NovelOutlineReconstructStart/
// NovelOutlineReconstructTaskGet/NovelBookSource* 7 名），重叠会把方向一的
// 「Go 删除 → 红」架空，这正是改生成的原因之一。
// 修复提示：Go 方法新增/改名/删除 → 先重新生成 bindingNames.ts，再按报错调整；
// 涉及 legacy 面时加跑 go run ./scripts/gen_bindings -legacy-ts。
/** 泛型工具：T 必须是 never（空联合），否则编译错误。 */
type AssertNever<T extends never> = T;

/** Go 侧全部绑定方法名（与 bindingNames.ts 同步）。 */
type BindingName = (typeof bindingNames)[number];

/** AppBindings 声明的方法经 gaeaToGaea 映射后在 Go 侧实际调用的方法名集合。 */
type AppBindingTarget = {
  [K in keyof AppBindings]: K extends keyof typeof gaeaToGaea
    ? (typeof gaeaToGaea)[K]
    : K;
}[keyof AppBindings];

/** AppBindings mock-only：Go 侧无对应绑定方法（仅 dev mock 提供）。
 *  导出运行时清单供 spaceBindings.test.ts 的推导断言复用（单源）。
 *  FE3-04：真机路径对清单内名字由 bridge/proxy.ts fail-fast 显式拒绝
 *  （BridgeError code=MockOnlyBinding），不再 undefined→TypeError→假可重试。 */
export const MOCK_ONLY_NAMES = [
  "Compact", // 无 Go 绑定；上下文压缩由后端会话事件自动执行，无手动压缩绑定
  // FE3-04 摘除三名（零前端调用者，已从 AppBindings/GAEA_METHOD_FACETS/mock
  // 三处整体移除）：SetSubagentTemperature（Go 从未实现）、SetEffort（实际走
  // GaeaSetSubagentEffort）、SetSubagentModel（实际走 GaeaSetSubagentModelForSkill）。
] as const;

/** AppBindings mock-only：Go 侧无对应绑定方法（仅 dev mock 提供）。 */
type MockOnlyNames = (typeof MOCK_ONLY_NAMES)[number];

/** legacy 绑定面：Go 侧存在但不经 AppBindings 消费（wailsjsCompat 直接调用）。
 *  FE4-04 起由 scripts/gen_bindings -legacy-ts 生成（勿手改），此处只做类型派生；
 *  下方锁三/锁四保证生成物不过期、不与认领集重叠。 */
type LegacySurfaceNames = (typeof legacyBindings)[number];

/** 显式排除 = mock-only + legacy 绑定面。 */
type ExcludeNames = MockOnlyNames | LegacySurfaceNames;

// 方向一：AppBindings 声明的每个绑定（映射后）必须真实存在于 Go 绑定清单。
// 报错 → Go 侧方法被改名/删除，或 gaeaToGaea 映射目标写错。
/** @public 编译期绑定漂移锁（AssertNever 契约，见文件头两方向说明）。 */
export type _CheckAppBindingsHasNoStray = AssertNever<
  Exclude<AppBindingTarget, BindingName | ExcludeNames>
>;

// 方向二：Go 绑定清单的每个方法名必须被 AppBindings 消费或显式排除。
// 报错 → Go 侧新增绑定无人认领（补 AppBindings/gaeaToGaea 或再生 legacy 清单：
// go run ./scripts/gen_bindings -legacy-ts）。
/** @public 编译期绑定漂移锁。 */
export type _CheckAppBindingsCoversAll = AssertNever<
  Exclude<BindingName, AppBindingTarget | ExcludeNames>
>;

// 锁三（FE4-04）：legacy 生成清单不过期——清单里每个名字必须仍是真实 Go 绑定。
// 报错 → Go 侧删除绑定后未再生 legacyBindings.ts（或该文件被手改）。
/** @public 编译期 legacy 清单过期锁。 */
export type _CheckLegacyNoStale = AssertNever<
  Exclude<LegacySurfaceNames, BindingName>
>;

// 锁四（FE4-04）：legacy 生成清单不与 AppBindings 认领集重叠——名字被认领后
// 必须移出 legacy 清单。重叠会架空方向一：被认领名经 ExcludeNames 逃过
// 「Go 删除 → 方向一红」的检查（手写时代实测 10 名在册重叠，见文件头）。
/** @public 编译期 legacy 清单重叠锁。 */
export type _CheckLegacyNoOverlap = AssertNever<
  Extract<LegacySurfaceNames, AppBindingTarget>
>;

