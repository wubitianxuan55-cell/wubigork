import { AlertCircle, CheckCircle, FileImage, Loader2 } from "../icons";
import { useI18n } from "../lib/i18n";
import type { DictKey } from "../locales/en";
import type { Item } from "../lib/store";
import { FileThumb } from "./FileThumb";

// ImageGenCard — 生图作业卡（2026-11-20，OCR 大卡同族）。
//
// Why：image_gen 云端生图数十秒到分钟级，此前对话流全程静默（工具卡只在
// dispatch/result 两点跳变），办公/文学接入生图后（spacetags play→shared）
// 用户没有任何进度可感。本卡由工具内置 wire 事件（imagegen_progress，与
// OCR ocr_progress 同通道：不进账本不消费 seq）驱动，按 id 原地 upsert：
// running（提示词+模型+张数，脉冲图标）→ done（文件名+缩略）/failed。
//
// 数据依赖：store item { kind: "imagegen", id, prompt, model, stage,
// files, err }。事件不落历史——恢复会话后卡片消失（生图结果本体经工具卡
// markdown 图片链接持久化，卡片只是活进度面）。缩略走 FileThumb（缓存
// dataURL，失败静默回退图标）。

export type ImageGenItem = Extract<Item, { kind: "imagegen" }>;

const STAGE_KEY: Record<ImageGenItem["stage"], DictKey> = {
  running: "imagegenCard.stageRunning",
  done: "imagegenCard.stageDone",
  failed: "imagegenCard.stageFailed",
};

export function ImageGenCard({ item }: { item: ImageGenItem }) {
  const { t } = useI18n();
  const running = item.stage === "running";
  const StatusIcon = item.stage === "failed"
    ? AlertCircle
    : item.stage === "done"
      ? CheckCircle
      : Loader2;
  const statusLabel = t(STAGE_KEY[item.stage]);
  return (
    <div className="imggen-card" data-stage={item.stage} data-testid="imggen-card">
      <div className="imggen-card-head">
        <span className="imggen-card-icon" aria-hidden>
          <FileImage size={17} />
        </span>
        <span className="imggen-card-title">
          <span className="imggen-card-kind">{t("imagegenCard.kind")}</span>
          <span className="imggen-card-model" title={item.model}>{item.model}</span>
        </span>
        <span className="imggen-card-status" data-stage={item.stage}>
          <StatusIcon size={12} className={running ? "animate-spin" : "shrink-0"} />
          <span>{statusLabel}</span>
        </span>
      </div>
      <div className="imggen-card-body">
        {item.prompt && (
          <p className="imggen-card-prompt" title={item.prompt}>{item.prompt}</p>
        )}
        {item.stage === "done" && item.files.length > 0 && (
          <div className="imggen-card-files">
            {item.files.map((f) => (
              <span key={f} className="imggen-card-file">
                <FileThumb path={f} ext={f.slice(f.lastIndexOf("."))} imgClassName="w-8 h-8 object-cover rounded-[5px] border border-border-soft" />
                <span title={f}>{f.split(/[\\/]/).pop() ?? f}</span>
              </span>
            ))}
          </div>
        )}
        {item.stage === "failed" && item.err && (
          <p className="imggen-card-err">{item.err}</p>
        )}
      </div>
    </div>
  );
}
