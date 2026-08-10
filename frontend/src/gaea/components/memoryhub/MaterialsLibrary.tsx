import { useCallback, useEffect, useMemo, useState } from "react";
import { ExternalLink, FileText, Pin, RefreshCw, Eye } from "../../icons";
import { app } from "../../lib/bridge";
import { usePreviewStore } from "../../lib/store";
import type { FileSearchHit } from "../../lib/types";

/**
 * MaterialsLibrary 记忆中枢「项目资料」库：工作区固定常用文件的管理视图。
 * 与办公面板的资料面板共用同一份 .gaea/pinned.json 清单（固定/取消互相可见），
 * 预览走全局 FilePreviewModal；固定后的文件在新会话启动时自动带入上下文。
 */
export function MaterialsLibrary() {
  const [pinned, setPinned] = useState<FileSearchHit[]>([]);
  const [materials, setMaterials] = useState<FileSearchHit[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([app.PinnedMaterials(), app.Materials(120)])
      .then(([p, m]) => {
        setPinned(p ?? []);
        setMaterials(m ?? []);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);
  useEffect(() => { load(); }, [load]);

  const pinnedPaths = useMemo(() => new Set(pinned.map((p) => p.path)), [pinned]);
  const candidates = useMemo(
    () => materials.filter((f) => !pinnedPaths.has(f.path)).slice(0, 30),
    [materials, pinnedPaths],
  );

  const togglePin = useCallback((path: string) => {
    const isPinned = pinnedPaths.has(path);
    (isPinned ? app.UnpinMaterial(path) : app.PinMaterial(path))
      .then((next) => setPinned(next ?? []))
      .catch(() => {});
  }, [pinnedPaths]);

  const openFile = useCallback((path: string) => {
    usePreviewStore.getState().openFilePreview(path);
  }, []);

  return (
    <div className="h-full flex flex-col text-fg-dim text-xs">
      <div className="shrink-0 flex items-center gap-2 px-4 py-2.5 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <Pin size={13} style={{ color: "#38bdf8" }} />
          项目资料
        </span>
        <span className="text-fg-faint text-[10.5px]">固定常用文件 · 新会话自动带入上下文</span>
        <button
          type="button"
          className="ml-auto inline-flex items-center gap-1 px-2 h-6 rounded-md border border-border-soft bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors text-[11px]"
          onClick={load}
          title="刷新资料列表"
        >
          <RefreshCw size={11} className={loading ? "animate-spin" : ""} />
          刷新
        </button>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto px-4 py-3 flex flex-col gap-4">
        {/* 已固定 */}
        <section>
          <div className="text-[10px] uppercase tracking-wider text-fg-faint/70 font-medium mb-1.5 flex items-center gap-1.5">
            <Pin size={10} style={{ color: "#38bdf8" }} />
            已固定 · {pinned.length}
          </div>
          {pinned.length === 0 ? (
            <div className="px-3 py-4 rounded-lg border border-dashed border-border-soft text-fg-faint/60 text-center text-[11px]">
              还没有固定常用文件。在工作区资料里点图钉，或在下方「可固定资料」中挑选。
            </div>
          ) : (
            <div className="flex flex-col gap-1">
              {pinned.map((f) => (
                <div
                  key={f.path}
                  className="group flex items-center gap-2 px-2.5 py-1.5 rounded-md border border-accent/20 bg-accent/5 hover:border-accent/35 transition-colors"
                >
                  <span className="shrink-0 w-6 h-6 rounded-md bg-accent/10 text-accent flex items-center justify-center">
                    <FileText size={12} />
                  </span>
                  <button
                    type="button"
                    onClick={() => openFile(f.path)}
                    className="min-w-0 flex-1 text-left cursor-pointer"
                    title={`点击预览 ${f.path}`}
                  >
                    <span className="block truncate text-[12px] text-fg font-medium leading-tight">{f.name}</span>
                    <span className="block truncate text-[10px] text-fg-faint font-mono leading-tight">{f.path}</span>
                  </button>
                  <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      type="button"
                      className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                      onClick={() => openFile(f.path)}
                      title="预览"
                    >
                      <Eye size={12} />
                    </button>
                    <button
                      type="button"
                      className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-err hover:bg-err/10 transition-colors"
                      onClick={() => togglePin(f.path)}
                      title="取消固定"
                    >
                      <Pin size={11} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>

        {/* 可固定资料 */}
        <section>
          <div className="text-[10px] uppercase tracking-wider text-fg-faint/70 font-medium mb-1.5">
            可固定资料 · {candidates.length}
          </div>
          {candidates.length === 0 ? (
            <div className="px-3 py-4 rounded-lg border border-dashed border-border-soft text-fg-faint/60 text-center text-[11px]">
              工作区暂无更多可固定的资料
            </div>
          ) : (
            <div className="flex flex-col gap-0.5">
              {candidates.map((f) => (
                <div
                  key={f.path}
                  className="group flex items-center gap-2 px-2.5 py-1.5 rounded-md border border-border-soft/60 bg-bg-soft/25 hover:border-accent/30 transition-colors"
                >
                  <span className="shrink-0 w-6 h-6 rounded-md bg-bg-soft text-fg-faint flex items-center justify-center">
                    <FileText size={12} />
                  </span>
                  <button
                    type="button"
                    onClick={() => openFile(f.path)}
                    className="min-w-0 flex-1 text-left cursor-pointer"
                    title={`点击预览 ${f.path}`}
                  >
                    <span className="block truncate text-[12px] text-fg font-medium leading-tight">{f.name}</span>
                    <span className="block truncate text-[10px] text-fg-faint font-mono leading-tight">{f.path}</span>
                  </button>
                  <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      type="button"
                      className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                      onClick={() => openFile(f.path)}
                      title="预览"
                    >
                      <Eye size={12} />
                    </button>
                    <button
                      type="button"
                      className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
                      onClick={() => togglePin(f.path)}
                      title="固定为常用资料（新会话自动带入）"
                    >
                      <Pin size={11} />
                    </button>
                    <button
                      type="button"
                      className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                      onClick={() => void app.OpenWorkspacePath(f.path).catch(() => {})}
                      title="在外部程序中打开"
                    >
                      <ExternalLink size={11} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
