import { memo, useCallback, useEffect, useRef, useState } from "react";
import { AlertCircle, Image as ImageIcon, Loader2 } from "../../icons";
import { app } from "../../lib/bridge";
import { useT } from "../../lib/i18n";
import type { PreviewResult } from "../../lib/types";
import {
  buildThumbSections,
  classifyDirProbe,
  mapPool,
  orderGroupDirs,
  splitArtifactEntries,
  verifyArtifactRelPath,
} from "../../lib/verifyArtifacts";
import type {
  VerifyArtifactsFault,
  VerifyDirEntry,
  VerifyPageFile,
  VerifyThumbSection,
} from "../../lib/verifyArtifacts";

// 单页缩略图状态：loading（占位骨架）→ ok（dataUrl）/ error（诚实失败占位，
// 绝不伪造页面图）。逐页 Preview 失败互不拖垮整批。
type ThumbState = { status: "loading" } | { status: "ok"; dataUrl: string } | { status: "error" };

type ThumbsLoad =
  | { phase: "idle" }
  | { phase: "listing" }
  | { phase: "fault"; fault: VerifyArtifactsFault }
  | { phase: "ready"; sections: VerifyThumbSection[]; thumbs: Record<string, ThumbState> };

// 与 DeliverablesPanel 内联操作按钮同款令牌化样式（iconBtn 私有于面板，此处
// 保持同形避免反向依赖）。
const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// VerifyArtifactsThumbs — Verifier 通道 B「老账」：视觉复核行内的逐页缩略图
// 预览（不跳出应用）。真实产物布局（gaea_verify.go + docmd/render.go）：
// 产物目录下 before/ 与 after/ 子目录各存 pdftoppm 逐页 PNG；before/after
// 折成一个成对区（逐页「改前 | 改后」并排，页数变化一侧以「无此页」诚实
// 占位），其余分组各自成区。
// 懒加载：点「查看缩略图」才列目录/取图（ListDir + Preview image dataUrl，
// 并发≤4）；展开过的数据保留，收起再展开不重拉。诚实降级：目录列不出 /
// 为空 / 无页面图 / 路径不可达（绝对路径无法相对化）时给出具体原因。
export const VerifyArtifactsThumbs = memo(function VerifyArtifactsThumbs({
  artifacts,
}: {
  /** verdict.channelBArtifacts：产物目录（绝对路径或相对路径）。 */
  artifacts: string;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [load, setLoad] = useState<ThumbsLoad>({ phase: "idle" });
  const startedRef = useRef(false);
  const liveRef = useRef(true);
  // liveRef 必须在 effect 本体重置为 true、cleanup 置 false——只写 cleanup
  // 的旧写法在 React 18 开发模式 StrictMode 的 effect double-invoke 下
  // （mount→cleanup→mount，第二个 effect 是同一函数、本体无事可做）会让
  // liveRef 恒 false：组件 dev 下永远「已卸载」，所有 setLoad 被门卫吞掉
  // （?mock=1 走查实锤 phase 恒 idle；生产构建与 jsdom 单测均不触发）。
  useEffect(() => {
    liveRef.current = true;
    return () => {
      liveRef.current = false;
    };
  }, []);

  const loadThumbs = useCallback(async () => {
    const rel = verifyArtifactRelPath(artifacts);
    if (!rel) {
      // 路径不可达（不在当前工作区且 ListDir 不收绝对路径）：只能外部打开
      if (liveRef.current) setLoad({ phase: "fault", fault: "unreachable" });
      return;
    }
    if (liveRef.current) setLoad({ phase: "listing" });
    let rootEntries: VerifyDirEntry[] = [];
    try {
      rootEntries = (await app.ListDir(rel)) ?? [];
    } catch {
      rootEntries = [];
    }
    const root = splitArtifactEntries(rootEntries, rel);
    let groups: Array<{ key: string; pages: VerifyPageFile[] }> = [];
    if (root.dirs.length > 0) {
      // 真实布局：before/ 与 after/ 子目录各一层页面图（子目录列举同样≤4 并发）
      groups = await mapPool(orderGroupDirs(root.dirs), 4, async (dir) => {
        const subRel = `${rel}/${dir}`;
        try {
          const sub = splitArtifactEntries((await app.ListDir(subRel)) ?? [], subRel);
          return { key: dir, pages: sub.pages };
        } catch {
          return { key: dir, pages: [] as VerifyPageFile[] };
        }
      });
    } else if (root.pages.length > 0) {
      // 防御容错：页面图平铺在产物根（无子目录）也能呈现
      groups = [{ key: "flat", pages: root.pages }];
    }
    if (groups.every((g) => g.pages.length === 0)) {
      // 列不出/列为空/无可识别页面 → Preview 探测定性（不存在/为空/无页面图）
      let probe: PreviewResult | undefined;
      try {
        probe = await app.Preview(rel);
      } catch {
        probe = undefined;
      }
      const fault = probe ? classifyDirProbe(probe.kind, probe.error) : "empty";
      if (liveRef.current) setLoad({ phase: "fault", fault });
      return;
    }
    const sections = buildThumbSections(groups);
    const thumbs: Record<string, ThumbState> = {};
    for (const s of sections) {
      for (const r of s.rows) {
        for (const c of r.cells) thumbs[c.relPath] = { status: "loading" };
      }
    }
    if (liveRef.current) setLoad({ phase: "ready", sections, thumbs: { ...thumbs } });
    // 逐页取 dataUrl：并发≤4，单页失败落 error 占位，不拖垮整批
    await mapPool(Object.keys(thumbs), 4, async (p) => {
      let st: ThumbState = { status: "error" };
      try {
        const r = await app.Preview(p);
        st = r.kind === "image" && r.dataUrl ? { status: "ok", dataUrl: r.dataUrl } : { status: "error" };
      } catch {
        st = { status: "error" };
      }
      thumbs[p] = st;
      if (liveRef.current) setLoad({ phase: "ready", sections, thumbs: { ...thumbs } });
    });
  }, [artifacts]);

  // 副作用（loadThumbs）必须在事件处理器本体执行，禁止写进 setOpen 的
  // updater——updater 必须纯，真实应用里父级（DeliverablesPanel 2s 轮询）
  // 更新竞争时 updater 内副作用会被 React 吞掉（?mock=1 走查实锤：open
  // 翻转成功但 phase 恒 idle；jsdom 单测无轮询竞争测不出）。
  const onToggle = useCallback(() => {
    const next = !open;
    setOpen(next);
    if (next && !startedRef.current) {
      startedRef.current = true;
      void loadThumbs();
    }
  }, [open, loadThumbs]);

  if (!artifacts) return null;

  const faultText = (fault: VerifyArtifactsFault): string => {
    switch (fault) {
      case "unreachable":
        return t("deliverPanel.thumbsUnreachable");
      case "missing":
        return t("deliverPanel.thumbsMissing");
      case "empty":
        return t("deliverPanel.thumbsEmpty");
      default:
        return t("deliverPanel.thumbsNoPages");
    }
  };

  const sectionTitle = (key: string): string => {
    if (key === "flat") return t("deliverPanel.thumbsGroupFlat");
    return key;
  };

  return (
    <>
      <button
        type="button"
        className={iconBtn}
        onClick={onToggle}
        aria-expanded={open}
        aria-label={t("deliverPanel.thumbsToggleTitle")}
        title={t("deliverPanel.thumbsToggleTitle")}
        data-testid="verify-thumbs-toggle"
      >
        <ImageIcon size={11} />
      </button>
      {open && (
        <div
          className="w-full flex flex-col gap-1.5 pl-1"
          data-testid="verify-thumbs-grid"
        >
          {load.phase === "listing" && (
            <div className="flex items-center gap-1 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              <Loader2 size={11} className="animate-spin" />
              {t("deliverPanel.thumbsLoading")}
            </div>
          )}
          {load.phase === "fault" && (
            <div
              className="flex items-center gap-1 text-[9px]"
              data-testid="verify-thumbs-fault"
              style={{ color: "var(--md-sys-color-warning)" }}
            >
              <AlertCircle size={11} className="shrink-0" />
              {faultText(load.fault)}
            </div>
          )}
          {load.phase === "ready" &&
            load.sections.map((section) => (
              <div key={section.key} className="flex flex-col gap-1">
                {section.key !== "pair" && (
                  <span className="text-[9px] font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                    {sectionTitle(section.key)}
                  </span>
                )}
                {section.rows.map((row) => (
                  <div key={row.key} className="flex items-center gap-1.5">
                    <span
                      className="shrink-0 text-[9px] font-mono"
                      style={{ color: "var(--md-sys-color-text-secondary)" }}
                    >
                      {t("deliverPanel.thumbsPageLabel", { page: row.page })}
                    </span>
                    {row.cells.map((cell) => (
                      <figure key={cell.key} className="flex flex-col gap-0.5 m-0" data-testid="verify-thumb-cell">
                        {cell.side !== "single" && (
                          <figcaption
                            className="text-[8px]"
                            style={{ color: "var(--md-sys-color-text-secondary)" }}
                          >
                            {cell.side === "before"
                              ? t("deliverPanel.thumbsGroupBefore")
                              : t("deliverPanel.thumbsGroupAfter")}
                          </figcaption>
                        )}
                        <div
                          className="w-24 h-32 rounded border overflow-hidden flex items-center justify-center"
                          style={{
                            borderColor: "var(--md-sys-color-outline-variant)",
                            background: "var(--md-sys-color-surface-container)",
                          }}
                        >
                          {load.thumbs[cell.relPath]?.status === "ok" ? (
                            <img
                              src={(load.thumbs[cell.relPath] as { dataUrl: string }).dataUrl}
                              alt={
                                cell.side === "single"
                                  ? t("deliverPanel.thumbsPageLabel", { page: cell.page })
                                  : `${t("deliverPanel.thumbsPageLabel", { page: cell.page })}（${cell.side === "before" ? t("deliverPanel.thumbsGroupBefore") : t("deliverPanel.thumbsGroupAfter")}）`
                              }
                              className="w-full h-full object-contain"
                              loading="lazy"
                            />
                          ) : load.thumbs[cell.relPath]?.status === "error" ? (
                            <span
                              className="flex items-center gap-0.5 text-[8px] px-1"
                              data-testid="verify-thumb-fail"
                              style={{ color: "var(--md-sys-color-warning)" }}
                            >
                              <AlertCircle size={9} className="shrink-0" />
                              {t("deliverPanel.thumbsPageFail")}
                            </span>
                          ) : (
                            <span
                              className="block w-full h-full animate-pulse"
                              title={t("deliverPanel.thumbsLoading")}
                              style={{ background: "var(--md-sys-color-surface-container-high)" }}
                            />
                          )}
                        </div>
                      </figure>
                    ))}
                    {section.key === "pair" && row.cells.length < 2 && (
                      <div
                        className="w-24 h-32 rounded border border-dashed flex items-center justify-center text-[8px]"
                        data-testid="verify-thumb-missing"
                        style={{
                          borderColor: "var(--md-sys-color-outline-variant)",
                          color: "var(--md-sys-color-text-secondary)",
                        }}
                      >
                        {t("deliverPanel.thumbsMissingSide")}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ))}
        </div>
      )}
    </>
  );
});
