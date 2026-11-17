import { Image as ImageIcon, RefreshCw } from "../icons";
import { app } from "../lib/bridge";
import { subscribe } from "../../events";
import type { WenxueImageItem } from "../lib/types/files";
import { useEffect, useMemo, useRef, useState } from "react";
import { WenxueHubEmpty, WenxueHubFrame } from "./WenxueHubFrame";
import type { WenxueHubAction } from "./WenxueHubFrame";

// WenxueGalleryHub — 文学板块「插图」标签页（2026-11-18）。
// 数据面=工作空间图片清单（WenxueGalleryList）：uploads（image_gen 封面/场景
// 插图/角色立绘 + 粘贴图）与工作空间其他图片；取图经 Preview（image→dataUrl）。
// agent/预览落盘后通过 gaea-wenxue-filetree 通道自动刷新（设定/角色同款纪律）。
// 大图查看=卡内点开 lightbox 遮罩（v1 不引入独立查看器）。

interface Loaded {
  rel: string;
  dataUrl: string;
}

function bytesLabel(n: number): string {
  if (n >= 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  if (n >= 1024) return `${(n / 1024).toFixed(0)} KB`;
  return `${n} B`;
}

export function WenxueGalleryHub() {
  const [items, setItems] = useState<WenxueImageItem[] | null>(null);
  const [loadErr, setLoadErr] = useState("");
  const [filter, setFilter] = useState<"all" | "uploads" | "workspace">("all");
  const [loaded, setLoaded] = useState<Map<string, Loaded>>(new Map());
  const [lightbox, setLightbox] = useState<WenxueImageItem | null>(null);
  const inflight = useRef(false);

  // Esc 关灯箱（提示词承诺的关闭路径）。
  useEffect(() => {
    if (lightbox === null) return undefined;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setLightbox(null);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [lightbox]);

  const load = async () => {
    if (inflight.current) return;
    inflight.current = true;
    try {
      const list = await app.GalleryList();
      setItems(list);
      setLoadErr("");
    } catch (e) {
      setLoadErr(e instanceof Error ? e.message : String(e));
    } finally {
      inflight.current = false;
    }
  };

  useEffect(() => {
    void load();
    // agent/预览落盘 → filetree 通道自动刷新（设定/角色 Hub 同款纪律）。
    return subscribe("gaea-wenxue-filetree", () => {
      void load();
    });
  }, []);

  // 可视卡片懒加载取图：WenxuePreview(image→dataUrl)。加载失败的项以占位呈现。
  const ensureImage = async (rel: string) => {
    if (loaded.has(rel) || inflight.current) return;
    try {
      const preview = await app.Preview(rel);
      if (preview.dataUrl) {
        setLoaded((prev) => {
          const next = new Map(prev);
          next.set(rel, { rel, dataUrl: preview.dataUrl });
          return next;
        });
      }
    } catch {
      // 取图失败：卡片显示占位，不炸整墙
    }
  };

  const visible = useMemo(() => {
    const list = items ?? [];
    return filter === "all" ? list : list.filter((it) => it.source === filter);
  }, [items, filter]);

  useEffect(() => {
    // 首屏加载可视区前 24 张（图片解码成本），滚动按需再补由后续刀迭代。
    visible.slice(0, 24).forEach((it) => void ensureImage(it.rel));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible]);

  const genCount = (items ?? []).filter((it) => it.source === "uploads").length;
  const actions: WenxueHubAction[] = [
    { key: "refresh", label: "刷新", icon: RefreshCw, onClick: () => void load() },
  ];

  return (
    <WenxueHubFrame
      tone="gallery"
      eyebrow="ILLUSTRATIONS · 图册"
      title="插图"
      description="这本书里由 AI 生成与粘贴收集的图片：封面、场景插图、角色立绘。生成产出落在书目录 .gaea/uploads。"
      metrics={[
        { label: "图片", value: items ? String(items.length) : "—" },
        { label: "生成", value: items ? String(genCount) : "—" },
      ]}
      actions={actions}
    >
      {loadErr !== "" ? (
        <div className="wx-gallery-error" role="alert">{loadErr}</div>
      ) : items === null ? (
        <div className="wx-gallery-empty" data-testid="gallery-loading">图册加载中…</div>
      ) : visible.length === 0 ? (
        <WenxueHubEmpty
          icon={ImageIcon}
          title="还没有插图"
          description="让 AI 用 image_gen 生成封面/场景插图/角色立绘（产出落书目录 .gaea/uploads），或在对话里粘贴图片，图册会自动收录。"
        />
      ) : (
        <>
          <div className="wx-gallery-filters" role="group" aria-label="图片来源筛选">
            {([["all", "全部"], ["uploads", "生成"], ["workspace", "其他"]] as const).map(([key, label]) => (
              <button
                key={key}
                type="button"
                className={`wx-gallery-filter${filter === key ? " is-active" : ""}`}
                aria-pressed={filter === key}
                onClick={() => setFilter(key)}
              >
                {label}
              </button>
            ))}
          </div>
          <div className="wx-gallery-grid" data-testid="gallery-grid">
            {visible.map((it) => {
              const img = loaded.get(it.rel);
              return (
                <figure key={it.rel} className="wx-gallery-item">
                  <button
                    type="button"
                    className="wx-gallery-thumb"
                    aria-label={`查看大图：${it.name}`}
                    onClick={() => setLightbox(it)}
                  >
                    {img ? (
                      <img src={img.dataUrl} alt={it.name} loading="lazy" />
                    ) : (
                      <span className="wx-gallery-thumb-placeholder" aria-hidden><ImageIcon size={18} /></span>
                    )}
                  </button>
                  <figcaption className="wx-gallery-meta">
                    <span className="wx-gallery-name" title={it.rel}>{it.name}</span>
                    <span className="wx-gallery-sub">
                      <span className={`wx-gallery-source is-${it.source}`}>
                        {it.source === "uploads" ? "生成" : "其他"}
                      </span>
                      <span>{bytesLabel(it.size)}</span>
                    </span>
                  </figcaption>
                </figure>
              );
            })}
          </div>
        </>
      )}
      {lightbox !== null && (
        <div
          className="wx-gallery-lightbox"
          role="dialog"
          aria-label={`大图查看：${lightbox.name}`}
          onClick={() => setLightbox(null)}
        >
          <img
            src={loaded.get(lightbox.rel)?.dataUrl ?? ""}
            alt={lightbox.name}
            onClick={(e) => e.stopPropagation()}
          />
          <div className="wx-gallery-lightbox-meta">{lightbox.name}（Esc/点击遮罩关闭）</div>
          <button type="button" className="wx-gallery-lightbox-close" aria-label="关闭大图" onClick={() => setLightbox(null)}>×</button>
        </div>
      )}
    </WenxueHubFrame>
  );
}
