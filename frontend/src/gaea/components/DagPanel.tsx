import { useCallback, useEffect, useState } from "react";
import { Check, ChevronDown, ListTree, Loader2 } from "../icons";
import { app } from "../lib/bridge";
import type { DagNodeView, DagRunView } from "../lib/types";
import { usePreviewStore } from "../lib/store";
import { useT } from "../lib/i18n";

// DagPanel — 任务视图「办公流水线」区块（6.3 办公多文件 DAG 首刀）。
//
// 数据源 = GaeaDagList（挂载自拉一次）；派生态 derived=running 的 run 存在时
// 每 2.5s 轮询自校正（后端受理型契约不推事件流），无 running 停轮询；所有
// 操作（起跑/单跑/重跑/验收/改向/终止）成功与否都立刻重拉一次。
//
// 与子代理区块同构：可折叠分组头（点击整块收展）+ 分区计数 + 空态/失败态分离
// （拉取失败给重试入口，不用「暂无」冒充失败态）。产物 chips 点击走全局预览
// 队列（usePreviewStore.openFilePreview，与资源管理器/交付面板同通道）。
//
// 口径注（诚实不造精确）：卡片底部小字说明「产物=运行窗口内新增证据卡」，
// 主对话同期写盘会并入，不承诺逐节点精确清单（精确归档属 Go 侧后续刀）。

const DAG_POLL_MS = 2_500;

/** 节点状态点/词：pending 灰 / running 蓝脉冲 / done 绿 / failed 红 / skipped 灰划线 / accepted 绿+勾。 */
const NODE_STATUS: Record<
  DagNodeView["status"],
  { label: string; cls: string; style?: string }
> = {
  pending: { label: "待跑", cls: "", style: "var(--md-sys-color-outline-variant)" },
  running: { label: "运行中", cls: "bg-accent animate-pulse" },
  done: { label: "完成", cls: "bg-ok" },
  failed: { label: "失败", cls: "bg-err" },
  skipped: { label: "跳过", cls: "", style: "var(--md-sys-color-outline-variant)" },
  accepted: { label: "已验收", cls: "bg-ok" },
};

/** run 派生徽标：与契约 derived 语义逐条对应（draft/running/failed/ready/accepted）。 */
const DERIVED_META: Record<
  DagRunView["derived"],
  { label: string; color: string }
> = {
  draft: { label: "未跑", color: "var(--md-sys-color-text-secondary)" },
  running: { label: "运行中", color: "var(--gaea-glow)" },
  failed: { label: "有失败", color: "var(--md-sys-color-error)" },
  ready: { label: "待验收", color: "var(--md-sys-color-primary)" },
  accepted: { label: "已验收", color: "var(--md-sys-color-primary)" },
};

const btnStyle = {
  color: "var(--md-sys-color-text-secondary)",
  border: "1px solid var(--md-sys-color-outline-variant)",
} as const;

