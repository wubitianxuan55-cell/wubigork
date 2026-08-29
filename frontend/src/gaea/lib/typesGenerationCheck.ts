// typesGenerationCheck.ts — S2.3「types 生成化」第一刀（机器校验）
//
// wailsjs/go/models.ts 由 Go 结构体生成（wails build 时）；types.ts 中与之
// 重叠的手写 view 类型若漏同步（后端新增字段），会导致前端类型漂移。本文件
// 用编译期断言把「生成模型字段 ⊆ 手写类型字段」钉死：Go 侧新增字段而
// 手写类型未跟上 → tsc 报错，提示同步。
//
// 说明：只做字段名覆盖校验（keyof），不校验类型值/可选性（后者由全量迁移
// 至生成模型的后续 Step 收口）。完整迁移（types.ts re-export 生成模型）为
// S2.3b，见 docs/gaea-space-shell-design.md §7。
import type { app as AppModels, tasks as TaskModels } from "../../../wailsjs/go/models";
import type {
  MemoryHubOverview,
  SemanticHitView,
  SessionMeta,
  SpaceActiveView,
  SpaceOption,
  TaskView,
  UnifiedSearchView,
  WorkspaceSearchHit,
} from "./types";

type Assert<T extends true> = T;
// wails 生成类对含数组字段的模型附加实例方法 convertValues，非数据字段，排除
type GeneratedFields<G> = Exclude<keyof G, "convertValues">;

// 生成模型新增字段 → 手写类型必须补齐（当前已发现：SessionMeta.spaceId）
export type _CheckSessionMeta = Assert<Exclude<GeneratedFields<AppModels.SessionMeta>, keyof SessionMeta> extends never ? true : false>;
export type _CheckSpaceOption = Assert<Exclude<GeneratedFields<AppModels.SpaceOption>, keyof SpaceOption> extends never ? true : false>;
export type _CheckSpaceActiveView = Assert<Exclude<GeneratedFields<AppModels.SpaceActiveView>, keyof SpaceActiveView> extends never ? true : false>;
export type _CheckWorkspaceSearchHit = Assert<Exclude<GeneratedFields<AppModels.WorkspaceSearchHit>, keyof WorkspaceSearchHit> extends never ? true : false>;
export type _CheckMemoryHubOverview = Assert<Exclude<GeneratedFields<AppModels.MemoryHubOverview>, keyof MemoryHubOverview> extends never ? true : false>;
export type _CheckSemanticHitView = Assert<Exclude<GeneratedFields<AppModels.SemanticHitView>, keyof SemanticHitView> extends never ? true : false>;
export type _CheckUnifiedSearchView = Assert<Exclude<GeneratedFields<AppModels.UnifiedSearchView>, keyof UnifiedSearchView> extends never ? true : false>;
export type _CheckTaskView = Assert<Exclude<GeneratedFields<TaskModels.Task>, keyof TaskView> extends never ? true : false>;
