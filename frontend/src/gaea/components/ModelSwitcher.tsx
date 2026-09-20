import { useCallback, useEffect, useMemo, useState } from "react";
import { Modal } from "antd";
import { Check, ChevronsUpDown } from "../icons";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import type { ModelInfo } from "../lib/types";

// ModelSwitcher is the header model picker: the model label becomes a button
// that opens a dropdown listing configured providers. v4.126 刀1 起按引擎展开
// 全部对话模型：本地引擎分组置前（组名带「本地」标）、条目带加载态徽标；
// 换模预估（刀2）按目标模型询问后端，本地引擎非 hot 时确认后再切。
// When allowInherit is true and the selected value is empty, the button shows
// inheritLabel and the dropdown includes an "inherit" option at the top.
export function ModelSwitcher({
  label,
  onPick,
  allowInherit = false,
  inheritLabel = "",
}: {
  label: string;
  onPick: (name: string) => void;
  allowInherit?: boolean;
  inheritLabel?: string;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [models, setModels] = useState<ModelInfo[]>([]);
  // v4.362：读取失败不再伪装「未配置任何模型」——区分失败态（可重试）与真空。
  const [loadFailed, setLoadFailed] = useState(false);
  const [loading, setLoading] = useState(false);

  const fetchModels = useCallback(() => {
    if (!open) return;
    setLoading(true);
    app
      .Models()
      .then((m) => {
        setModels(m);
        setLoadFailed(false);
      })
      .catch(() => {
        setModels([]);
        setLoadFailed(true);
      })
      .finally(() => setLoading(false));
  }, [open]);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  // 分组视图：本地引擎在前（组内保持后端返回序），云端在后。
  // provider 兜底从 ref 解析（Go 恒填，防御 mock/旧数据缺省）。
  const groups = useMemo(() => {
    const withProvider = models.map((m) => ({
      ...m,
      provider: m.provider || (m.ref.includes("/") ? m.ref.slice(0, m.ref.indexOf("/")) : m.ref),
    }));
    const local = withProvider.filter((m) => m.local);
    const cloud = withProvider.filter((m) => !m.local);
    const byProvider = (list: ModelInfo[]) => {
      const out: { provider: string; items: ModelInfo[] }[] = [];
      for (const m of list) {
        const g = out.find((x) => x.provider === m.provider);
        if (g) g.items.push(m);
        else out.push({ provider: m.provider, items: [m] });
      }
      return out;
    };
    return [
      ...byProvider(local).map((g) => ({ ...g, local: true })),
      ...byProvider(cloud).map((g) => ({ ...g, local: false })),
    ];
  }, [models]);

  // 本地引擎换模前先做预估：非 hot（未运行/未加载）时提示预计等待，
  // 用户确认「继续切换」才真正切模型，避免切过去后长时间卡在冷启动/下载。
  const pick = async (name: string) => {
    setOpen(false);
    const slash = name.indexOf("/");
    const engineID = slash > 0 ? name.slice(0, slash) : name;
    const model = slash > 0 ? name.slice(slash + 1) : "";
    const isLocal = models.find((m) => m.ref === name)?.local ?? false;
    if (!isLocal) {
      onPick(name);
      return;
    }
    try {
      const est = await app.ModelSwitchEstimate(engineID, model);
      if (!est || est.status === "hot") {
        onPick(name);
        return;
      }
      const waitText = est.waitSeconds > 0 ? t("model.waitSeconds", { n: est.waitSeconds }) : t("model.takeTime");
      const ok = await new Promise<boolean>((resolve) => {
        Modal.confirm({
          title: t("model.switchTitle"),
          content: t("model.switchAsk", { name, note: est.note || t("model.needColdStart"), wait: waitText }),
          okText: t("model.switchOk"),
          cancelText: t("common.cancel"),
          onOk: () => resolve(true),
          onCancel: () => resolve(false),
        });
      });
      if (ok) onPick(name);
    } catch {
      // 预估失败（如后端暂未实现）：不阻塞用户，照常切换
      onPick(name);
    }
  };

  return (
    <div className="relative inline-flex">
      <button className="flex items-center gap-1 px-1.5 py-0.5 border border-border-soft rounded-lg bg-transparent text-fg-dim text-[12px] font-medium cursor-pointer no-drag hover:text-fg hover:border-fg-faint" onClick={() => setOpen((v) => !v)} title={t("status.switchModel")}>
        <span className="max-w-28 truncate font-mono text-[11px]">{label}</span>
        <ChevronsUpDown size={11} />
      </button>
      {open && (
        <>
          <div className="fixed inset-0 z-10" onClick={() => setOpen(false)} />
          <div className="absolute top-full left-1/2 -translate-x-1/2 mt-1 w-64 max-h-72 overflow-y-auto bg-bg-elev-2 border border-border rounded-lg z-20 p-1" role="listbox" style={{boxShadow: "var(--ds-shadow-dropdown)"}}>
            {loadFailed ? (
              <button
                type="button"
                className="px-3 py-4 text-fg-dim text-xs text-center w-full bg-transparent border-0 cursor-pointer hover:bg-bg-soft"
                onClick={(e) => { e.stopPropagation(); fetchModels() }}
              >
                {loading ? t("common.loading") : t("status.modelsLoadFailedRetry")}
              </button>
            ) : models.length === 0 && <div className="px-3 py-4 text-fg-faint text-xs text-center">{t("status.noModels")}</div>}
            {allowInherit && (
              <button
                role="option"
                aria-selected={!label || label === inheritLabel}
                className={`flex items-center gap-2.5 w-full px-2.5 py-2 bg-transparent border-0 rounded-md text-left cursor-pointer text-fg-dim text-[13px] hover:bg-bg-soft hover:text-fg ${!label || label === inheritLabel ? "text-accent bg-accent-soft font-semibold hover:bg-accent-soft hover:text-accent" : ""}`}
                onClick={() => void pick("")}
              >
                <span className="flex-1 min-w-0 text-left font-medium">{inheritLabel || t("settings.subagentInherit")}</span>
                {(!label || label === inheritLabel) && <Check size={13} className="shrink-0 text-accent" />}
              </button>
            )}
            {groups.map((g) => (
              <div key={g.provider}>
                <div className="px-2.5 pt-1.5 pb-0.5 text-[10px] uppercase tracking-wide text-fg-faint flex items-center gap-1">
                  <span className="font-mono">{g.provider}</span>
                  {g.local && <span className="text-accent normal-case">· {t("model.localTag")}</span>}
                </div>
                {g.items.map((m) => (
                  <button
                    key={m.ref}
                    role="option"
                    aria-selected={m.current}
                    title={m.label && m.label !== m.model ? `${m.label} (${m.model})` : m.ref}
                    className={`flex items-center gap-2 w-full px-2.5 py-1.5 bg-transparent border-0 rounded-md text-left cursor-pointer text-fg-dim text-[12px] hover:bg-bg-soft hover:text-fg ${m.current ? "text-accent bg-accent-soft font-semibold hover:bg-accent-soft hover:text-accent" : ""}`}
                    onClick={() => void pick(m.ref)}
                  >
                    <span className="flex-1 min-w-0 truncate font-medium">{m.label || m.model}</span>
                    {m.status === "running" && (
                      <span className="shrink-0 text-[10px] text-green-500 flex items-center gap-0.5">
                        <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-500" aria-hidden />
                        {t("model.running")}
                      </span>
                    )}
                    {m.current && <Check size={13} className="shrink-0 text-accent" />}
                  </button>
                ))}
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
