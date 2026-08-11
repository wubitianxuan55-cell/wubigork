import { useCallback, useEffect, useState } from "react";
import { Modal } from "antd";
import { CloudUpload, Copy, ExternalLink, Pencil, RefreshCw, Trash2 } from "../../icons";
import { app, openExternal } from "../../lib/bridge";
import type { PriceSource } from "../../lib/types";
import { useToast } from "../Toast";
import { PriceSourceFormModal } from "./PriceSourceFormModal";

const FREQ_LABEL: Record<number, string> = {
  0: "仅手动",
  6: "每 6 小时",
  24: "每天",
  168: "每周",
};

function freqLabel(h: number): string {
  return FREQ_LABEL[h] ?? `每 ${h} 小时`;
}

function timeText(s: string): string {
  if (!s) return "从未抓取";
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return s;
  return d.toLocaleString("zh-CN", { hour12: false });
}

// PriceSourcesRepository 价格源阅览仓库：只读陈列系统里所有已添加的价格源
// 及其抓取地址，支持复制地址 / 浏览器打开；管理（增删改/抓取）仍在价格源页。
export function PriceSourcesRepository() {
  const [sources, setSources] = useState<PriceSource[]>([]);
  const [loading, setLoading] = useState(true);
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<PriceSource | null>(null);
  const [deleting, setDeleting] = useState<PriceSource | null>(null);
  const toast = useToast();

  const load = useCallback(() => {
    setLoading(true);
    // 后端调用偶发卡住时 8 秒兜底，避免“加载中”永久转圈。
    Promise.race([
      app.PriceSources(),
      new Promise<PriceSource[]>((res) => setTimeout(() => res([]), 8000)),
    ])
      .then((s) => setSources(s ?? []))
      .catch(() => setSources([]))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const copyUrl = useCallback(
    async (url: string) => {
      try {
        await navigator.clipboard.writeText(url);
        toast.show("已复制抓取地址", "info");
      } catch {
        toast.show("复制失败：剪贴板不可用", "warn");
      }
    },
    [toast],
  );

  const enabledCount = sources.filter((s) => s.enabled).length;

  const removeSource = async () => {
    if (!deleting) return;
    await app.PriceSourceDelete(deleting.id).catch(() => {});
    toast.show(`已删除价格源「${deleting.name}」`, "info");
    setDeleting(null);
    load();
  };

  return (
    <div className="flex flex-col h-full text-fg-dim text-xs">
      <div className="flex items-center justify-between px-3 py-2 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <CloudUpload size={13} className="text-sky-400" />
          价格源阅览仓库
        </span>
        <div className="flex items-center gap-2">
          {!loading && sources.length > 0 && (
            <span className="text-[10px] text-fg-faint">
              共 {sources.length} 个（启用 {enabledCount}）
            </span>
          )}
          <button
            className="flex items-center justify-center w-6 h-6 border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft rounded"
            onClick={load}
            title="刷新仓库"
            disabled={loading}
          >
            <RefreshCw size={12} className={loading ? "animate-spin" : ""} />
          </button>
        </div>
      </div>

      <div className="shrink-0 px-3 pt-2 text-[10.5px] text-fg-faint">
        系统内全部已添加的价格源地址一览（管理/抓取在「价格源」页）
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto p-2">
        {loading ? (
          <div className="py-8 text-center text-fg-faint text-[11px]">加载中…</div>
        ) : sources.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full gap-2 px-6 text-center text-fg-faint/50">
            <CloudUpload size={24} className="opacity-40" />
            <span className="text-[11px] leading-relaxed">
              仓库为空
              <br />
              去「价格源」页添加造价信息网地址后，会在这里集中展示
            </span>
          </div>
        ) : (
          <div className="flex flex-col gap-1.5">
            {sources.map((src) => (
              <div key={src.id} className="p-2 rounded-lg border border-border-soft/70 bg-bg-soft/30">
                <div className="flex items-center gap-1.5">
                  <span className="truncate text-fg text-[12px] font-medium">{src.name}</span>
                  <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px] shrink-0">
                    {freqLabel(src.frequencyHours)}
                  </span>
                  {src.area && (
                    <span className="px-1.5 py-px rounded bg-bg-elev text-fg-faint text-[9.5px] shrink-0">
                      {src.area}
                    </span>
                  )}
                  <span className={`px-1.5 py-px rounded text-[9.5px] shrink-0 ${src.enabled ? "bg-ok/15 text-ok" : "bg-bg-elev text-fg-faint"}`}>
                    {src.enabled ? "启用" : "停用"}
                  </span>
                  <span className="ml-auto shrink-0 text-fg-faint text-[10px]">最近抓取：{timeText(src.lastFetchAt)}</span>
                </div>
                <div className="mt-1 flex items-start gap-1">
                  <span
                    className="min-w-0 flex-1 break-all text-fg-faint text-[9.5px] font-mono leading-snug"
                    title={src.url}
                  >
                    抓取地址：{src.url}
                  </span>
                  <span className="shrink-0 flex items-center gap-0.5">
                    <button
                      type="button"
                      className="flex items-center justify-center w-5 h-5 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft"
                      onClick={() => {
                        setEditing(src);
                        setEditOpen(true);
                      }}
                      title="编辑价格源"
                    >
                      <Pencil size={10} />
                    </button>
                    <button
                      type="button"
                      className="flex items-center justify-center w-5 h-5 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-red-400 hover:bg-bg-soft"
                      onClick={() => setDeleting(src)}
                      title="删除价格源"
                    >
                      <Trash2 size={10} />
                    </button>
                    <button
                      type="button"
                      className="flex items-center justify-center w-5 h-5 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft"
                      onClick={() => void copyUrl(src.url)}
                      title="复制抓取地址"
                    >
                      <Copy size={10} />
                    </button>
                    <button
                      type="button"
                      className="flex items-center justify-center w-5 h-5 rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft"
                      onClick={() => openExternal(src.url)}
                      title="在浏览器打开抓取地址"
                    >
                      <ExternalLink size={10} />
                    </button>
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* 编辑价格源（与价格源页共用弹窗） */}
      <PriceSourceFormModal
        open={editOpen}
        editing={editing}
        onClose={() => setEditOpen(false)}
        onSaved={() => {
          setEditOpen(false);
          load();
        }}
      />

      {/* 删除确认 */}
      <Modal
        title="删除价格源"
        open={!!deleting}
        destroyOnHidden
        transitionName=""
        maskTransitionName=""
        onCancel={() => setDeleting(null)}
        onOk={() => void removeSource()}
        okText="删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <p className="text-[13px] text-fg-dim">确定删除价格源「{deleting?.name}」吗？（抓取记录与价格历史保留）</p>
      </Modal>
    </div>
  );
}
