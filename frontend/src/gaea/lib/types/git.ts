// GitFileStatus 是一条文件状态（git status porcelain v1 的 X/Y 两列展开）。
export interface GitFileStatus {
  path: string;
  x: string;
  y: string;
  staged?: boolean;
  untracked?: boolean;
  deleted?: boolean;
  modified?: boolean;
  renamed?: boolean;
  // G1.7：未合并冲突（任一 U，或 AA/DD/AU/UA/DU/UD）——不可提交，独立分组。
  conflict?: boolean;
  // G1.7：重命名/复制来源路径（R/C 的 porcelain -z 第二字段，工作区内才带）。
  origPath?: string;
}

// GitStatusView 是仓库状态快照（非 Git 仓库时 isRepo=false + error）。
export interface GitStatusView {
  isRepo: boolean;
  branch?: string;
  ahead?: number;
  behind?: number;
  files: GitFileStatus[];
  error?: string;
  // G1 深化：工作区位于 linked worktree（非主检出）时标注 + 主检出路径溯源。
  isLinkedWorktree?: boolean;
  mainWorktreePath?: string;
}

// GitWorktreeView 是一条 worktree（git worktree list --porcelain 展开）。
// G1 深化只做展示：切换=工作区切换器（G2/W1 级产品门），不做 add/remove/prune。
export interface GitWorktreeView {
  path: string;
  head?: string;
  branch?: string;
  isMain?: boolean;
  isBare?: boolean;
  isDetached?: boolean;
  isLocked?: boolean;
  lockedNote?: string;
  isCurrent?: boolean;
}

// GitCommitInfoView 是一条历史提交。
export interface GitCommitInfoView {
  hash: string;
  subject: string;
  author?: string;
  ts?: number;
}

// G2 只读提交图：节点（lane 拓扑由前端按 parents 纯函数计算）。
export interface GitGraphNodeView {
  hash: string;
  short: string;
  subject: string;
  author?: string;
  ts?: number;
  parents?: string[];
  // 装饰原样 token（HEAD -> main / tag: v1 / origin/main）。
  refs?: string[];
}

// GitGraphPageView 是提交图一页：分页提交 + 本地分支清单（筛选用，不 checkout）。
export interface GitGraphPageView {
  commits: GitGraphNodeView[];
  branches: string[];
}

// GitCommitFileView 是提交详情中一个变更文件（numstat；二进制 add/del=-1）。
export interface GitCommitFileView {
  path: string;
  add?: number;
  del?: number;
}

// GitCommitDetailView 是单个提交详情（diff 为全量 unified 文本；合并提交默认空串）。
export interface GitCommitDetailView {
  hash: string;
  subject: string;
  body?: string;
  author?: string;
  ts?: number;
  parents?: string[];
  refs?: string[];
  files: GitCommitFileView[];
  diff: string;
}

