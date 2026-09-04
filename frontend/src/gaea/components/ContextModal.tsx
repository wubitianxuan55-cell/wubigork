// ContextModal + ContextPill（蒸馏规划 2.5e + 调研回填「常驻上下文徽标」）。
//
// ContextPill：codex 式「剩余上下文 %」常驻徽标——数据来自会话 store 的
// ContextUsage（每轮/加载/回合末自动刷新），窗口未知（win<=0）时不渲染；
// 配色三档（≥90% err / ≥75% warning / 常规），点击打开居中弹层。
//
// ContextModal：dsh「/context 居中弹层」同款语义——不离开对话查看当前
// 上下文构成，内容复用 ContextView（与主区上下文 tab 同一组件，打开时
// 挂载拉最新快照，关闭即卸载）。斜杠命令 /context 从「切主区 tab」改道
// 到本弹层（主区 tab 仍保留，手动切换路径不变）。
import { Modal } from "antd";
import { fmtTokens } from "../lib/stats";
import { ContextView } from "./ContextView";

export function ContextPill({ used, window: win, onClick }: {
  used: number;
  window: number;
  onClick?: () => void;
}) {
  if (!(win > 0)) return null;
  const pct = Math.min(100, Math.round((used / win) * 100));
  const remain = Math.max(0, 100 - pct);
  const tone =
    pct >= 90
      ? "var(--md-sys-color-destructive)"
      : pct >= 75
        ? "var(--md-sys-color-warning, #f59e0b)"
        : "var(--md-sys-color-text-secondary)";
  return (
    <button
      type="button"
      data-testid="ctx-pill"
      onClick={onClick}
      title={`上下文窗口占用 ${fmtTokens(used)}/${fmtTokens(win)} · 点击查看构成`}
      className="inline-flex cursor-pointer items-center gap-1.5 rounded-full border px-2 py-0.5 text-[10.5px] tabular-nums transition-colors hover:bg-bg-soft"
      style={{ borderColor: "var(--md-sys-color-outline-variant)", color: tone, background: "transparent" }}
    >
      <span className="inline-block h-1 w-10 overflow-hidden rounded-full bg-bg-soft align-middle">
        <span
          className="block h-full rounded-full transition-all duration-500"
          style={{ width: `${pct}%`, background: tone }}
        />
      </span>
      <span>剩余 {remain}%</span>
    </button>
  );
}

export function ContextModal({ open, onClose, running, sessionPath, sessionName, model }: {
  open: boolean;
  onClose: () => void;
  running: boolean;
  sessionPath?: string;
  sessionName?: string;
  model?: string;
}) {
  return (
    <Modal
      open={open}
      onCancel={onClose}
      footer={null}
      width={1080}
      centered
      title="当前上下文"
      destroyOnHidden
    >
      {open && (
        <div className="max-h-[68vh] overflow-y-auto" data-testid="ctx-modal-body">
          <ContextView running={running} sessionPath={sessionPath} sessionName={sessionName} model={model} />
        </div>
      )}
    </Modal>
  );
}
