// DirectorModal.tsx — 绘梦导演（11.6；roadmap 绘梦#1）
// 自然语言需求 → 后端解析（LLM 计划）→ 生成 → 视觉自评 → 带参重试（有界）。
// 轨迹经 image-director-step 事件实时追加 + 回执全量步骤；诚实口径：accepted
// =false 时显式标「未达标」，自评不可用也如实呈现。
import { useEffect, useRef, useState } from "react";
import { Alert, Modal, Spin } from "antd";
import { app } from "../../gaea/lib/bridge";
import { subscribeWailsEvent } from "../../gaea/lib/wailsEvents";

interface DirectorPlan {
  model: string;
  prompt: string;
  negative?: string;
  style?: string;
  loras?: string[];
  reason?: string;
}

interface DirectorVerdict {
  pass: boolean;
  score: number;
  issues?: string[];
  fix?: string;
}

interface DirectorStep {
  stage: string; // plan|generate|eval|adjust|done
  attempt: number;
  detail: string;
}

interface DirectorImage {
  image: string;
  seed: number;
  prompt: string;
  model: string;
  size: string;
  file_path?: string;
}

interface DirectorResult {
  images: DirectorImage[];
  plan?: DirectorPlan;
  verdict?: DirectorVerdict;
  steps: DirectorStep[];
  attempts: number;
  accepted: boolean;
}

interface DirectorStepEvent {
  stage: string;
  attempt: number;
  detail: string;
}

interface DirectorModalProps {
  open: boolean;
  onClose: () => void;
}

const ASPECTS = [
  { value: "1:1", label: "1:1 方形" },
  { value: "16:9", label: "16:9 横幅" },
  { value: "9:16", label: "9:16 竖幅" },
  { value: "4:3", label: "4:3 横" },
  { value: "3:4", label: "3:4 竖" },
];

const STAGE_LABEL: Record<string, string> = {
  plan: "解析",
  generate: "生成",
  eval: "自评",
  adjust: "调整",
  done: "完成",
};

