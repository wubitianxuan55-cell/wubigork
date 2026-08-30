// useBridgeWatch — 监控 Wails 前后端桥接心跳
// 每 5 秒调用 Meta() 检测桥接存活，断连时通知调用方。
import { useState, useEffect, useRef, useCallback } from "react";
import { app } from "../lib/bridge";
import { usePollingGate } from "../../hooks/usePollingGate";

const PING_INTERVAL_MS = 5000;
const PING_TIMEOUT_MS = 3000;

export interface BridgeWatchState {
  alive: boolean;
  lastCheck: number; // Date.now() of last successful ping
}

export function useBridgeWatch() {
  const [alive, setAlive] = useState(true);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const onReconnectRef = useRef<(() => void) | null>(null);
  // v4.5.2：桥接心跳轮询接入系统级后台轮询门控（页面不可见时暂停 ping；
  // 恢复可见立即补一次检测，断连最多延迟到用户回到窗口时发现）
  const gate = usePollingGate();

  const check = useCallback(async () => {
    try {
      // 超时竞速：3 秒内没响应视为断连
      const result = await Promise.race([
        app.Meta(),
        new Promise<never>((_, reject) =>
          setTimeout(() => reject(new Error("timeout")), PING_TIMEOUT_MS)
        ),
      ]);
      // 响应成功 → 桥接存活
      setAlive((prev) => {
        if (!prev && onReconnectRef.current) {
          onReconnectRef.current(); // 触发重连回调
        }
        return true;
      });
      void result; // 使用返回值避免 unused warning
    } catch {
      setAlive(false);
    }
  }, []);

  useEffect(() => {
    // 启动时立即检查一次；不可见时跳过（恢复可见由 gate 翻转触发 effect 重跑）
    const tick = () => { if (gate) void check() };
    tick();
    timerRef.current = setInterval(tick, PING_INTERVAL_MS);
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [check, gate]);

  const onReconnect = useCallback((fn: () => void) => {
    onReconnectRef.current = fn;
  }, []);

  return { alive, onReconnect } as const;
}
