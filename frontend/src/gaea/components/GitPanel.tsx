// GitPanel — Git 面板最小集（蒸馏规划 2b，决策门 D3 采纳推荐默认：单仓库
// status / diff / stage / unstage / discard / commit / history，无 push/pull/
// fetch，与源 better-sidebar git tab 范围一致； discard 为破坏性操作走两击
// 确认，沿 v4.78 任务强杀两击先例）。
//
// 数据自取（GaeaGit* 绑定，仓库锚定 gaea 工作区 cwd）；非 Git 仓库/git 未
// 安装时显示诚实空态。diff 渲染复用 ChangesDiff（buildGitDiff 把 unified
// diff 解析为同款 ChangeDiff）。面板语言与工作台既有面板一致使用中文。
import { useCallback, useEffect, useMemo, useState } from "react";
import { Check, ChevronDown, FileText, GitBranch, Loader2, Plus, RefreshCw, Rollback } from "../icons";
import { app } from "../lib/bridge";
import type { GitCommitInfoView, GitFileStatus, GitWorktreeView } from "../lib/types";
import { buildGitDiff } from "../lib/planDiff";
import { ChangesDiff } from "./ChangesDiff";
import { ChangesPanel } from "./ChangesPanel";
import { CommitGraphView } from "./CommitGraphView";
import type { SessionChange } from "../lib/changes";
import { useToast } from "./Toast";
import { acquireGitStatus, useGitStatusStore } from "../lib/gitStatusStore";
import { buildUnifiedChangeTree, countUnifiedChangeFiles, type UnifiedChangeFile } from "../lib/unifiedChangesTree";
import { UnifiedChangesTree } from "./UnifiedChangesTree";

const STAGE_LABELS: Record<string, string> = {
  A: "新增", M: "修改", D: "删除", R: "重命名", C: "复制", T: "类型变更", U: "冲突",
};

function statusLetter(f: GitFileStatus, stagedSide: boolean): string {
  const letter = (stagedSide ? f.x : f.y).trim();
  return letter || "M";
}

function statusLabel(f: GitFileStatus, stagedSide: boolean): string {
  return STAGE_LABELS[statusLetter(f, stagedSide)] ?? "变更";
}

// 两击确认按钮（沿 v4.78 防误杀先例：首击进入确认态，3s 未再击自动复位）。
function ConfirmButton({ label, confirmLabel, onConfirm, disabled }: {
  label: string;
  confirmLabel: string;
  onConfirm: () => void;
  disabled?: boolean;
}) {
  const [armed, setArmed] = useState(false);
  useEffect(() => {
    if (!armed) return;
    const t = window.setTimeout(() => setArmed(false), 3000);
    return () => window.clearTimeout(t);
  }, [armed]);
  return (
    <button
      type="button"
      disabled={disabled}
      aria-pressed={armed}
      className={`inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[10px] transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
        armed ? "border-err/60 bg-err/10 text-err" : "border-border-soft bg-transparent text-fg-dim hover:bg-bg-soft hover:text-fg"
      }`}
      onClick={() => {
        if (armed) {
          setArmed(false);
          onConfirm();
        } else {
          setArmed(true);
        }
      }}
    >
      {armed ? confirmLabel : label}
    </button>
  );
}

