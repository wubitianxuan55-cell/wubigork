// 权限管理 hook — V10.19: 统一 ask/auto/yolo 三级权限
// 主题统一：跟随主应用（老栈）darkMode，gaea 不再维护独立主题。
import { useState, useCallback } from "react";
import type { PermLevel } from "../lib/types";
import { app } from "../lib/bridge";

const THINK_TEMPS: Record<string, number> = { fast: 0.1, normal: 0.3, deep: 0.7 };

export function useModeManager(
  // 保留签名兼容旧调用；参数当前未使用（bypass 由权限系统接管）
  _setBypass: (level: string) => void,
  setModel: (name: string) => Promise<void>,
) {
  const [permLevel, setPermLevelState] = useState<PermLevel>("ask");
  const [thinkLevel, setThinkLevel] = useState<"fast" | "normal" | "deep">("normal");
  const [switchingModel, setSwitchingModel] = useState(false);

  const setPermLevel = useCallback((level: PermLevel) => {
    setPermLevelState(level);
    app.SetPermLevel(level).catch(() => {});
  }, []);

  const handleThinkLevelChange = useCallback(async (level: string) => {
    setThinkLevel(level as "fast" | "normal" | "deep");
    const temp = THINK_TEMPS[level] ?? 0.3;
    try {
      const settings = await app.Settings();
      app.SetAgentParams(temp, settings.agent.maxSteps, settings.agent.systemPrompt).catch(() => {});
    } catch {
      app.SetAgentParams(temp, 0, "").catch(() => {});
    }
  }, []);

  const switchModel = useCallback(
    async (name: string) => {
      setSwitchingModel(true);
      await setModel(name);
      setSwitchingModel(false);
    },
    [setModel],
  );

  return { permLevel, setPermLevel, thinkLevel, switchingModel, handleThinkLevelChange, switchModel };
}
