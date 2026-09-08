// usePreviewAutoFront — 预览浮窗语义状态机（U2）+ 写后预览实时跟随（U4）接线。
// UI 布局预览自动置前/关闭优先/跨回合意图、以及 800ms 防抖刷新调度器全部迁出
// App.tsx，执行顺序与 deps 原样保留。仅消费 App 派生数据，不回写任何 App 状态。
import { useCallback, useEffect, useRef } from "react";
import { usePreviewStore } from "../lib/store";
import type { ControllerState } from "../lib/store";
import {
  createPreviewRefreshScheduler,
  initialPreviewAutoFrontState,
  isOfficeDeliverablePath,
  normalizePreviewPath,
  OFFICE_WRITE_TOOLS,
  previewAutoFrontReduce,
  extractOfficeWritePaths,
  type PreviewAutoFrontEvent,
} from "../lib/officeTurnProjection";
import { openPaneFileOrPreview } from "../lib/paneFileOpen";
import { usePaneTabsStore } from "../lib/paneTabs";
import { notifyScheduleFileChanged } from "../../schedule/store";

interface UsePreviewAutoFrontParams {
  state: ControllerState;
  currentSessionKey: string;
  workspacePanelOpen: boolean;
  previewFile: string | null;
  focusMode: boolean;
}