export function GitPanel({
  onOpenFile,
  changes = [],
  cwd,
}: {
  onOpenFile?: (path: string) => void;
  changes?: SessionChange[];
  cwd?: string;
}) {
  const toast = useToast();
  const status = useGitStatusStore((s) => s.status);
  const loading = useGitStatusStore((s) => s.loading);
  const statusError = useGitStatusStore((s) => s.error);
  const refreshShared = useGitStatusStore((s) => s.refresh);
  const [expandedPath, setExpandedPath] = useState<string | null>(null);
  const [diffText, setDiffText] = useState<string | null>(null);
  const [diffLoading, setDiffLoading] = useState(false);
  const [diffSide, setDiffSide] = useState<"staged" | "work">("work");
  const [message, setMessage] = useState("");
  const [committing, setCommitting] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [history, setHistory] = useState<GitCommitInfoView[] | null>(null);
  const [lens, setLens] = useState<"unified" | "git" | "session" | "worktrees" | "graph">("git");
  const [unifiedFilter, setUnifiedFilter] = useState<"all" | "git" | "agent">("all");
  // G1 深化：worktree 只读展示（切换=工作区切换器，另立产品门）。
  // panelRevision 是 worktree/提交图镜头共用的刷新序号（刷新钮自增驱动各自重拉）。
  const [worktrees, setWorktrees] = useState<GitWorktreeView[] | null>(null);
  const [worktreesError, setWorktreesError] = useState("");
  const [panelRevision, setPanelRevision] = useState(0);

  // G1.7：冲突文件独立分组（Go 侧 conflict 标记；staged 语义已排除冲突，
  // 这里再过滤一层兜住旧后端/mock 未带标记的 UU 形态）。
  const conflicts = status?.files.filter((f) => f.conflict || f.x.trim() === "U" || f.y.trim() === "U") ?? [];
  const staged = status?.files.filter((f) => f.staged && !f.conflict) ?? [];
  const unstaged = status?.files.filter((f) => !f.staged && !f.untracked && !f.conflict) ?? [];
  const untracked = status?.files.filter((f) => f.untracked && !f.conflict) ?? [];
  const stagedCount = staged.length + unstaged.filter((f) => f.staged).length;
  const gitUnavailable = (status && !status.isRepo) || (!status && statusError !== "" && !loading);
  const unifiedAllNodes = useMemo(
    () => buildUnifiedChangeTree(status, changes, cwd),
    [status, changes, cwd],
  );
  const unifiedNodes = useMemo(
    () => unifiedFilter === "all"
      ? unifiedAllNodes
      : buildUnifiedChangeTree(
          unifiedFilter === "agent" ? null : status,
          unifiedFilter === "git" ? [] : changes,
          cwd,
        ),
    [status, changes, cwd, unifiedAllNodes, unifiedFilter],
  );
  const unifiedAllCount = useMemo(() => countUnifiedChangeFiles(unifiedAllNodes), [unifiedAllNodes]);

  const refresh = useCallback(() => {
    void refreshShared();
    setPanelRevision((r) => r + 1);
  }, [refreshShared]);

  useEffect(() => {
    return acquireGitStatus();
  }, []);

  useEffect(() => {
    if (!historyOpen || history) return;
    let live = true;
    Promise.resolve(app.GaeaGitLog?.(30))
      .then((r) => live && setHistory(Array.isArray(r) ? r : []))
      .catch(() => live && setHistory([]));
    return () => {
      live = false;
    };
  }, [historyOpen, history]);

  // G1 深化：worktree 列表数据面独立于 status（GaeaGitWorktrees）；镜头打开
  // 或刷新钮时拉取，错误诚实落地为空态文案（与 status 同一容错口径）。
  useEffect(() => {
    if (lens !== "worktrees") return;
    let live = true;
    Promise.resolve(app.GaeaGitWorktrees?.())
      .then((r) => {
        if (!live) return;
        setWorktrees(Array.isArray(r) ? r : []);
        setWorktreesError("");
      })
      .catch((e: unknown) => {
        if (!live) return;
        setWorktrees([]);
        setWorktreesError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      live = false;
    };
  }, [lens, panelRevision]);

  const loadDiff = (path: string, staged: boolean) => {
    setDiffText(null);
    setDiffLoading(true);
    void Promise.resolve(app.GaeaGitDiff?.(path, staged))
      .then((d) => setDiffText(typeof d === "string" ? d : "绑定不可用"))
      .catch((e: unknown) => setDiffText(`git diff 失败：${e instanceof Error ? e.message : String(e)}`))
      .finally(() => setDiffLoading(false));
  };

  const openDiff = (f: GitFileStatus) => {
    if (expandedPath === f.path) {
      setExpandedPath(null);
      return;
    }
    setExpandedPath(f.path);
    const side = f.staged ? "staged" : "work";
    setDiffSide(side);
    setDiffText(null);
    setDiffLoading(false);
    // 未跟踪无 diff 语义；冲突文件是合并态（combined diff 非经典 unified），
    // 都走诚实提示而不是拉 diff（G1.7）。
    if (f.untracked || f.conflict) return;
    loadDiff(f.path, side === "staged");
  };

  const toggleUnifiedDiff = (file: { path: string; git?: GitFileStatus }) => {
    if (!file.git) return;
    if (expandedPath === file.path) {
      setExpandedPath(null);
      return;
    }
    setExpandedPath(file.path);
    const side = file.git.y.trim() !== "" ? "work" : "staged";
    setDiffSide(side);
    loadDiff(file.path, side === "staged");
  };

  const act = async (fn: () => Promise<unknown>, ok: string) => {
    try {
      await fn();
      toast.show(ok, "info");
      setExpandedPath(null);
      refresh();
    } catch (e) {
      toast.show(e instanceof Error ? e.message : String(e), "warn");
    }
  };

  const renderUnifiedDiff = (file: UnifiedChangeFile) => {
    const f = file.git;
    if (!f) return null;
    const staged = f.x.trim() !== "";
    const work = f.y.trim() !== "";
    return (
      <div className="flex flex-col gap-1.5">
        <div className="flex items-center gap-1">
          {staged && (
            <button
              type="button"
              aria-pressed={diffSide === "staged"}
              className={`cursor-pointer rounded border px-1.5 py-0.5 text-[9.5px] ${diffSide === "staged" ? "border-accent/35 bg-accent/10 text-accent" : "border-border-soft bg-transparent text-fg-faint"}`}
              onClick={() => {
                setDiffSide("staged");
                loadDiff(file.path, true);
              }}
            >
              暂存区 diff
            </button>
          )}
          {work && (
            <button
              type="button"
              aria-pressed={diffSide === "work"}
              className={`cursor-pointer rounded border px-1.5 py-0.5 text-[9.5px] ${diffSide === "work" ? "border-accent/35 bg-accent/10 text-accent" : "border-border-soft bg-transparent text-fg-faint"}`}
              onClick={() => {
                setDiffSide("work");
                loadDiff(file.path, false);
              }}
            >
              工作区 diff
            </button>
          )}
          {onOpenFile && (
            <button
              type="button"
              className="ml-auto inline-flex cursor-pointer items-center gap-1 border-0 bg-transparent p-0 text-[9.5px] text-accent hover:underline"
              onClick={() => onOpenFile(file.path)}
            >
              <FileText size={9} aria-hidden />
              打开文件
            </button>
          )}
        </div>
        {diffLoading ? (
          <div className="flex items-center gap-1 text-[10px] text-fg-faint">
            <Loader2 size={10} className="animate-spin" />
            读取 diff…
          </div>
        ) : diffText !== null ? (
          <ChangesDiff diff={buildGitDiff(diffText)} path={file.path} />
        ) : (
          <div className="text-[10px] text-fg-faint">暂无 diff 内容</div>
        )}
      </div>
    );
  };

  const doCommit = async () => {
    if (committing || !message.trim() || stagedCount === 0) return;
    setCommitting(true);
    try {
      const hash = await app.GaeaGitCommit(message);
      toast.show(`已提交 ${hash || ""}`.trim(), "info");
      setMessage("");
      refresh();
    } catch (e) {
      toast.show(e instanceof Error ? e.message : String(e), "warn");
    } finally {
      setCommitting(false);
    }
  };

  const fileRow = (f: GitFileStatus, side: "staged" | "work" | "untracked" | "conflict") => {
    const isOpen = expandedPath === f.path;
    return (
      <div key={`${side}-${f.path}`}>
        <div className="group flex items-center gap-1.5 px-2.5 py-1.5 text-[11px]">
          <button
            type="button"
            className="flex min-w-0 flex-1 cursor-pointer items-center gap-1.5 border-0 bg-transparent p-0 text-left"
            onClick={() => openDiff(f)}
            title={`查看 ${f.path} 的${f.staged && side === "staged" ? "暂存区" : "工作区"} diff`}
          >
            <span
              className="w-4 shrink-0 rounded text-center font-mono text-[9.5px] font-semibold"
              style={{
                background: "var(--md-sys-color-surface-container-high)",
                color: f.deleted || f.conflict ? "var(--md-sys-color-destructive)" : "var(--md-sys-color-text-secondary)",
              }}
              title={statusLabel(f, side === "staged")}
            >
              {statusLetter(f, side === "staged")}
            </span>
            <span className="truncate text-fg-dim" title={f.path}>
              {f.path.split("/").pop() || f.path}
            </span>
            {f.origPath && (
              <span
                className="truncate font-mono text-[9.5px] text-fg-faint"
                title={`重命名自 ${f.origPath}`}
              >
                ← {f.origPath.split("/").pop() || f.origPath}
              </span>
            )}
            <span
              className="truncate font-mono text-[9.5px] opacity-0 transition-opacity duration-150 group-hover:opacity-100"
              style={{ color: "var(--md-sys-color-text-secondary)" }}
            >
              {f.path}
            </span>
          </button>
          <span className="flex shrink-0 items-center gap-1">
            {(side === "work" || side === "untracked") && (
              <button
                type="button"
                aria-label={`暂存 ${f.path}`}
                title="加入暂存区（git add）"
                className="cursor-pointer border-0 bg-transparent p-0.5 text-fg-faint hover:text-accent"
                onClick={() => void act(() => app.GaeaGitStage([f.path]), `已暂存 ${f.path}`)}
              >
                <Plus size={11} />
              </button>
            )}
            {side === "staged" && (
              <button
                type="button"
                aria-label={`取消暂存 ${f.path}`}
                title="移出暂存区（不动工作区内容）"
                className="cursor-pointer border-0 bg-transparent p-0.5 text-fg-faint hover:text-fg"
                onClick={() => void act(() => app.GaeaGitUnstage([f.path]), `已移出暂存区 ${f.path}`)}
              >
                <Rollback size={11} />
              </button>
            )}
            {side === "work" && (
              <ConfirmButton
                label="丢弃"
                confirmLabel="确认丢弃"
                onConfirm={() => void act(() => app.GaeaGitDiscard(f.path), `已丢弃 ${f.path} 的工作区改动`)}
              />
            )}
          </span>
        </div>
        {isOpen && (
          <div className="ml-6 mr-1.5 mb-1.5 px-2.5 py-2 rounded-[var(--radius-md)] border border-border-soft bg-bg-soft/30">
            {f.untracked ? (
              <div className="text-[10.5px] text-fg-faint">
                未跟踪文件没有 diff 语义。点击「暂存」纳入版本管理后即可查看变更。
              </div>
            ) : f.conflict ? (
              <div className="text-[10.5px] text-fg-faint">
                未合并冲突：在编辑器中解决 <code className="font-mono">&lt;&lt;&lt;&lt;&lt;&lt;&lt;</code> 标记后{" "}
                <code className="font-mono">git add</code> 标记已解决，再提交。本面板不代解冲突。
              </div>
            ) : diffLoading ? (
              <div className="flex items-center gap-1 text-[10px] text-fg-faint">
                <Loader2 size={10} className="animate-spin" />
                读取 diff…
              </div>
            ) : (
              <ChangesDiff diff={buildGitDiff(diffText ?? "")} path={expandedPath ?? undefined} />
            )}
            {onOpenFile && (
              <button
                type="button"
                className="mt-1 inline-flex cursor-pointer items-center gap-1 rounded-md border-0 bg-transparent px-0 text-[10px] text-accent hover:underline"
                onClick={() => onOpenFile(f.path)}
              >
                <FileText size={10} aria-hidden />
                打开预览
              </button>
            )}
          </div>
        )}
      </div>
    );
  };

  const groupHead = (label: string, n: number) => (
    <div
      className="mt-1 flex w-full items-center gap-1.5 px-2.5 py-1 first:mt-0"
      data-testid={`git-group-${label}`}
    >
      <span className="text-[10.5px] font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
        {label}
      </span>
      <span
        className="rounded-full px-1.5 text-[9px] tabular-nums"
        style={{ background: "var(--md-sys-color-surface-container-high)", color: "var(--md-sys-color-text-secondary)" }}
      >
        {n}
      </span>
    </div>
  );

  const worktreeRow = (w: GitWorktreeView) => {
    const name = w.path.split(/[/\\]/).pop() || w.path;
    return (
      <div key={w.path} className="px-2 py-1.5">
        <div className="flex items-center gap-1.5">
          <GitBranch size={11} aria-hidden style={{ color: "var(--md-sys-color-text-secondary)" }} />
          <span className="min-w-0 flex-1 truncate text-[11px] text-fg" title={w.path}>{name}</span>
          {w.isMain && (
            <span className="shrink-0 rounded-full border border-border-soft px-1.5 py-0.5 text-[9px] text-fg-faint">
              主检出
            </span>
          )}
          {w.isCurrent && (
            <span className="shrink-0 rounded-full border border-accent/35 bg-accent/10 px-1.5 py-0.5 text-[9px] text-accent">
              当前
            </span>
          )}
          {w.isLocked && (
            <span className="shrink-0 text-[9px] text-fg-faint" title={w.lockedNote || "已锁定"}>
              锁
            </span>
          )}
        </div>
        <div className="mt-0.5 flex items-center gap-1.5 pl-[18px]">
          <span className="min-w-0 flex-1 truncate font-mono text-[9.5px] text-fg-faint" title={w.path}>{w.path}</span>
          {w.isBare ? (
            <span className="shrink-0 text-[9.5px] text-fg-faint">bare</span>
          ) : w.isDetached ? (
            <span className="shrink-0 font-mono text-[9.5px] text-fg-faint" title={w.head || undefined}>detached</span>
          ) : (
            <span className="shrink-0 font-mono text-[9.5px]" style={{ color: "var(--gaea-glow)" }}>
              {w.branch || "?"}
            </span>
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="flex min-h-0 flex-col py-1">
      <div className="v3-panel-head">
        <GitBranch size={12} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">Git 工作台</span>
        <span className="ml-1 flex rounded-md border border-border-soft p-0.5">
          <button
            type="button"
            aria-pressed={lens === "git"}
            className={`cursor-pointer rounded px-1.5 py-0.5 text-[9.5px] border-0 ${lens === "git" ? "bg-accent text-accent-fg" : "bg-transparent text-fg-faint hover:text-fg"}`}
            onClick={() => setLens("git")}
          >
            Git
          </button>
          <button
            type="button"
            aria-pressed={lens === "unified"}
            className={`cursor-pointer rounded px-1.5 py-0.5 text-[9.5px] border-0 ${lens === "unified" ? "bg-accent text-accent-fg" : "bg-transparent text-fg-faint hover:text-fg"}`}
            onClick={() => setLens("unified")}
          >
            统一
          </button>
          <button
            type="button"
            aria-pressed={lens === "session"}
            className={`cursor-pointer rounded px-1.5 py-0.5 text-[9.5px] border-0 ${lens === "session" ? "bg-accent text-accent-fg" : "bg-transparent text-fg-faint hover:text-fg"}`}
            onClick={() => setLens("session")}
          >
            本轮文件
          </button>
          <button
            type="button"
            aria-pressed={lens === "worktrees"}
            className={`cursor-pointer rounded px-1.5 py-0.5 text-[9.5px] border-0 ${lens === "worktrees" ? "bg-accent text-accent-fg" : "bg-transparent text-fg-faint hover:text-fg"}`}
            onClick={() => setLens("worktrees")}
          >
            工作树
          </button>
          <button
            type="button"
            aria-pressed={lens === "graph"}
            className={`cursor-pointer rounded px-1.5 py-0.5 text-[9.5px] border-0 ${lens === "graph" ? "bg-accent text-accent-fg" : "bg-transparent text-fg-faint hover:text-fg"}`}
            onClick={() => setLens("graph")}
          >
            提交图
          </button>
        </span>
        <span className="v3-panel-spacer" />
        {lens !== "session" && status?.branch && (
          <span className="font-mono text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {status.branch}
            {(status.ahead || status.behind) ? ` ↑${status.ahead || 0} ↓${status.behind || 0}` : ""}
          </span>
        )}
        {lens !== "session" && status?.isLinkedWorktree && (
          <span
            className="rounded-full border border-border-soft px-1.5 py-0.5 text-[9px] text-fg-faint"
            title={`linked worktree，主检出：${status.mainWorktreePath || "未知"}`}
          >
            工作树
          </span>
        )}
        {lens !== "session" && <button
          type="button"
          aria-label="刷新 Git 状态"
          title="刷新 Git 状态"
          className="cursor-pointer border-0 bg-transparent p-0.5 text-fg-faint hover:text-fg"
          onClick={refresh}
        >
          <RefreshCw size={11} className={loading ? "animate-spin" : ""} />
        </button>}
      </div>
      {lens === "session" ? (
        <div className="min-h-0 flex-1 overflow-auto">
          <ChangesPanel changes={changes} cwd={cwd} onOpenFile={onOpenFile ?? (() => {})} />
        </div>
      ) : lens === "unified" ? (
        <div className="min-h-0 flex-1 overflow-auto">
          {gitUnavailable && (
            <div className="mx-2 mb-1 rounded-md border border-border-soft bg-bg-soft/40 px-2 py-1.5 text-[10px] text-fg-faint">
              Git：{status?.error || statusError || "当前工作区不是 Git 仓库"}；仍显示本轮 Agent 文件变更。
            </div>
          )}
          <div className="flex items-center gap-1 px-2 py-1 text-[9.5px] text-fg-faint">
            <span>来源</span>
            {([
              ["all", `全部 ${unifiedAllCount}`],
              ["git", `Git ${status?.files.length ?? 0}`],
              ["agent", `AI ${changes.length}`],
            ] as const).map(([key, label]) => (
              <button
                key={key}
                type="button"
                aria-pressed={unifiedFilter === key}
                className={`cursor-pointer rounded-full border px-1.5 py-0.5 ${unifiedFilter === key ? "border-accent/35 bg-accent/10 text-accent" : "border-border-soft bg-transparent text-fg-faint hover:text-fg"}`}
                onClick={() => setUnifiedFilter(key)}
              >
                {label}
              </button>
            ))}
          </div>
          <UnifiedChangesTree
            nodes={unifiedNodes}
            onOpenFile={onOpenFile}
            onStage={(path) => void act(() => app.GaeaGitStage([path]), `已暂存 ${path}`)}
            onUnstage={(path) => void act(() => app.GaeaGitUnstage([path]), `已移出暂存区 ${path}`)}
            onDiscard={(path) => void act(() => app.GaeaGitDiscard(path), `已丢弃 ${path} 的工作区改动`)}
            expandedDiffPath={expandedPath}
            onToggleDiff={toggleUnifiedDiff}
            renderDiff={renderUnifiedDiff}
          />
        </div>
      ) : lens === "worktrees" ? (
        <div className="min-h-0 flex-1 overflow-auto">
          {gitUnavailable || worktreesError ? (
            <div className="flex flex-col items-center justify-center py-12 px-6 text-center">
              <GitBranch size={18} aria-hidden style={{ color: "var(--md-sys-color-text-secondary)" }} />
              <div className="mt-3 text-[12.5px] font-medium" style={{ color: "var(--md-sys-color-text)" }}>
                工作树不可用
              </div>
              <div className="mt-1 max-w-[260px] text-[11px] leading-snug" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                {worktreesError || status?.error || statusError || "当前工作区不是 Git 仓库"}
              </div>
            </div>
          ) : !worktrees ? (
            <div className="px-2.5 py-2 text-[10px] text-fg-faint">读取中…</div>
          ) : worktrees.length === 0 ? (
            <div className="px-2.5 py-2 text-[10px] text-fg-faint">暂无 worktree</div>
          ) : (
            <div className="flex flex-col px-1 pb-1">
              {worktrees.map((w) => worktreeRow(w))}
            </div>
          )}
        </div>
      ) : lens === "graph" ? (
        // G2 只读提交图：数据自取（GaeaGitGraph/Detail），刷新钮经 panelRevision 驱动。
        <CommitGraphView revision={panelRevision} />
      ) : gitUnavailable ? (
        <div className="flex flex-col items-center justify-center py-12 px-6 text-center">
          <GitBranch size={18} aria-hidden style={{ color: "var(--md-sys-color-text-secondary)" }} />
          <div className="mt-3 text-[12.5px] font-medium" style={{ color: "var(--md-sys-color-text)" }}>
            Git 不可用
          </div>
          <div className="mt-1 max-w-[260px] text-[11px] leading-snug" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {status?.error || statusError || "当前工作区不是 Git 仓库"}
          </div>
        </div>
      ) : (
        <>
      <div className="flex flex-col px-1.5 pb-1">
        {conflicts.length > 0 && (
          <>
            {groupHead("冲突", conflicts.length)}
            <div className="mx-2.5 mb-1 rounded-md border border-border-soft bg-bg-soft/40 px-2 py-1 text-[9.5px] text-fg-faint">
              未合并冲突不可直接提交；在编辑器解决标记后 git add，或回退合并。本面板不代解。
            </div>
            {conflicts.map((f) => fileRow(f, "conflict"))}
          </>
        )}
        {groupHead("已暂存", staged.length)}
        {staged.length === 0 && <div className="px-2.5 pb-1 text-[10px] text-fg-faint">暂存区为空</div>}
        {staged.map((f) => fileRow(f, "staged"))}
        {groupHead("未暂存", unstaged.length)}
        {unstaged.map((f) => fileRow(f, "work"))}
        {groupHead("未跟踪", untracked.length)}
        {untracked.map((f) => fileRow(f, "untracked"))}
      </div>
      {/* 提交区 */}
      <div className="mx-1.5 mt-1 rounded-[var(--radius-md)] border border-border-soft bg-bg-soft/30 p-2">
        <textarea
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder="提交说明（提交的是暂存区内容）"
          aria-label="提交说明"
          rows={2}
          className="w-full resize-none rounded-md border border-border-soft bg-bg px-2 py-1 text-[11px] text-fg placeholder:text-fg-faint focus:border-accent focus:outline-none"
        />
        <div className="mt-1 flex items-center justify-between">
          <span className="text-[9.5px] text-fg-faint">暂存 {stagedCount} 个文件</span>
          <button
            type="button"
            data-testid="git-commit-btn"
            disabled={stagedCount === 0 || !message.trim() || committing}
            className="inline-flex cursor-pointer items-center gap-1 rounded-md border-0 bg-accent px-2.5 py-1 text-[10.5px] font-medium text-bg hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
            onClick={() => void doCommit()}
          >
            {committing ? <Loader2 size={10} className="animate-spin" /> : <Check size={10} aria-hidden />}
            提交
          </button>
        </div>
      </div>
      {/* 历史（折叠，懒加载） */}
      <div className="mx-1.5 mb-1 mt-1.5">
        <button
          type="button"
          aria-expanded={historyOpen}
          data-testid="git-history-toggle"
          className="flex w-full cursor-pointer items-center gap-1.5 rounded-md border-0 bg-transparent px-2.5 py-1 text-left"
          onClick={() => setHistoryOpen((o) => !o)}
        >
          <ChevronDown
            size={10}
            aria-hidden
            className={`transition-transform duration-200 ${historyOpen ? "" : "-rotate-90"}`}
            style={{ color: "var(--md-sys-color-text-secondary)" }}
          />
          <span className="text-[10.5px] font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            提交历史
          </span>
        </button>
        {historyOpen && (
          <div className="mt-0.5 max-h-56 overflow-y-auto pr-0.5">
            {!history && <div className="px-2.5 py-1 text-[10px] text-fg-faint">读取中…</div>}
            {history && history.length === 0 && <div className="px-2.5 py-1 text-[10px] text-fg-faint">暂无提交</div>}
            {history?.map((c) => (
              <div key={c.hash} className="flex items-baseline gap-1.5 px-2.5 py-1 text-[10.5px]">
                <span className="shrink-0 font-mono text-accent">{c.hash}</span>
                <span className="min-w-0 flex-1 truncate text-fg-dim" title={c.subject}>{c.subject}</span>
                <span className="shrink-0 text-fg-faint" title={c.author}>{c.author}</span>
              </div>
            ))}
          </div>
        )}
      </div>
        </>
      )}
    </div>
  );
}
