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

