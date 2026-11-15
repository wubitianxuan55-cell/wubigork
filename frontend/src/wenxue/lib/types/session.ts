import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

// SessionMeta is one saved session for the history panel.
export interface SessionMeta {
  path: string;
  preview: string;
  title?: string; // user-chosen name; falls back to preview when empty
  turns: number;
  modTime: number; // unix milliseconds
  current: boolean;
  pinned?: boolean; // 置顶会话排在同项目会话最前
  archived?: boolean; // 已归档（在 <sessions>/archive/ 下，可恢复）
  // interrupted=true 表示上次运行中断未完成；恢复会话时后端注入「上次会话中断」
  // 摘要提示并清除该标记（可选字段：后端始终返回，前端对缺失按 false 处理）。
  interrupted?: boolean;
  // S1 空间归属（后端 SessionMeta.spaceId，typesGenerationCheck 漂移校验钉死）。
  spaceId?: string;
}

// ProjectGroup 是侧边栏「项目」分组：一个工作区 + 它的会话列表。
// 由后端 GaeaListProjectSessions 聚合（当前工作区在前，其余为最近打开过的）。
export interface ProjectGroup {
  path: string;
  name: string;
  current: boolean;
  sessions: SessionMeta[];
  archived: SessionMeta[]; // 已归档会话（项目内「已归档」分组）
  modTime: number; // 分组内最近会话时间（unix milliseconds）
}

export type WorkspaceView = WireShape<AppModels.WorkspaceView>;

// SpaceOption 是双空间静态枚举项（GaeaSpaceList，work/play 固定两值）。
export type SpaceOption = WireShape<AppModels.SpaceOption>;

// SpaceActiveView 是当前生效空间视图（GaeaSpaceActive / GaeaSpaceActivate）。
// space.mode=off 时分区整体关闭，space 恒报 work（modeOn=false 标记关闭态）。
export type SpaceActiveView = WireShape<AppModels.SpaceActiveView>;

// SpaceProfileView 是单个空间装配 profile 视图（GaeaSpaceProfiles，模型中心
// 「总闸/空间策略」分区只读消费；gaea 非空且 gaeaOk=false = 引用无法解析）。
export type SpaceProfileView = WireShape<AppModels.SpaceProfileView>;
