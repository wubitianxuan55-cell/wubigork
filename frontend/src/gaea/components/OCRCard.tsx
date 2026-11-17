import { FileText } from "../icons";
import { useT } from "../lib/i18n";
import type { DictKey } from "../locales/en";
import type { Item } from "../lib/store";

// OCRCard — OCR 专属大卡（2026-11-18）：办公板块文件 OCR 解析的进度/状态面。
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

export function OCRCard({ item }: { item: OCRItem }) {
  const t = useT();
  const running = item.stage === "engine-start" || item.stage === "running";
  const isPDF = item.total > 1;
  const pct = item.total > 0 ? Math.min(100, Math.round((item.done / item.total) * 100)) : null;
  const stageLabel = t(STAGE_KEY[item.stage] ?? "ocrCard.stageRunning");

  return (
    <div
      className="ctx-card ocr-card"
      data-testid="ocr-card"
      data-stage={item.stage}
      title={item.err || item.path}
    >
      <div className="ocr-card-head">
        <FileText size={14} className="shrink-0" aria-hidden />
        <span className="ocr-card-name" title={item.path}>{item.name}</span>
        {item.engine !== "" && <span className="ocr-card-engine">{item.engine}</span>}
        <span className="ocr-card-status" data-stage={item.stage}>
          <i aria-hidden />
          {stageLabel}
        </span>
      </div>
      {running && isPDF && (
        <div className="ocr-card-progress">
          <span className="ocr-card-bar" aria-hidden>
            <i style={{ width: `${pct ?? 0}%` }} />
          </span>
          <span className="ocr-card-pages tabular-nums">
            {t("ocrCard.pages", { done: item.done, total: item.total })}
          </span>
        </div>
      )}
      {running && !isPDF && item.stage === "engine-start" && (
        <div className="ocr-card-progress">
          <span className="ocr-card-hint">{t("ocrCard.engineHint")}</span>
        </div>
      )}
      {(item.stage === "done" || item.stage === "cached") && (
        <div className="ocr-card-result">
          <span>{t("ocrCard.chars", { n: item.chars })}</span>
        </div>
      )}
      {item.stage === "failed" && item.err !== "" && (
        <div className="ocr-card-error">{item.err}</div>
      )}
    </div>
  );
}
