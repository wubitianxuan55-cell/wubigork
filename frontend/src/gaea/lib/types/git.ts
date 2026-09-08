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
}

// GitCommitInfoView 是一条历史提交。
export interface GitCommitInfoView {
  hash: string;
  subject: string;
  author?: string;
  ts?: number;
}

