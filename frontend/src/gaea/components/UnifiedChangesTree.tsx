import { useEffect, useState } from "react";
import { ChevronDown, ChevronRight, Diff, Plus, Rollback } from "../icons";
import { FileTypeIcon } from "../lib/fileIcon";
import type { UnifiedChangeDir, UnifiedChangeFile, UnifiedChangeNode } from "../lib/unifiedChangesTree";

function collectDirs(nodes: UnifiedChangeNode[]): string[] {
  const out: string[] = [];
  for (const node of nodes) {
    if (node.kind !== "dir") continue;
    out.push(node.path, ...collectDirs(node.children));
  }
  return out;
}

function SourceBadges({ file }: { file: UnifiedChangeFile }) {
  const x = file.git?.x.trim() ?? "";
  const y = file.git?.y.trim() ?? "";
  return (
    <span className="ml-auto flex shrink-0 items-center gap-1">
      {x && (
        <span className="rounded bg-accent/10 px-1 font-mono text-[8.5px] text-accent" title={`暂存区：${x}`}>
          暂 {x}
        </span>
      )}
      {y && (
        <span className="rounded bg-amber-500/10 px-1 font-mono text-[8.5px] text-amber-500" title={`工作区：${y}`}>
          工 {y}
        </span>
      )}
      {(file.agent?.count ?? 0) > 0 && (
        <span className="rounded bg-sky-500/10 px-1 text-[8.5px] text-sky-400" title="本轮 Agent 改动次数">
          AI {file.agent?.count}
        </span>
      )}
    </span>
  );
}

