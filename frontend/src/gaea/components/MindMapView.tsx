import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Maximize2 } from "../icons";
import type { MdViewMode } from "../lib/mdViewPref";
import {
  MINDMAP_MAX_NODES,
  MM_NODE_H,
  layoutMindmap,
  parseMindmapOutline,
  type MindLayoutNode,
  type MindNode,
} from "../lib/mindmap";
import {
  mindAddChild,
  mindAddSibling,
  mindMove,
  mindRemove,
  mindRename,
  serializeMindmapOutline,
} from "../lib/mindmapEdit";

const MIN_SCALE = 0.4;
const MAX_SCALE = 2;
const INIT_TX = 16;
const INIT_TY = 16;

// markdown 双视图偏好的键与读写见 lib/mdViewPref.ts；本文件只留组件。

/** 头部「文档/导图」分段切换（FilePreview 与 FilePreviewModal 双入口共用）。 */
export function MdViewToggle({ value, onChange, size = "sm" }: {
  value: MdViewMode;
  onChange: (v: MdViewMode) => void;
  size?: "sm" | "md";
}) {
  const btn = (active: boolean) =>
    `px-1.5 py-0.5 border-0 cursor-pointer transition-colors ${
      size === "sm" ? "text-[10px]" : "text-[11px]"
    } ${active ? "bg-accent/12 text-accent" : "bg-transparent text-fg-dim hover:bg-bg-soft"}`;
  return (
    <span
      className="inline-flex items-center rounded-md border border-border-soft overflow-hidden shrink-0"
      role="group"
      aria-label="markdown 预览视图"
    >
      <button type="button" aria-pressed={value === "doc"} data-testid="md-view-doc" className={btn(value === "doc")} onClick={() => onChange("doc")}>
        文档
      </button>
      <button type="button" aria-pressed={value === "mindmap"} data-testid="md-view-mindmap" className={btn(value === "mindmap")} onClick={() => onChange("mindmap")}>
        导图
      </button>
    </span>
  );
}