export function DirectorModal({ open, onClose }: DirectorModalProps) {
  const [need, setNeed] = useState("");
  const [aspect, setAspect] = useState("1:1");
  const [maxAttempts, setMaxAttempts] = useState(2);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [result, setResult] = useState<DirectorResult | null>(null);
  const [liveSteps, setLiveSteps] = useState<DirectorStepEvent[]>([]);
  const logRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    setLiveSteps([]);
    if (!window.runtime?.EventsOn) return;
    return subscribeWailsEvent(window.runtime, "image-director-step", (ev: unknown) => {
      const step = ev as DirectorStepEvent;
      if (step && step.stage) setLiveSteps((prev) => [...prev, step]);
    });
  }, [open]);

  useEffect(() => {
    if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [liveSteps, result]);

  const run = async () => {
    setBusy(true);
    setError("");
    setResult(null);
    setLiveSteps([]);
    try {
      const raw = await app.GaeaImageDirector({
        need,
        aspect,
        maxAttempts,
      });
      setResult(raw as unknown as DirectorResult);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const stepsView = result ? result.steps.map((s) => ({ stage: s.stage, attempt: s.attempt, detail: s.detail })) : liveSteps;

  return (
    <Modal
      title="绘梦导演（描述需求，自动出图·自评·重试）"
      open={open}
      onCancel={() => { if (!busy) onClose(); }}
      width={760}
      footer={null}
      destroyOnHidden
    >
      <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        {!busy && !result && (
          <>
            <textarea
              data-testid="director-need"
              className="w-full h-24 rounded-lg border border-border bg-transparent p-2 text-[12.5px] text-fg"
              placeholder="用一句话描述要画什么，例如：「雨夜的都市天台，一位穿风衣的女性背影，霓虹倒影，电影感构图」"
              value={need}
              onChange={(e) => setNeed(e.target.value)}
            />
            <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
              <select
                className="h-7 rounded-lg border border-border bg-transparent px-2 text-[12px] text-fg"
                value={aspect}
                onChange={(e) => setAspect(e.target.value)}
                aria-label="构图比例"
              >
                {ASPECTS.map((a) => (
                  <option key={a.value} value={a.value}>{a.label}</option>
                ))}
              </select>
              <select
                className="h-7 rounded-lg border border-border bg-transparent px-2 text-[12px] text-fg"
                value={maxAttempts}
                onChange={(e) => setMaxAttempts(Number(e.target.value))}
                aria-label="自评轮数上限"
              >
                {[1, 2, 3].map((n) => (
                  <option key={n} value={n}>最多 {n} 轮</option>
                ))}
              </select>
              <div className="ml-auto" />
              <button type="button" className="px-3 h-7 rounded-lg border border-border text-fg text-[12px]" onClick={onClose}>
                取消
              </button>
              <button
                type="button"
                data-testid="director-run"
                className="px-3 h-7 rounded-lg bg-accent text-white text-[12px] disabled:opacity-50"
                disabled={!need.trim()}
                onClick={() => void run()}
              >
                开拍
              </button>
            </div>
          </>
        )}

        {busy && (
          <div style={{ textAlign: "center", padding: 16 }}>
            <Spin tip="导演工作中（解析→生成→自评，长调用请耐心）…" />
          </div>
        )}

        {error && <Alert type="error" showIcon message={error} />}

        {(stepsView.length > 0 || result) && (
          <div>
            <div className="text-[12px] text-fg-dim mb-1">导演轨迹</div>
            <div ref={logRef} data-testid="director-trace" style={{ maxHeight: 180, overflowY: "auto", fontSize: 12, border: "1px solid var(--border-subtle, #eee)", borderRadius: 8, padding: "4px 8px" }}>
              {stepsView.map((s, i) => (
                <div key={i} style={{ padding: "2px 0" }}>
                  <span className="font-mono text-fg-faint">[{STAGE_LABEL[s.stage] ?? s.stage}{s.attempt ? ` R${s.attempt}` : ""}]</span>{" "}
                  <span className="text-fg-dim">{s.detail}</span>
                </div>
              ))}
            </div>
          </div>
        )}

        {result && (
          <div data-testid="director-result">
            {!result.accepted && (
              <Alert
                type="warning"
                showIcon
                message={`已用尽 ${result.attempts} 轮自评未过——以下为末轮结果（诚实口径：未达标）`}
              />
            )}
            {result.images?.[0] && (
              <div style={{ display: "flex", gap: 12, marginTop: 8 }}>
                <img
                  src={result.images[0].image}
                  alt="导演成片"
                  style={{ maxWidth: 260, maxHeight: 260, borderRadius: 8, border: "1px solid var(--border-subtle, #eee)" }}
                />
                <div style={{ fontSize: 12, color: "var(--color-text-secondary)", display: "flex", flexDirection: "column", gap: 4 }}>
                  {result.plan && (
                    <>
                      <div>模型：{result.plan.model}{result.plan.reason ? `（${result.plan.reason}）` : ""}</div>
                      <div>提示词：{result.plan.prompt}</div>
                    </>
                  )}
                  {result.verdict && (
                    <div>
                      自评：{result.verdict.score} 分{result.verdict.issues?.length ? `，问题：${result.verdict.issues.join("；")}` : "，无拦截项"}
                    </div>
                  )}
                  <div>种子：{result.images[0].seed} · 尺寸：{result.images[0].size}</div>
                </div>
              </div>
            )}
            <div style={{ marginTop: 12, textAlign: "right" }}>
              <button
                type="button"
                className="px-3 h-7 rounded-lg border border-border text-fg text-[12px]"
                onClick={() => { setResult(null); setLiveSteps([]); }}
              >
                再导一部
              </button>
              <button type="button" className="ml-2 px-3 h-7 rounded-lg bg-accent text-white text-[12px]" onClick={onClose}>
                关闭
              </button>
            </div>
          </div>
        )}
      </div>
    </Modal>
  );
}
