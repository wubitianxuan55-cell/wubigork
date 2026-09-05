// 右侧工作台「UI」Tab：展示模型经 panel:true 围栏投递的会话级 UI 面板。
import { useState } from "react";
import { GenuiBlock } from "../../genui/GenuiBlock";
import { GenuiActionProvider } from "../../genui/GenuiActionContext";
import { genuiPanelKey } from "../../genui/fingerprint";
import { sanitizeSessionKey, useGenuiPanelStore } from "../lib/genuiPanel";
import { getGenuiActionHandler } from "../lib/genuiHost";

export function GenuiPanel({ sessionPath }: { sessionPath?: string }) {
  const sessionKey = sanitizeSessionKey(sessionPath);
  const session = useGenuiPanelStore((s) => s.sessions[sessionKey]);
  const [confirm, setConfirm] = useState(false);
  const clear = useGenuiPanelStore((s) => s.clear);
  const content = session?.content;
  const actionHandler = getGenuiActionHandler();

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex items-center gap-2 border-b border-border-soft px-3 py-2">
        <span className="text-[12px] font-semibold text-fg">UI 面板</span>
        <span className="text-[10.5px] text-fg-faint">模型生成的可交互工作台 · 随回复原地更新</span>
        {content !== undefined && (
          <button
            type="button"
            className={`ml-auto rounded-md px-2 py-0.5 text-[11px] transition-colors ${
              confirm
                ? "bg-err/15 text-err"
                : "text-fg-faint hover:bg-bg-soft hover:text-fg"
            }`}
            onClick={() => {
              if (!confirm) {
                setConfirm(true);
                window.setTimeout(() => setConfirm(false), 3000);
                return;
              }
              clear(sessionKey);
              setConfirm(false);
            }}
          >
            {confirm ? "确认清空？" : "清空"}
          </button>
        )}
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-3">
        {content === undefined ? (
          <div className="flex h-full items-center justify-center text-[12px] text-fg-faint">
            模型在回答里输出 {"{\"panel\":true}"} 的 genui 围栏后，内容会显示在这里
          </div>
        ) : (
          <GenuiActionProvider onAction={actionHandler}>
            <GenuiBlock spec={content} stateKey={genuiPanelKey("office", sessionKey)} />
          </GenuiActionProvider>
        )}
      </div>
    </div>
  );
}
