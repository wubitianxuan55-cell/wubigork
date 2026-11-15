import { useCallback, useState } from "react";
import { Button, Drawer, message } from "antd";
import { ExperimentOutlined, PlayCircleOutlined } from "@ant-design/icons";
import { app } from "../../lib/bridge";
import type { MemoryEvalReport } from "../../lib/types/memory";

/**
 * MemoryEvalSection — 记忆注入体检（市场调研候选2，Drawer 内嵌于办公记忆库）。
 *
 * 对真实库的 work 视图跑与装配点同款的两个注入构建器（晨报预载 + 项目本体），
 * 断言预算/泄漏/悬空五条结构不变量；与检索质量测评（模型中心）分立——
 * 那个测「查得到」，这个测「注得对」。设计档
 * docs/gaea-memory-injection-eval-design-2026-09.md。
 */

function Row({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="flex items-baseline justify-between gap-3 py-1">
      <span className="text-[12px] text-fg-faint shrink-0">{label}</span>
      <span className="text-[12px] text-fg text-right min-w-0" title={hint}>
        {value}
      </span>
    </div>
  );
}

export function MemoryEvalSection({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [report, setReport] = useState<MemoryEvalReport | null>(null);
  const [running, setRunning] = useState(false);

  const run = useCallback(async () => {
    setRunning(true);
    try {
      const r = await app.MemoryEvalRun();
      setReport(r);
      message.success(
        r.passed
          ? "记忆注入体检通过"
          : `记忆注入体检发现 ${r.violations?.length ?? 0} 条违规`,
      );
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : "记忆注入体检失败");
    } finally {
      setRunning(false);
    }
  }, []);

  return (
    <Drawer
      title="记忆注入体检"
      placement="right"
      width={460}
      open={open}
      onClose={onClose}
      destroyOnClose
      extra={
        <Button
          size="small"
          type="primary"
          icon={<PlayCircleOutlined />}
          loading={running}
          onClick={() => void run()}
        >
          运行体检
        </Button>
      }
    >
      <div className="flex items-start gap-2 text-[11.5px] text-fg-faint mb-3">
        <ExperimentOutlined className="mt-0.5 shrink-0" />
        <span>
          对当前办公记忆库跑与 work 会话装配同款的两个注入构建器（晨报预载 +
          项目本体），检查预算合规、归档/跨空间泄漏与引用悬空。确定性纯读，不调用模型。
        </span>
      </div>

      {!report ? (
        <div className="text-[12px] text-fg-faint">尚未运行：点右上「运行体检」。</div>
      ) : (
        <div className="flex flex-col gap-3">
          <div
            className={`px-3 py-2 rounded-lg text-[12px] font-medium ${
              report.passed
                ? "bg-emerald-500/10 text-emerald-500"
                : "bg-rose-500/10 text-rose-500"
            }`}
          >
            {report.passed ? "✓ 通过：五条结构不变量全部成立" : "✗ 未通过：存在违规项"}
          </div>

          <div className="rounded-lg border border-border px-3 py-2">
            <Row
              label="晨报预载块"
              value={
                report.preloadPresent
                  ? `${report.entryCount} 条 · ${report.preloadRunes}/${report.preloadBudget} runes`
                  : "未注入"
              }
              hint="work 会话装配的高频工作记忆块"
            />
            <Row
              label="项目本体块"
              value={
                report.briefPresent
                  ? `${report.refCount} 引用 · ${report.briefRunes}/${report.briefBudget} runes`
                  : "未注入"
              }
              hint="固化/项目/反馈决策带 [MEM:] 引用键的注入块"
            />
            <Row
              label="固化覆盖"
              value={
                report.pinnedTotal === 0
                  ? "库内无固化条"
                  : `${report.pinnedInBrief}/${report.pinnedTotal}${
                      report.missingPinned?.length
                        ? `（未进：${report.missingPinned.join("、")}）`
                        : ""
                    }`
              }
              hint="固化全收是设计口径；预算挤兑下未进只透出不判死"
            />
          </div>

          {report.violations && report.violations.length > 0 && (
            <div className="rounded-lg border border-rose-500/40 px-3 py-2">
              <div className="text-[12px] font-medium text-rose-500 mb-1">违规项</div>
              <ul className="list-disc pl-4 text-[12px] text-fg flex flex-col gap-0.5">
                {report.violations.map((v, i) => (
                  <li key={i}>{v}</li>
                ))}
              </ul>
            </div>
          )}

          <div className="text-[11px] text-fg-faint flex flex-col gap-0.5">
            <span>
              门控：记忆{report.memoryEnabled ? "开" : "关"} · 晨报预载
              {report.morningPreload ? "开" : "关"} · 项目本体
              {report.projectBrief ? "开" : "关"} · 空间分区
              {report.spaceModeOn ? "on" : "off"}
            </span>
            {report.note && <span>{report.note}</span>}
            <span>口径：门槛=零违规（确定性规则，与检索测评的 Recall@10 统计门槛分立）。</span>
          </div>
        </div>
      )}
    </Drawer>
  );
}
