import { useId, useMemo } from "react";
import { AlertCircle, CheckCircle, FileImage, FilePdf, FileText, Scan } from "../icons";
import { useI18n } from "../lib/i18n";
import type { DictKey } from "../locales/en";
import type { Item } from "../lib/store";

// OCRCard — 办公文件识别作业卡（2026-11-18 重设计）。
//
// Why：OCR 在对话流里此前全程静默——agent `ocr` 工具只有 dispatch→result 两
// 点、@附件 OCR 腿完全无声、扫描件 PDF 转换分钟级只有预览面板可见。本卡由
// docmd OCR 进度钩子（wire ocr_progress）驱动，按 path 原地 upsert 同一张卡：
// 引擎拉起（冷启动分钟级，进度价值最大的阶段）→ 页粒度进度条 → 缓存命中/
// 完成/失败终态。
//
// 数据依赖：store item { kind: "ocr", path, name, engine, stage, done, total,
// chars, err }；stage 词汇与 docmd.OCRProgressEvent 对齐。事件不落历史——
// 恢复会话后卡片消失（OCR 结果本体经注入块/工具卡持久化，卡片只是活进度面）。

export type OCRItem = Extract<Item, { kind: "ocr" }>;

// STAGE_KEY 的键必须是 DictKey 成员（i18n 字典类型锁）；用 as 断言前先保证
// 键位已登记（work.trigger 同款锁形态）。

const STAGE_KEY: Record<OCRItem["stage"], DictKey> = {
  "engine-start": "ocrCard.stageEngine",
  running: "ocrCard.stageRunning",
  cached: "ocrCard.stageCached",
  done: "ocrCard.stageDone",
  failed: "ocrCard.stageFailed",
};

const IMAGE_EXTENSIONS = new Set(["png", "jpg", "jpeg", "webp", "bmp", "gif", "tif", "tiff"]);

function fileKind(name: string): "pdf" | "image" | "text" {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  if (ext === "pdf") return "pdf";
  if (IMAGE_EXTENSIONS.has(ext)) return "image";
  return "text";
}

function clampedPages(done: number, total: number): number {
  if (total <= 0) return 0;
  return Math.max(0, Math.min(total, Math.floor(done)));
}

export function OCRCard({ item }: { item: OCRItem }) {
  const { locale, t } = useI18n();
  const titleId = useId();
  const running = item.stage === "engine-start" || item.stage === "running";
  const kind = fileKind(item.name);
  const isPDF = kind === "pdf";
  const isPaged = isPDF && item.total > 1;
  const done = clampedPages(item.done, item.total);
  const pct = item.total > 0 ? Math.min(100, Math.round((done / item.total) * 100)) : null;
  const stageLabel = t(STAGE_KEY[item.stage] ?? "ocrCard.stageRunning");
  const FileIcon = kind === "pdf" ? FilePdf : kind === "image" ? FileImage : FileText;
  const StatusIcon = item.stage === "failed"
    ? AlertCircle
    : item.stage === "cached" || item.stage === "done"
      ? CheckCircle
      : Scan;
  const chars = useMemo(
    () => new Intl.NumberFormat(locale).format(Math.max(0, item.chars)),
    [item.chars, locale],
  );

  return (
    <section
      className="ocr-card"
      data-testid="ocr-card"
      data-stage={item.stage}
      data-file-kind={kind}
      aria-labelledby={titleId}
      aria-busy={running}
    >
      <header className="ocr-card-head">
        <span className="ocr-card-file" data-kind={kind} aria-hidden>
          <FileIcon size={18} />
          {running && <i className="ocr-card-scanline" />}
        </span>
        <span className="ocr-card-copy">
          <span className="ocr-card-label">{t("ocrCard.label")}</span>
          <h3 id={titleId} className="ocr-card-name" title={item.path}>{item.name}</h3>
        </span>
        <span className="ocr-card-aside">
          {item.engine !== "" && (
            <span
              className="ocr-card-engine"
              aria-label={t("ocrCard.engineAria", { engine: item.engine })}
              title={t("ocrCard.engineAria", { engine: item.engine })}
            >
              {item.engine}
            </span>
          )}
          <span className="ocr-card-status" data-stage={item.stage} role="status" aria-live="polite" aria-atomic="true">
            <StatusIcon size={12} aria-hidden />
            {stageLabel}
          </span>
        </span>
      </header>
      {running && item.stage === "engine-start" && (
        <div className="ocr-card-body ocr-card-engine-start">
          <Scan size={15} aria-hidden />
          <span>{t("ocrCard.engineHint")}</span>
        </div>
      )}
      {running && isPaged && item.stage === "running" && pct !== null && (
        <div className="ocr-card-body ocr-card-progress-body">
          <div className="ocr-card-progress-meta">
            <span>{t("ocrCard.runningPages", { done, total: item.total })}</span>
            <strong className="tabular-nums">{pct}%</strong>
          </div>
          <div
            className="ocr-card-bar"
            role="progressbar"
            aria-label={t("ocrCard.progressAria", { done, total: item.total })}
            aria-valuemin={0}
            aria-valuemax={item.total}
            aria-valuenow={done}
            aria-valuetext={t("ocrCard.progressAria", { done, total: item.total })}
          >
            <i style={{ transform: `scaleX(${pct / 100})` }} />
          </div>
          <div className="ocr-card-page-meta">
            <span>{t("ocrCard.pages", { done, total: item.total })}</span>
            <span>{t("ocrCard.runningHint")}</span>
          </div>
        </div>
      )}
      {running && !isPaged && item.stage === "running" && (
        <div className="ocr-card-body ocr-card-image-running">
          <Scan size={15} aria-hidden />
          <span>{t(isPDF ? "ocrCard.runningPage" : "ocrCard.runningImage")}</span>
        </div>
      )}
      {(item.stage === "done" || item.stage === "cached") && (
        <div className="ocr-card-body ocr-card-outcome">
          <span className="ocr-card-outcome-icon" aria-hidden><CheckCircle size={15} /></span>
          <span className="ocr-card-outcome-copy">
            <strong>{item.stage === "cached" ? t("ocrCard.cachedHint") : t("ocrCard.doneHint")}</strong>
            <span>
              {t("ocrCard.chars", { n: chars })}
              {isPDF && <span className="ocr-card-stat-sep"> · </span>}
              {isPDF && t("ocrCard.pages", { done: item.total, total: item.total })}
            </span>
          </span>
        </div>
      )}
      {item.stage === "failed" && item.err !== "" && (
        <div className="ocr-card-body ocr-card-error" role="alert">
          <AlertCircle size={15} aria-hidden />
          <span className="ocr-card-error-copy">
            <strong>{item.err}</strong>
            <span>{t("ocrCard.failedHint")}</span>
          </span>
        </div>
      )}
    </section>
  );
}