export function DagPanel() {
  const t = useT();
  // v6.3：整块可折叠（镜像子代理区块交互），默认展开
  const [open, setOpen] = useState(true);
  const [runs, setRuns] = useState<DagRunView[] | null>(null);
  const [failed, setFailed] = useState(false);
  // run 卡展开表（runId → 节点列表是否展开）；steer 输入草稿按 run:node 记。
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [steerDrafts, setSteerDrafts] = useState<Record<string, string>>({});
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const openFilePreview = usePreviewStore((s) => s.openFilePreview);

  const reload = useCallback(async () => {
    try {
      const rs = await app.DagList();
      setRuns(rs);
      setFailed(false);
    } catch {
      // 失败保留旧快照；首拉失败（无数据）时由下方 failed 空态出重试入口
      setFailed(true);
    }
  }, []);

  useEffect(() => {
    void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 自校正轮询：存在 running 的 run 才起 2.5s 定时器，无 running 停轮询。
  const anyRunning = (runs ?? []).some((r) => r.derived === "running");
  useEffect(() => {
    if (!anyRunning) return;
    const h = setInterval(() => {
      void (async () => {
        try {
          setRuns(await app.DagList());
          setFailed(false);
        } catch {
          // 轮询失败保旧快照，下个 tick 继续
        }
      })();
    }, DAG_POLL_MS);
    return () => clearInterval(h);
  }, [anyRunning]);

  // 统一操作通道：防在途重入 + 完成后立刻重拉（成败都拉，受理文案不细究）。
  const op = useCallback(
    async (key: string, fn: () => Promise<unknown>) => {
      if (busyKey) return;
      setBusyKey(key);
      try {
        await fn();
      } catch {
        // 后端拒绝（如节点状态不允许）：保留旧快照，重拉对齐真实态
      } finally {
        setBusyKey(null);
      }
      void reload();
    },
    [busyKey, reload],
  );

  const runAction = (run: DagRunView) => {
    const k = `run:${run.id}`;
    return (
      <>
        {(run.derived === "draft" || run.derived === "failed" || run.derived === "ready") && (
          <button
            type="button"
            data-testid={`dag-run-start-${run.id}`}
            disabled={busyKey !== null}
            className="cursor-pointer rounded-md px-1.5 py-0.5 text-[10.5px] font-medium disabled:opacity-50"
            style={btnStyle}
            onClick={() => void op(k, () => app.DagRun(run.id))}
            title="起跑/续跑整链（按依赖顺序推进未完成节点）"
          >
            起跑/续跑
          </button>
        )}
        {run.derived === "running" && (
          <button
            type="button"
            data-testid={`dag-run-cancel-${run.id}`}
            disabled={busyKey !== null}
            className="cursor-pointer rounded-md px-1.5 py-0.5 text-[10.5px] font-medium disabled:opacity-50"
            style={{ ...btnStyle, color: "var(--md-sys-color-error)" }}
            onClick={() => void op(k, () => app.DagCancel(run.id))}
            title="终止级联：停推进，未跑节点不再起跑"
          >
            终止
          </button>
        )}
      </>
    );
  };

  const nodeActions = (run: DagRunView, node: DagNodeView) => {
    const k = `node:${run.id}:${node.id}`;
    const doneLike = node.status === "done" || node.status === "accepted";
    const incomplete =
      node.status === "pending" || node.status === "running" || node.status === "skipped";
    return (
      <div className="flex shrink-0 items-center gap-1">
        {doneLike && (
          <button
            type="button"
            data-testid={`dag-node-rerun-${run.id}-${node.id}`}
            disabled={busyKey !== null}
            className="cursor-pointer rounded-md border-0 bg-transparent px-1 py-0.5 text-[10.5px] underline-offset-2 hover:underline disabled:opacity-50"
            style={{ color: "var(--md-sys-color-text-secondary)" }}
            onClick={() => void op(k, () => app.DagNodeRun(run.id, node.id))}
            title="重跑本节点（不动下游）"
          >
            重跑
          </button>
        )}
        {(node.status === "done" || node.status === "failed") && (
          <button
            type="button"
            data-testid={`dag-node-accept-${run.id}-${node.id}`}
            disabled={busyKey !== null}
            className="cursor-pointer rounded-md border-0 bg-transparent px-1 py-0.5 text-[10.5px] underline-offset-2 hover:underline disabled:opacity-50"
            style={{ color: "var(--md-sys-color-primary)" }}
            onClick={() => void op(k, () => app.DagNodeAccept(run.id, node.id))}
            title="验收（人拍板，产物回流记忆）"
          >
            验收
          </button>
        )}
        {incomplete && (
          <button
            type="button"
            data-testid={`dag-node-run-${run.id}-${node.id}`}
            disabled={busyKey !== null}
            className="cursor-pointer rounded-md border-0 bg-transparent px-1 py-0.5 text-[10.5px] underline-offset-2 hover:underline disabled:opacity-50"
            style={{ color: "var(--md-sys-color-text-secondary)" }}
            onClick={() => void op(k, () => app.DagNodeRun(run.id, node.id))}
            title="单跑本节点"
          >
            单跑
          </button>
        )}
      </div>
    );
  };

  const steerBox = (run: DagRunView, node: DagNodeView) => {
    if (node.status !== "done" && node.status !== "failed") return null;
    const key = `${run.id}:${node.id}`;
    const k = `steer:${key}`;
    const val = steerDrafts[key] ?? "";
    const send = () => {
      const prompt = val.trim();
      if (!prompt) return;
      setSteerDrafts((m) => ({ ...m, [key]: "" }));
      void op(k, () => app.DagNodeSteer(run.id, node.id, prompt));
    };
    return (
      <div className="mt-1 flex items-center gap-1">
        <input
          data-testid={`dag-steer-input-${run.id}-${node.id}`}
          value={val}
          placeholder="对本节点补充指令…"
          className="min-w-0 flex-1 rounded-md px-1.5 py-0.5 text-[11px] outline-none"
          style={{
            background: "var(--md-sys-color-surface-container-high)",
            border: "1px solid var(--md-sys-color-outline-variant)",
            color: "var(--md-sys-color-text)",
          }}
          onChange={(e) => setSteerDrafts((m) => ({ ...m, [key]: e.target.value }))}
          onKeyDown={(e) => {
            if (e.key === "Enter") send();
          }}
        />
        <button
          type="button"
          data-testid={`dag-steer-send-${run.id}-${node.id}`}
          disabled={busyKey !== null || !val.trim()}
          className="shrink-0 cursor-pointer rounded-md px-1.5 py-0.5 text-[10.5px] font-medium disabled:opacity-50"
          style={btnStyle}
          onClick={send}
          title="发送改向指令（对已完成节点续跑）"
        >
          发送
        </button>
      </div>
    );
  };

  const loading = runs === null && !failed;
  const list = runs ?? [];
  const runningCount = list.filter((r) => r.derived === "running").length;

  return (
    <div data-testid="dag-section" className="mt-1 border-t border-border-soft" style={{ paddingTop: 2 }}>
      <button
        type="button"
        data-testid="dag-section-toggle"
        aria-expanded={open}
        className="flex w-full items-center gap-1.5 px-2 pt-2 pb-1 text-left text-[10px] uppercase tracking-wider cursor-pointer border-0 bg-transparent transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
        style={{ color: "var(--md-sys-color-text-secondary)" }}
        onClick={() => setOpen((v) => !v)}
        title={open ? "收起办公流水线区块" : "展开办公流水线区块"}
      >
        <ChevronDown
          size={10}
          aria-hidden
          className="transition-transform duration-150"
          style={{ transform: open ? "rotate(0deg)" : "rotate(-90deg)" }}
        />
        <ListTree size={10} aria-hidden />
        {t("dag.sectionTitle")}
        <span className="ml-auto normal-case font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {runningCount > 0 ? `${runningCount} 运行中` : list.length > 0 ? `${list.length} 条` : ""}
        </span>
      </button>

      {open && (
        <>
          {loading && (
            <div className="flex items-center gap-2 px-4 py-4 text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              <Loader2 size={13} className="animate-spin" />
              {t("dag.loading")}
            </div>
          )}
          {!loading && failed && list.length === 0 && (
            <div className="flex flex-col items-center gap-2 px-6 py-6 text-center">
              <span className="text-[11px] leading-relaxed" style={{ color: "var(--md-sys-color-error)" }}>
                {t("dag.loadFail")}
              </span>
              <button
                type="button"
                data-testid="dag-retry"
                className="cursor-pointer rounded-md border px-2 py-0.5 text-[11px] transition-colors"
                onClick={() => void reload()}
                style={{
                  color: "var(--md-sys-color-error)",
                  border: "1px solid color-mix(in srgb, var(--md-sys-color-error) 40%, transparent)",
                  background: "color-mix(in srgb, var(--md-sys-color-error) 8%, transparent)",
                }}
              >
                {t("subagent.retry")}
              </button>
            </div>
          )}
          {!loading && !failed && list.length === 0 && (
            <div className="px-4 py-4 text-center text-[11px] leading-relaxed" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {t("dag.sectionEmpty")}
            </div>
          )}
          <div className="flex flex-col gap-1 px-1 pb-1">
            {list.map((run) => {
              const meta = DERIVED_META[run.derived];
              const isOpen = expanded[run.id] ?? false;
              return (
                <div
                  key={run.id}
                  data-testid={`dag-run-${run.id}`}
                  className="rounded-md px-1.5 py-1.5 transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
                >
                  {/* run 头：goal + 派生徽标 + 节点数 + run 级操作 */}
                  <div className="flex items-start gap-2">
                    <button
                      type="button"
                      className="mt-[2px] shrink-0 cursor-pointer rounded border-0 bg-transparent p-0"
                      style={{ color: "var(--md-sys-color-text-secondary)" }}
                      aria-label={isOpen ? "收起节点列表" : "展开节点列表"}
                      onClick={() => setExpanded((m) => ({ ...m, [run.id]: !isOpen }))}
                    >
                      <ChevronDown
                        size={10}
                        aria-hidden
                        className="transition-transform duration-150"
                        style={{ transform: isOpen ? "rotate(0deg)" : "rotate(-90deg)" }}
                      />
                    </button>
                    <div className="min-w-0 flex-1">
                      <button
                        type="button"
                        className="block w-full cursor-pointer truncate border-0 bg-transparent p-0 text-left text-[12px] leading-snug"
                        style={{ color: "var(--md-sys-color-text)" }}
                        title={run.goal}
                        onClick={() => setExpanded((m) => ({ ...m, [run.id]: !isOpen }))}
                      >
                        {run.goal}
                      </button>
                      <div className="mt-0.5 flex items-center gap-1.5 text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                        <span
                          className="rounded-full px-1"
                          style={{ color: meta.color, background: "var(--md-sys-color-surface-container-high)" }}
                        >
                          {meta.label}
                        </span>
                        <span className="font-mono">{run.nodes.length} 节点</span>
                      </div>
                    </div>
                    {runAction(run)}
                  </div>
                  {/* 节点列表 */}
                  {isOpen && (
                    <div className="mt-1 flex flex-col gap-0.5 border-t border-border-soft pt-1">
                      {run.nodes.map((node) => {
                        const st = NODE_STATUS[node.status];
                        return (
                          <div
                            key={node.id}
                            data-testid={`dag-node-${run.id}-${node.id}`}
                            className="rounded-md px-1 py-1"
                          >
                            <div className="flex items-start gap-2">
                              <span
                                data-testid={`dag-node-status-${run.id}-${node.id}`}
                                title={st.label}
                                className={`mt-[5px] h-2 w-2 shrink-0 rounded-full ${st.cls}`}
                                style={st.style ? { background: st.style } : undefined}
                              />
                              <div className="min-w-0 flex-1">
                                <div className="flex items-center gap-1">
                                  <span
                                    className={`truncate text-[12px] leading-snug ${node.status === "skipped" ? "line-through" : ""}`}
                                    style={{
                                      color:
                                        node.status === "failed"
                                          ? "var(--md-sys-color-error)"
                                          : "var(--md-sys-color-text)",
                                    }}
                                    title={`${node.title}（${st.label}${(node.steerCount ?? 0) > 0 ? ` · 改向 ${node.steerCount} 次` : ""}）`}
                                  >
                                    {node.title}
                                  </span>
                                  {node.status === "accepted" && (
                                    <Check size={10} aria-hidden style={{ color: "var(--md-sys-color-primary)" }} />
                                  )}
                                </div>
                                {(node.outputs ?? []).length > 0 && (
                                  <div className="mt-0.5 flex flex-wrap gap-1">
                                    {(node.outputs ?? []).map((out) => (
                                      <button
                                        key={out}
                                        type="button"
                                        className="max-w-full cursor-pointer truncate rounded px-1 py-px font-mono text-[10px] border-0"
                                        style={{
                                          background: "var(--md-sys-color-surface-container-high)",
                                          color: "var(--md-sys-color-text-secondary)",
                                        }}
                                        title={`预览产物：${out}`}
                                        onClick={() => openFilePreview(out)}
                                      >
                                        {out}
                                      </button>
                                    ))}
                                  </div>
                                )}
                                {node.status === "failed" && node.error && (
                                  <div className="mt-0.5 text-[10px] leading-snug" style={{ color: "var(--md-sys-color-error)" }}>
                                    {node.error}
                                  </div>
                                )}
                                {steerBox(run, node)}
                              </div>
                              {nodeActions(run, node)}
                            </div>
                          </div>
                        );
                      })}
                      {/* 口径注：产物=运行窗口内新增证据卡，主对话同期写盘会并入——诚实不造精确 */}
                      <div className="px-1 pb-0.5 pt-0.5 text-[10px] leading-snug" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                        {t("dag.scopeNote")}
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </>
      )}
    </div>
  );
}