export function UnifiedChangesTree({
  nodes,
  onOpenFile,
  onStage,
  onUnstage,
  onDiscard,
  expandedDiffPath,
  onToggleDiff,
  renderDiff,
}: {
  nodes: UnifiedChangeNode[];
  onOpenFile?: (path: string) => void;
  onStage?: (path: string) => void;
  onUnstage?: (path: string) => void;
  onDiscard?: (path: string) => void;
  expandedDiffPath?: string | null;
  onToggleDiff?: (file: UnifiedChangeFile) => void;
  renderDiff?: (file: UnifiedChangeFile) => React.ReactNode;
}) {
  const [expanded, setExpanded] = useState<Set<string>>(() => new Set(collectDirs(nodes)));

  const toggle = (path: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const render = (list: UnifiedChangeNode[], depth: number): React.ReactNode => list.map((node) => {
    if (node.kind === "dir") {
      const open = expanded.has(node.path);
      return <DirRow key={`d:${node.path}`} node={node} depth={depth} open={open} onToggle={() => toggle(node.path)} render={render} />;
    }
    return (
      <FileRow
        key={`f:${node.path}`}
        file={node}
        depth={depth}
        onOpenFile={onOpenFile}
        onStage={onStage}
        onUnstage={onUnstage}
        onDiscard={onDiscard}
        expandedDiffPath={expandedDiffPath}
        onToggleDiff={onToggleDiff}
        renderDiff={renderDiff}
      />
    );
  });

  if (nodes.length === 0) {
    return <div className="px-3 py-8 text-center text-[11px] text-fg-faint">当前工作区没有文件变更</div>;
  }
  return <div className="flex flex-col py-1" data-testid="unified-changes-tree">{render(nodes, 0)}</div>;
}

function DirRow({
  node,
  depth,
  open,
  onToggle,
  render,
}: {
  node: UnifiedChangeDir;
  depth: number;
  open: boolean;
  onToggle: () => void;
  render: (list: UnifiedChangeNode[], depth: number) => React.ReactNode;
}) {
  return (
    <div>
      <button
        type="button"
        className="group flex w-full cursor-pointer items-center gap-1 border-0 bg-transparent py-1 text-left text-[11px] text-fg-dim hover:bg-bg-soft"
        style={{ paddingLeft: `${8 + depth * 13}px` }}
        onClick={onToggle}
      >
        {open ? <ChevronDown size={9} className="shrink-0 text-fg-faint" /> : <ChevronRight size={9} className="shrink-0 text-fg-faint" />}
        <FileTypeIcon name={node.name} isDir open={open} size={13} />
        <span className="truncate">{node.name}</span>
        <span className="ml-auto shrink-0 text-[9px] text-fg-faint">
          {node.fileCount} 文件{node.gitCount > 0 ? ` · Git ${node.gitCount}` : ""}{node.agentCount > 0 ? ` · AI ${node.agentCount}` : ""}
        </span>
      </button>
      {open && render(node.children, depth + 1)}
    </div>
  );
}

function FileRow({
  file,
  depth,
  onOpenFile,
  onStage,
  onUnstage,
  onDiscard,
  expandedDiffPath,
  onToggleDiff,
  renderDiff,
}: {
  file: UnifiedChangeFile;
  depth: number;
  onOpenFile?: (path: string) => void;
  onStage?: (path: string) => void;
  onUnstage?: (path: string) => void;
  onDiscard?: (path: string) => void;
  expandedDiffPath?: string | null;
  onToggleDiff?: (file: UnifiedChangeFile) => void;
  renderDiff?: (file: UnifiedChangeFile) => React.ReactNode;
}) {
  const [discardArmed, setDiscardArmed] = useState(false);
  const x = file.git?.x.trim() ?? "";
  const y = file.git?.y.trim() ?? "";
  const canStage = !!file.git && (!!y || file.git.untracked === true);
  const canUnstage = !!file.git && !!x;
  const canDiscard = !!file.git && !!y && file.git.untracked !== true;
  const canDiff = !!file.git && file.git.untracked !== true;
  const diffOpen = expandedDiffPath === file.path;
  useEffect(() => {
    if (!discardArmed) return;
    const timer = window.setTimeout(() => setDiscardArmed(false), 3000);
    return () => window.clearTimeout(timer);
  }, [discardArmed]);
  return (
    <div data-unified-file={file.path}>
      <div
        className="group flex w-full items-center gap-1 py-1 text-[11px] text-fg-dim hover:bg-bg-soft"
        style={{ paddingLeft: `${21 + depth * 13}px` }}
        title={file.path}
      >
        <button
          type="button"
          className="flex min-w-0 flex-1 cursor-pointer items-center gap-1 border-0 bg-transparent p-0 text-left text-[11px] text-fg-dim"
          onClick={() => onOpenFile?.(file.path)}
        >
          <FileTypeIcon name={file.name} size={13} />
          <span className="truncate">{file.name}</span>
          <SourceBadges file={file} />
        </button>
        <span className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100">
        {canDiff && onToggleDiff && (
          <button
            type="button"
            aria-label={`${diffOpen ? "收起" : "查看"} ${file.path} 的 diff`}
            title={diffOpen ? "收起 diff" : "查看 diff"}
            className={`cursor-pointer border-0 bg-transparent p-0.5 ${diffOpen ? "text-accent" : "text-fg-faint hover:text-accent"}`}
            onClick={() => onToggleDiff(file)}
          >
            <Diff size={10} />
          </button>
        )}
        {canUnstage && onUnstage && (
          <button
            type="button"
            aria-label={`取消暂存 ${file.path}`}
            title="移出暂存区"
            className="cursor-pointer border-0 bg-transparent p-0.5 text-fg-faint hover:text-fg"
            onClick={() => onUnstage(file.path)}
          >
            <Rollback size={10} />
          </button>
        )}
        {canStage && onStage && (
          <button
            type="button"
            aria-label={`暂存 ${file.path}`}
            title="加入暂存区"
            className="cursor-pointer border-0 bg-transparent p-0.5 text-fg-faint hover:text-accent"
            onClick={() => onStage(file.path)}
          >
            <Plus size={10} />
          </button>
        )}
        {canDiscard && onDiscard && (
          <button
            type="button"
            aria-pressed={discardArmed}
            aria-label={`${discardArmed ? "确认丢弃" : "丢弃"} ${file.path}`}
            title={discardArmed ? "再次点击确认丢弃工作区改动" : "丢弃工作区改动"}
            className={`cursor-pointer rounded border-0 bg-transparent px-1 py-0.5 text-[8.5px] ${discardArmed ? "text-err" : "text-fg-faint hover:text-err"}`}
            onClick={() => {
              if (discardArmed) {
                setDiscardArmed(false);
                onDiscard(file.path);
              } else {
                setDiscardArmed(true);
              }
            }}
          >
            {discardArmed ? "确认" : "丢"}
          </button>
        )}
        </span>
      </div>
      {diffOpen && renderDiff && (
        <div className="mr-1.5 mb-1.5 ml-5 rounded-md border border-border-soft bg-bg-soft/25 px-2.5 py-2">
          {renderDiff(file)}
        </div>
      )}
    </div>
  );
}