// MindMapView — markdown 大纲的交互思维导图视图（M1）。
// 布局/解析全部来自 lib/mindmap 纯函数；组件只持有视图态：
//   - 点击节点折叠/展开（有子节点时）并选中，折叠 badge 显直接子节点数；
//   - 拖拽平移 + 缩放按钮（40%–200%）；「回中」复位视图；
//   - 拖拽超过 4px 阈值的当次点击不触发折叠（平移与点击互不误伤）。
// M2 画布编辑（onSave 传入且文件为纯大纲时启用）：双击改名、Tab 加子节点、
// Enter 加兄弟、Delete 删节点、卡片拖到目标节点=挂载；编辑态树与 Ctrl+S /
// 保存按钮回写规范大纲（serializeMindmapOutline）。混合内容（skipped>0）或
// 截断解析时编辑闸关闭（防回写丢内容），M1 只读行为不变。
export function MindMapView({ text, title, onSave }: { text: string; title: string; onSave?: (text: string) => void }) {
  const parsed = useMemo(() => parseMindmapOutline(text, title), [text, title]);
  const editingEnabled = !!onSave && !parsed.truncated && parsed.skipped === 0;
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(() => new Set<string>());
  const [scale, setScale] = useState(1);
  const [tx, setTx] = useState(INIT_TX);
  const [ty, setTy] = useState(INIT_TY);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [draftRoot, setDraftRoot] = useState<MindNode | null>(null);
  const [dirty, setDirty] = useState(false);
  const panRef = useRef<{ sx: number; sy: number; tx: number; ty: number } | null>(null);
  const dragIdRef = useRef<string | null>(null);
  const renameValueRef = useRef("");
  // pointerup 先于 click；拖拽标志独立保存供当次 click 判定，下次 pointerdown 复位
  const movedRef = useRef(false);

  // 内容换文件：视图态整体复位（折叠/缩放/平移/编辑态）
  useEffect(() => {
    setCollapsed(new Set());
    setScale(1);
    setTx(INIT_TX);
    setTy(INIT_TY);
    setSelectedId(null);
    setRenamingId(null);
    setDraftRoot(null);
    setDirty(false);
  }, [text, title]);

  const tree = draftRoot ?? parsed.root;
  const layout = useMemo(() => layoutMindmap(tree, collapsed), [tree, collapsed]);

  const applyEdit = useCallback((next: MindNode | null) => {
    if (!next) return;
    setDraftRoot(next);
    setDirty(true);
  }, []);

  const commitRename = useCallback(() => {
    if (!renamingId) return;
    const id = renamingId;
    setRenamingId(null);
    const t = renameValueRef.current;
    if (t === (findNodeText(tree, id) ?? "")) return; // 未变更不标脏
    applyEdit(mindRename(tree, id, t));
  }, [renamingId, tree, applyEdit]);

  // 画布编辑快捷键：Tab/Enter/Delete + Ctrl/S_cmd+S（输入框事件在节点内自行处理）
  useEffect(() => {
    if (!editingEnabled) return;
    const onKey = (e: KeyboardEvent) => {
      const tag = (e.target as HTMLElement | null)?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA") return;
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
        if (dirty && onSave) {
          e.preventDefault();
          onSave(serializeMindmapOutline(draftRoot ?? parsed.root));
          setDirty(false);
          setDraftRoot(null);
        }
        return;
      }
      if (renamingId || !selectedId) return;
      if (e.key === "Tab") {
        e.preventDefault();
        applyEdit(mindAddChild(draftRoot ?? parsed.root, selectedId, "新节点"));
      } else if (e.key === "Enter") {
        e.preventDefault();
        applyEdit(mindAddSibling(draftRoot ?? parsed.root, selectedId, "新节点"));
      } else if (e.key === "Delete" || e.key === "Backspace") {
        e.preventDefault();
        if (selectedId !== (draftRoot ?? parsed.root).id) {
          applyEdit(mindRemove(draftRoot ?? parsed.root, selectedId));
          setSelectedId(null);
        }
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [editingEnabled, dirty, renamingId, selectedId, draftRoot, parsed, applyEdit, onSave]);

  const zoomBy = useCallback((factor: number) => {
    setScale((s) => Math.min(MAX_SCALE, Math.max(MIN_SCALE, Math.round(s * factor * 100) / 100)));
  }, []);

  const resetView = useCallback(() => {
    setScale(1);
    setTx(INIT_TX);
    setTy(INIT_TY);
  }, []);

  const onPointerDown = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    if (e.button !== 0) return;
    movedRef.current = false;
    panRef.current = null; // 平移锚点在首个 move 事件时记录（拿当前 tx/ty）
    pendingPanRef.current = { sx: e.clientX, sy: e.clientY };
  }, []);
  const pendingPanRef = useRef<{ sx: number; sy: number } | null>(null);

  const onPointerMove = useCallback(
    (e: React.PointerEvent<HTMLDivElement>) => {
      const start = pendingPanRef.current;
      if (!start) return;
      const dx = e.clientX - start.sx;
      const dy = e.clientY - start.sy;
      if (!panRef.current) {
        if (Math.abs(dx) <= 4 && Math.abs(dy) <= 4) return;
        panRef.current = { sx: start.sx, sy: start.sy, tx, ty };
        movedRef.current = true;
      }
      const p = panRef.current;
      setTx(p.tx + (e.clientX - p.sx));
      setTy(p.ty + (e.clientY - p.sy));
    },
    [tx, ty],
  );

  const onPointerUp = useCallback(() => {
    pendingPanRef.current = null;
    panRef.current = null;
  }, []);

  const toggleNode = useCallback((n: MindLayoutNode) => {
    if (movedRef.current || !n.hasChildren) return;
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(n.id)) next.delete(n.id);
      else next.add(n.id);
      return next;
    });
  }, []);

  const onNodeClick = useCallback(
    (n: MindLayoutNode) => {
      if (editingEnabled) setSelectedId(n.id);
      toggleNode(n);
    },
    [editingEnabled, toggleNode],
  );

  const onNodeDrop = useCallback(
    (targetId: string) => {
      const dragId = dragIdRef.current;
      dragIdRef.current = null;
      if (!editingEnabled || !dragId) return;
      applyEdit(mindMove(draftRoot ?? parsed.root, dragId, targetId));
      setSelectedId(targetId);
    },
    [editingEnabled, draftRoot, parsed, applyEdit],
  );

  return (
    <div className="relative h-full min-h-[420px] select-none" data-testid="mind-map">
      <div
        className="absolute inset-0 overflow-hidden cursor-grab active:cursor-grabbing"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerLeave={onPointerUp}
      >
        <div
          className="absolute left-0 top-0 origin-top-left"
          style={{
            width: layout.width,
            height: layout.height,
            transform: `translate(${tx}px, ${ty}px) scale(${scale})`,
          }}
        >
          <svg
            className="absolute inset-0 pointer-events-none"
            width={layout.width}
            height={layout.height}
            aria-hidden="true"
          >
            {layout.edges.map((d, i) => (
              <path key={i} d={d} fill="none" stroke="var(--border, rgba(128,128,140,0.3))" strokeWidth={1.2} />
            ))}
          </svg>
          {layout.nodes.map((n) => {
            const isRenaming = renamingId === n.id;
            return (
              <div
                key={n.id}
                data-testid="mind-node"
                data-node-id={n.id}
                data-selected={selectedId === n.id ? "true" : undefined}
                draggable={editingEnabled && !isRenaming}
                onDragStart={(e) => {
                  if (!editingEnabled) return;
                  dragIdRef.current = n.id;
                  e.dataTransfer.effectAllowed = "move";
                }}
                onDragOver={(e) => {
                  if (!editingEnabled || !dragIdRef.current || dragIdRef.current === n.id) return;
                  e.preventDefault();
                }}
                onDrop={(e) => {
                  e.preventDefault();
                  onNodeDrop(n.id);
                }}
                onClick={() => (isRenaming ? undefined : onNodeClick(n))}
                onDoubleClick={() => {
                  if (!editingEnabled || isRenaming) return;
                  renameValueRef.current = n.text;
                  setRenamingId(n.id);
                }}
                className={`absolute flex items-center gap-1 rounded-lg border px-2 text-left text-[12px] leading-none truncate ${
                  isRenaming ? "cursor-text" : "cursor-pointer transition-colors"
                } ${
                  n.depth === 0
                    ? "border-accent/40 bg-accent/10 text-accent font-medium"
                    : "border-border-soft bg-bg-elev text-fg hover:border-accent/40"
                } ${selectedId === n.id ? "outline outline-2 outline-accent" : ""}`}
                style={{ left: n.x, top: n.y, width: n.w, height: MM_NODE_H }}
                title={n.text}
              >
                {isRenaming ? (
                  <input
                    autoFocus
                    data-testid="mind-rename-input"
                    defaultValue={n.text}
                    className="w-full bg-transparent outline-none text-[12px] text-fg"
                    onChange={(e) => {
                      renameValueRef.current = e.target.value;
                    }}
                    onClick={(e) => e.stopPropagation()}
                    onDoubleClick={(e) => e.stopPropagation()}
                    onKeyDown={(e) => {
                      e.stopPropagation();
                      if (e.key === "Enter") commitRename();
                      else if (e.key === "Escape") setRenamingId(null);
                    }}
                    onBlur={commitRename}
                  />
                ) : (
                  <>
                    <span className="truncate flex-1">{n.text}</span>
                    {n.collapsed && n.hiddenChildren > 0 && (
                      <span
                        data-testid="mind-collapsed-badge"
                        className="shrink-0 rounded-full bg-accent/15 text-accent text-[9px] px-1 py-0.5 leading-none"
                      >
                        +{n.hiddenChildren}
                      </span>
                    )}
                    {n.hasChildren && !n.collapsed && (
                      <span aria-hidden="true" className="shrink-0 text-fg-faint/60 text-[9px]">
                        ▾
                      </span>
                    )}
                  </>
                )}
              </div>
            );
          })}
        </div>
      </div>
      <div className="absolute top-2 right-2 flex items-center gap-0.5 rounded-lg border border-border-soft bg-bg-elev/90 px-1 py-0.5 text-[11px] text-fg-dim">
        <button
          type="button"
          className="w-6 h-6 rounded-md border-0 bg-transparent cursor-pointer hover:bg-bg-soft text-fg-dim"
          onClick={() => zoomBy(1 / 1.2)}
          aria-label="缩小"
          data-testid="mind-zoom-out"
        >
          −
        </button>
        <span data-testid="mind-zoom" className="w-9 text-center tabular-nums">
          {Math.round(scale * 100)}%
        </span>
        <button
          type="button"
          className="w-6 h-6 rounded-md border-0 bg-transparent cursor-pointer hover:bg-bg-soft text-fg-dim"
          onClick={() => zoomBy(1.2)}
          aria-label="放大"
          data-testid="mind-zoom-in"
        >
          ＋
        </button>
        <button
          type="button"
          className="w-6 h-6 rounded-md border-0 bg-transparent cursor-pointer hover:bg-bg-soft text-fg-dim flex items-center justify-center"
          onClick={resetView}
          title="回中"
          aria-label="回中"
          data-testid="mind-reset"
        >
          <Maximize2 size={10} />
        </button>
      </div>
      {parsed.count <= 1 && (
        <div className="absolute bottom-2 left-1/2 -translate-x-1/2 px-3 py-1 rounded-md border border-amber-500/30 bg-amber-500/5 text-amber-500 text-[11px] whitespace-nowrap">
          未发现大纲结构（# 标题或 - 列表）——添加标题/列表后自动成图
        </div>
      )}
      {parsed.truncated && (
        <div className="absolute bottom-2 right-2 px-3 py-1 rounded-md border border-amber-500/30 bg-amber-500/5 text-amber-500 text-[11px]">
          导图过大，仅渲染前 {MINDMAP_MAX_NODES} 个节点
        </div>
      )}
      {editingEnabled && (
        <div className="absolute bottom-2 left-2 rounded-md border border-border-soft bg-bg-elev/90 px-2 py-1 text-[10px] text-fg-faint whitespace-nowrap">
          双击改名 · Tab 加子节点 · Enter 加同级 · Delete 删除 · 拖拽挂载
        </div>
      )}
      {onSave && !editingEnabled && !parsed.truncated && parsed.skipped > 0 && (
        <div className="absolute bottom-2 left-2 px-3 py-1 rounded-md border border-amber-500/30 bg-amber-500/5 text-amber-500 text-[11px] whitespace-nowrap">
          文件含非大纲内容（段落/代码块），导图编辑暂不可用（防保存丢失内容）
        </div>
      )}
      {dirty && (
        <div className="absolute bottom-2 left-1/2 -translate-x-1/2 flex items-center gap-2 rounded-lg border border-accent/40 bg-bg-elev/95 px-3 py-1 text-[11px] shadow-lg">
          <span className="text-fg-dim">有未保存修改</span>
          <button
            type="button"
            data-testid="mind-save"
            className="px-2 py-0.5 rounded-md border border-accent/40 bg-accent/10 text-accent cursor-pointer"
            onClick={() => {
              if (!onSave) return;
              onSave(serializeMindmapOutline(draftRoot ?? parsed.root));
              setDirty(false);
              setDraftRoot(null);
            }}
          >
            保存（Ctrl+S）
          </button>
          <button
            type="button"
            className="px-2 py-0.5 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg"
            onClick={() => {
              setDraftRoot(null);
              setDirty(false);
              setSelectedId(null);
            }}
          >
            放弃
          </button>
        </div>
      )}
    </div>
  );
}

function findNodeText(root: MindNode, id: string): string | null {
  if (root.id === id) return root.text;
  for (const c of root.children) {
    const hit = findNodeText(c, id);
    if (hit !== null) return hit;
  }
  return null;
}
