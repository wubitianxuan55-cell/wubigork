// 右侧工作台「UI」Tab：展示模型经 panel:true 围栏投递的会话级 UI 面板。
import { useState } from "react";
import { GenuiBlock } from "../../genui/GenuiBlock";
import { GenuiActionProvider } from "../../genui/GenuiActionContext";
import { genuiPanelKey } from "../../genui/fingerprint";
import { sanitizeSessionKey, useGenuiPanelStore } from "../lib/genuiPanel";
import { getGenuiActionHandler } from "../lib/genuiHost";
import { Blocks } from "../icons";

export function GenuiPanel({ sessionPath }: { sessionPath?: string }) {
  const sessionKey = sanitizeSessionKey(sessionPath);
  const session = useGenuiPanelStore((s) => s.sessions[sessionKey]);
  const [confirm, setConfirm] = useState(false);
  const clear = useGenuiPanelStore((s) => s.clear);
  const content = session?.content;
  const actionHandler = getGenuiActionHandler();

  return (
    <div className="genui-panel flex h-full min-h-0 flex-col">
      <div className="genui-panel__head flex items-center gap-2 border-b border-border-soft px-3 py-2">
        <span className="genui-panel__mark" aria-hidden>
          <Blocks size={13} />
        </span>
        <span className="flex min-w-0 flex-col">
          <strong className="text-[12px] font-semibold text-fg">交互面板</strong>
          <span className="truncate text-[9.5px] text-fg-faint">随模型回复原地更新</span>
        </span>
        {content !== undefined && (
          <button
            type="button"
            aria-label={confirm ? "确认清空交互面板" : "清空交互面板"}
            className={`genui-panel__clear ml-auto rounded-md px-2 py-1 text-[10px] transition-colors ${
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
          <div className="genui-panel__empty" role="status">
            <span className="genui-panel__empty-mark" aria-hidden>
              <Blocks size={21} />
            </span>
            <strong>还没有交互面板</strong>
            <p>在对话里让模型用面板围栏生成看板、表单或工作卡片，内容会实时出现在这里。</p>
            <code>{'{"panel":true}'}</code>
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
