import { useState, useEffect, useCallback } from "react";
import { Loader2 } from "lucide-react";
import { saveComfyUIConfig, getComfyUIConfig } from "../lib/image";
import type { SectionProps } from "./SettingsShared";

/** ImagegenSection — 绘梦 ComfyUI 配置（仅保存设置，启停在绘梦面板中操作） */
export function ImagegenSection({ busy, apply }: SectionProps) {
  const [model, setModel] = useState("");
  const [comfyURL, setComfyURL] = useState("http://127.0.0.1:8188");
  const [comfyPath, setComfyPath] = useState("");
  const [comfyPythonPath, setComfyPythonPath] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void (async () => {
      try {
        const cfg = await getComfyUIConfig();
        if (cfg.url) setComfyURL(cfg.url);
        if (cfg.model) setModel(cfg.model);
        if (cfg.path) setComfyPath(cfg.path);
        if (cfg.pythonPath) setComfyPythonPath(cfg.pythonPath);
      } catch { /* ignore */ }
      setLoading(false);
    })();
  }, []);

  const handleSave = useCallback(() => {
    apply(async () => {
      await saveComfyUIConfig(comfyURL, model, comfyPath, comfyPythonPath);
    });
  }, [apply, comfyURL, model, comfyPath, comfyPythonPath]);

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-fg-faint text-sm py-4">
        <Loader2 size={14} className="animate-spin" />
        加载中…
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-col gap-3 p-3 rounded-lg border border-border-soft bg-bg-soft">
        {/* 安装路径 */}
        <div className="flex items-center gap-3">
          <label className="text-fg-dim text-[13px] shrink-0 w-16">安装路径</label>
          <input
            className="flex-1 bg-bg border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent"
            placeholder="D:\ComfyUI（main.py 所在目录）"
            value={comfyPath}
            disabled={busy}
            onChange={(e) => setComfyPath(e.target.value)}
            onBlur={handleSave}
          />
        </div>

        {/* Python 路径 */}
        <div className="flex items-center gap-3">
          <label className="text-fg-dim text-[13px] shrink-0 w-16">Python</label>
          <input
            className="flex-1 bg-bg border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent"
            placeholder="留空自动查找（python_embeded\python.exe）"
            value={comfyPythonPath}
            disabled={busy}
            onChange={(e) => setComfyPythonPath(e.target.value)}
            onBlur={handleSave}
          />
        </div>

        {/* URL */}
        <div className="flex items-center gap-3">
          <label className="text-fg-dim text-[13px] shrink-0 w-16">URL</label>
          <input
            className="flex-1 bg-bg border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent"
            placeholder="http://127.0.0.1:8188"
            value={comfyURL}
            disabled={busy}
            onChange={(e) => setComfyURL(e.target.value)}
            onBlur={handleSave}
          />
        </div>

        {/* 模型 */}
        <div className="flex items-center gap-3">
          <label className="text-fg-dim text-[13px] shrink-0 w-16">模型</label>
          <input
            className="flex-1 bg-bg border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent"
            placeholder="flux"
            value={model}
            disabled={busy}
            onChange={(e) => setModel(e.target.value)}
            onBlur={() => model !== "" && handleSave()}
          />
        </div>

        <div className="flex items-center justify-end pt-2 border-t border-border-soft/50">
          <button
            className="flex items-center gap-1 px-3 py-1.5 border border-border-soft rounded-md bg-transparent text-fg-dim text-xs cursor-pointer hover:bg-bg-soft disabled:opacity-50"
            disabled={busy}
            onClick={handleSave}
          >
            保存
          </button>
        </div>
      </div>
    </div>
  );
}
