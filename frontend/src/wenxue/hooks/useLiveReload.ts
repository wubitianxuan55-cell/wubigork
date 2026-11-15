import { useEffect, useRef } from "react";
import { onEvent } from "../lib/bridge";
import type { WireEvent } from "../lib/types";

// 运行中事件驱动的节流刷新间隔：工具结果/用量/文本事件密集，看板随事件
// 流增量刷新（后端每事件已落盘事件日志，折叠读盘即最新）。
const THROTTLE_MS = 1200;

// useLiveReload 在看板组件里订阅 gaea 事件流：
//  - 运行中：随事件节流刷新（最多每 THROTTLE_MS 一次），让轨迹/上下文/
//    Agent 网络在回合进行中「活着」；
//  - turn_done：立即刷新（回合边界，不等待节流窗口）；
//  - running true→false：整轮完成刷新（保留旧行为）。
// 挂载后的首次加载由调用方负责（load 在 mount 时调用一次）。
export function useLiveReload(running: boolean, load: () => void) {
  const runningRef = useRef(running);
  const loadRef = useRef(load);
  loadRef.current = load;
  const lastRef = useRef(0);

  useEffect(() => {
    return onEvent((e: WireEvent) => {
      const now = Date.now();
      if (e.kind === "turn_done") {
        loadRef.current();
        lastRef.current = now;
        return;
      }
      if (!runningRef.current) return;
      if (now - lastRef.current < THROTTLE_MS) return;
      lastRef.current = now;
      loadRef.current();
    });
  }, []);

  useEffect(() => {
    if (runningRef.current && !running) loadRef.current();
    runningRef.current = running;
  }, [running]);
}
