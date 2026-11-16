// wenxue.ts — 文学板块（图文）专属绑定面：工作空间限定 C:\AI\tuwen 的
// WenxueB 新增方法（WenxueCreateNovel）。文件面既有方法（ListDirBatch/
// Preview 等）经 mappings 重定向到 WenxueB 同名 fork，不在本接口重复声明。
// 2026-11-15 二次改造隔离刀。
import type { Meta, ProjectGroup, SessionMeta, WenxueMemoryArchivedView, WorkspaceView } from "../types";

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
  /** 切换当前小说工作空间，越界路径由后端拒绝。 */
  WenxueSwitchWorkspace(path: string): Promise<string>;
}