export function usePreviewAutoFront({ state, currentSessionKey, workspacePanelOpen, previewFile, focusMode }: UsePreviewAutoFrontParams) {
  // ── U2 预览浮窗语义状态机接线（docs/gaea-dsh-univer-office-distill-plan-2026-09.md
  // §4.2-5）：写弹读不弹（office 写类工具成功回执才自动置前预览）、关闭优先（用户
  // 手动关闭清除打开意图，同回合读取不复活）、意图跨回合（写意图未兑现跨 turnEnd
  // 保持）、终态清理（回合终结复位回合级状态）。状态迁移全部在 lib/
  // officeTurnProjection.previewAutoFrontReduce（纯函数，单测钉死），此处只做
  // 事件喂入与 open 动作执行（经 paneFileOpen 统一入口：工作台开 pane 文件 tab，
  // 未注册页面回落大预览）。置前范围收窄在 Office 文档路径，bash/脚本写盘不打扰。
  const previewAutoRef = useRef(initialPreviewAutoFrontState);
  // 工具卡状态基线：挂载/会话恢复/补拉快照只建基线不触发（与 v4.30 freshDeliverablePaths
  // 的 baselinePending 同式），此后状态跃迁才产生 dispatch/result 事件。
  const previewAutoToolSeenRef = useRef<Map<string, string>>(new Map());
  const previewAutoBaselineRef = useRef(true);
  const focusModeRef = useRef(focusMode);
  useEffect(() => { focusModeRef.current = focusMode; }, [focusMode]);

  // 返回值 = 状态机给出 open 且本机真的执行了置前（U4 刷新信号用它跳过刚
  // 置前的文件——挂载即加载最新）；专注模式/窄屏守卫拦下时返回 false（此时
  // 预览并未打开，isOpen 判定天然为假，刷新信号会被调度器丢弃）。
  const applyPreviewAutoEvent = useCallback((event: PreviewAutoFrontEvent): boolean => {
    const next = previewAutoFrontReduce(previewAutoRef.current, event);
    previewAutoRef.current = next.state;
    if (next.action.type !== "open") return false;
    // 专注模式 = 用户明确只要对话；窄屏（<1240）工作台 pane 被 CSS 隐藏——
    // 两种场景都不自动置前（与 openTasksAuto 的窄屏守卫同口径）。
    if (focusModeRef.current || window.innerWidth < 1240) return false;
    if (!isOfficeDeliverablePath(next.action.path)) return false;
    openPaneFileOrPreview(next.action.path);
    return true;
  }, []);

  // ── U4 写后预览实时跟随接线（docs/gaea-u4-render-evidence-inventory-2026-09.md
  // §3 推荐组合：GaeaPreview 直读刷新，零绑定）──写类工具成功回执 → 刷新调度器
  // （800ms 防抖合并连写，纯逻辑在 lib/officeTurnProjection.createPreviewRefresh
  // Scheduler，单测钉死触发/合并/关闭抑制）→ 目标文件此刻正打开在预览面才递增
  // paneTabs.reloadTicks → FilePreview 静默重读（不进 loading、不重挂、滚动位保持）。
  // 刷新≠置前：绝不打开已关闭的预览（U2 关闭优先/读不弹语义不变）；专注模式/
  // 窄屏下工作台面板本就关闭，isOpen 天然为假。主区大预览（previewFile）与 pane
  // 活动文件 tab 共用同一刷新总线。
  const previewRefreshCtxRef = useRef<{ panelOpen: boolean; previewFile: string | null }>({
    panelOpen: false,
    previewFile: null,
  });
  useEffect(() => {
    previewRefreshCtxRef.current = { panelOpen: workspacePanelOpen, previewFile };
  }, [workspacePanelOpen, previewFile]);
  const previewRefreshRef = useRef<ReturnType<typeof createPreviewRefreshScheduler> | null>(null);
  if (previewRefreshRef.current === null) {
    previewRefreshRef.current = createPreviewRefreshScheduler({
      isOpen: (path) => {
        const key = normalizePreviewPath(path);
        if (!key) return false;
        const ctx = previewRefreshCtxRef.current;
        // 主区大预览正开着该文件 → 刷新
        if (ctx.previewFile && normalizePreviewPath(ctx.previewFile) === key) return true;
        // 工作台面板未打开（含专注模式/窄屏）→ pane 文件 tab 不可见，不刷
        if (!ctx.panelOpen) return false;
        const pane = usePaneTabsStore.getState();
        const active = pane.tabs.find((tb) => tb.id === pane.active);
        // 仅活动文件 tab 算「正在看」：后台 tab 未挂载 FilePreview，重开即最新
        return active?.kind === "file" && !!active.path && normalizePreviewPath(active.path) === key;
      },
      refresh: (path) => {
        usePaneTabsStore.getState().requestReload(path);
      },
    });
  }

  // 会话切换：状态机整体复位 + 重建工具状态基线（恢复会话绝不弹预览）；
  // U4 刷新调度器同步清空待刷新计时（新会话文件 tab 全新挂载，无需跟随）。
  useEffect(() => {
    previewAutoRef.current = initialPreviewAutoFrontState;
    previewAutoToolSeenRef.current = new Map();
    previewAutoBaselineRef.current = true;
    previewRefreshRef.current?.cancelAll();
  }, [currentSessionKey]);

  // 工具事件喂入：写类工具卡（状态 running→done/error 跃迁）→ writeDispatch /
  // writeResult。读取事件对状态机是恒 no-op（读不弹），不喂、省一次路径提取。
  useEffect(() => {
    const seen = previewAutoToolSeenRef.current;
    const baseline = previewAutoBaselineRef.current;
    if (baseline) previewAutoBaselineRef.current = false;
    for (const it of state.items) {
      if (it.kind !== "tool") continue;
      const prev = seen.get(it.id);
      if (prev === it.status) continue;
      seen.set(it.id, it.status);
      if (baseline && prev === undefined) continue; // 基线期只登记
      // 进度计划（v4.114 刀5）：agent schedule_apply 成功 → 板块即时回读
      //（绕过 15s 轻扫等待；防抖/未保存守卫在 store 侧，宁守勿冲）。
      if (it.name === "schedule_apply" && it.status === "done") {
        notifyScheduleFileChanged();
        continue;
      }
      if (!OFFICE_WRITE_TOOLS.has(it.name)) continue;
      const paths = extractOfficeWritePaths(it.name, it.args).filter(isOfficeDeliverablePath);
      if (paths.length === 0) continue;
      const path = paths[0]; // 同回合多文件取首个：同文件同回合至多一窗（纯函数层守卫）
      let openedNow = false;
      if (prev === undefined) {
        applyPreviewAutoEvent({ type: "writeDispatch", path });
        if (it.status === "done") {
          openedNow = applyPreviewAutoEvent({ type: "writeResult", path, ok: true });
        } else if (it.status === "error") {
          applyPreviewAutoEvent({ type: "writeResult", path, ok: false });
        }
      } else if (it.status === "done") {
        openedNow = applyPreviewAutoEvent({ type: "writeResult", path, ok: true });
      } else if (it.status === "error" || it.status === "stopped") {
        applyPreviewAutoEvent({ type: "writeResult", path, ok: false });
      }
      // U4 写后预览实时跟随：成功落盘 → 逐路径喂刷新调度器（失败/中断不派：
      // 内容未变或无回执，刷新也是旧内容；打开判定在计时到点进行，已关闭的
      // 文件不会被刷新也不会被弹）。刚被本回执自动置前的路径跳过——挂载即
      // 加载最新，无需二次静默重读。
      if (it.status === "done") {
        for (const p of paths) {
          if (openedNow && normalizePreviewPath(p) === normalizePreviewPath(path)) continue;
          previewRefreshRef.current?.notify(p);
        }
      }
    }
  }, [state.items, applyPreviewAutoEvent]);

  // 终态清理：回合终结（running 熄灭）复位回合级状态；写意图跨回合保持。
  const prevRunningRef = useRef(state.running);
  useEffect(() => {
    if (prevRunningRef.current && !state.running) {
      applyPreviewAutoEvent({ type: "turnEnd" });
    }
    prevRunningRef.current = state.running;
  }, [state.running, applyPreviewAutoEvent]);

  // 关闭优先：预览从有到无（Esc/×/切 pane/专注模式）= 用户手动关闭 → 喂
  // userClose，清除该路径的打开意图；同回合后续读取/回执不再复活浮窗。
  useEffect(() => {
    return usePreviewStore.subscribe((s, prev) => {
      if (prev.previewFile != null && s.previewFile == null) {
        applyPreviewAutoEvent({ type: "userClose", path: prev.previewFile });
      }
    });
  }, [applyPreviewAutoEvent]);
}
