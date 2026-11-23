// wenxue.ts — 文学板块（图文）专属绑定面：工作空间限定 C:\AI\tuwen 的
// WenxueB 新增方法（WenxueCreateNovel）。文件面既有方法（ListDirBatch/
// Preview 等）经 mappings 重定向到 WenxueB 同名 fork，不在本接口重复声明。
// 2026-11-15 二次改造隔离刀。
import type { Meta, ProjectGroup, SessionMeta, WenxueMemoryArchivedView, WorkspaceView } from "../types";

// ToolSurfaceViewB/ToolSurfaceEntryB 与 Go app.ToolSurfaceView / control.ToolSurfaceEntry
// 同形（wenxue 树局部重述——跨树 import 会破坏板块隔离）。
export interface ToolSurfaceEntryB {
  name: string;
  description: string;
  source: string;
  hidden: boolean;
}

export interface ToolSurfaceViewB {
  board: string;
  mode: string;
  tools: ToolSurfaceEntryB[];
}

export interface WenxueExtraBindings {
  /** 新建小说工作空间：C:\AI\tuwen\<name>，创建成功后自动切换为当前工作空间。 */
  // WenxueCreateNovel 建书 + 关键信息（infoJson={genre,intro,targetWords,adultLevel}，
  // 空串=全年龄缺省）：落 书籍信息.md 并自动固定注入新会话。
  WenxueCreateNovel(name: string, infoJson: string): Promise<WorkspaceView>;
  // ── 图文脑生命周期（记忆面板优化批，2026-11-16）──
  WenxueMemoryPin(name: string, pinned: boolean): Promise<void>;
  WenxueMemoryArchive(name: string): Promise<void>;
  WenxueMemoryArchivedList(): Promise<WenxueMemoryArchivedView[]>;
  WenxueMemoryUnarchive(name: string): Promise<void>;
  WenxueMemoryInvalidate(name: string, reason: string): Promise<void>;
  WenxueMemoryRestoreInvalidate(name: string): Promise<void>;
  /** 删除 tuwen 根下的一部小说及其整个工作空间。 */
  WenxueDeleteNovel(path: string): Promise<void>;
  /** 当前小说工作空间列表（只含 tuwen 根下一级目录）。 */
  WenxueListWorkspaces(): Promise<WorkspaceView[]>;
  /** 当前小说工作空间的会话。 */
  WenxueListSessions(): Promise<SessionMeta[]>;
  /** 小说目录分组（含尚未产生会话的新工作空间）。 */
  WenxueListProjectSessions(): Promise<ProjectGroup[]>;
  /** 图文侧元信息，Cwd 固定为当前小说目录。 */
  WenxueMeta(): Promise<Meta>;
  /** 在当前小说目录下新建会话。 */
  WenxueNewSession(): Promise<void>;
  /** 能力面板「工具」tab 装配面（2026-11-21 面板缝）：主注册表全量活取
   *  （描述=模型可见口径，含藏面）；引擎未就绪时 tools 空。 */
  WenxueToolSurface(): Promise<ToolSurfaceViewB>;
  /** 切换当前小说工作空间，越界路径由后端拒绝。 */
  WenxueSwitchWorkspace(path: string): Promise<string>;
  // ── 私密保护面（2026-11-23 无限制档小说四形态，Go 侧 wenxue_privacy.go）──
  /** 私密总览：PIN/解锁态/遮罩开关/隐藏计数/逐书档位与隐藏位（锁定态隐藏书
   *  名字与路径已由后端置空）。 */
  WenxuePrivacyOverview(): Promise<WenxuePrivacyOverview>;
  /** 设置/修改 PIN（已设 PIN 时 oldPin 必须正确；成功即回锁）。 */
  WenxuePrivacySetPin(pin: string, oldPin: string): Promise<void>;
  /** 会话解锁（连续 5 次失败退避 30s）。 */
  WenxuePrivacyUnlock(pin: string): Promise<boolean>;
  /** 手动回锁（即时生效）。 */
  WenxuePrivacyLock(): Promise<void>;
  /** 失焦自动遮罩总开关。 */
  WenxuePrivacySetMaskOnBlur(on: boolean): Promise<void>;
  /** 单书隐藏开关（同步目录系统隐藏属性）。 */
  WenxuePrivacySetBookHidden(name: string, hidden: boolean): Promise<void>;
}

// WenxuePrivacyBook 单书隐私条目（锁定态隐藏书 Name/Path 为空串）。
export interface WenxuePrivacyBook {
  name: string;
  path: string;
  adultLevel: string;
  hidden: boolean;
  current: boolean;
}

// WenxuePrivacyOverview 私密保护总览（Sidebar 隐私区与遮罩/解锁流数据源）。
export interface WenxuePrivacyOverview {
  pinSet: boolean;
  unlocked: boolean;
  maskOnBlur: boolean;
  hiddenCount: number;
  books: WenxuePrivacyBook[];
}
