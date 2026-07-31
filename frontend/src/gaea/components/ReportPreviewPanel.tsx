import { useCallback, useEffect, useState } from "react";
import { FileText, RefreshCw, ExternalLink, FileSpreadsheet, Image } from "lucide-react";
import type { DirEntry } from "../lib/types";
import { app } from "../lib/bridge";

const REPORT_EXTS = [".md", ".docx", ".xlsx", ".csv", ".pdf", ".pptx", ".html", ".txt", ".png", ".svg"];

function getSourceTool(name: string): string {
  const n = name.toLowerCase();
  if (n.includes("survey") || n.includes("调查") || n.includes("初调") || n.includes("详调")) return "survey_report";
  if (n.includes("bid") || n.includes("投标") || n.includes("标书")) return "bid_proposal";
  if (n.includes("imple") || n.includes("实施") || n.includes("施工") || n.includes("方案")) return "imple_plan";
  if (n.includes("cost") || n.includes("成本") || n.includes("测算") || n.includes("费用")) return "cost_estimate";
  if (n.includes("data") || n.includes("检测") || n.includes("数据") || n.includes("分析")) return "data_analysis";
  if (n.includes("chart") || n.includes("图")) return "chart_builder";
  if (n.includes("ppt") || n.includes("演示") || n.includes("汇报")) return "pptx_create";
  if (n.includes("report") || n.includes("报告") || n.includes("汇总") || n.includes("总报告")) return "doc_assemble";
  return "other";
}

function getSourceLabel(source: string): string {
  const map: Record<string, string> = {
    survey_report: "场地调查",
    bid_proposal: "投标文件",
    imple_plan: "实施方案",
    cost_estimate: "成本测算",
    risk_assessment: "风险评估",
    data_analysis: "数据分析",
    chart_builder: "图表生成",
    pptx_create: "PPT制作",
    other: "其他",
  };
  return map[source] ?? source;
}

function getExtIcon(ext: string) {
  if (ext === ".md") return <FileText size={14} className="text-accent" />;
  if (ext === ".docx") return <FileText size={14} className="text-blue-400" />;
  if (ext === ".xlsx" || ext === ".csv") return <FileSpreadsheet size={14} className="text-emerald-400" />;
  if (ext === ".pdf") return <FileText size={14} className="text-red-400" />;
  if (ext === ".pptx") return <FileText size={14} className="text-amber-400" />;
  if (ext === ".html") return <FileText size={14} className="text-cyan-400" />;
  if (ext === ".png" || ext === ".svg") return <Image size={14} className="text-purple-400" />;
  if (ext === ".txt") return <FileText size={14} className="text-fg-faint" />;
  return <FileText size={14} className="text-fg-faint" />;
}

export function ReportPreviewPanel(_props: { cwd?: string }) {
  const [reports, setReports] = useState<{ name: string; ext: string }[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const fetchReports = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const entries: DirEntry[] = await app.ListDir(".");
      const files = entries
        .filter((e) => {
          const idx = e.name.lastIndexOf(".");
          if (idx < 0) return false;
          return REPORT_EXTS.includes(e.name.slice(idx).toLowerCase());
        })
        .map((e) => ({
          name: e.name,
          ext: e.name.slice(e.name.lastIndexOf(".")).toLowerCase(),
        }))
        .sort((a, b) => a.name.localeCompare(b.name));
      setReports(files);
    } catch (err) {
      setError("无法加载文件列表: " + (err instanceof Error ? err.message : String(err)));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchReports();
  }, [fetchReports]);

  const handleOpen = useCallback(async (name: string) => {
    try {
      await app.OpenWorkspacePath(name);
    } catch {
      // fallback
    }
  }, []);

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 py-2 border-b border-border-soft">
        <span className="text-xs font-semibold text-fg-dim flex items-center gap-1.5">
          <FileText size={13} />
          报告文件
          {reports.length > 0 && (
            <span className="text-fg-faint font-normal">({reports.length})</span>
          )}
        </span>
        <button
          className="toolbar-btn"
          onClick={() => void fetchReports()}
          disabled={loading}
          title="刷新列表"
        >
          <RefreshCw size={12} className={loading ? "animate-spin" : ""} />
        </button>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto">
        {loading && reports.length === 0 && (
          <div className="empty-state">
            <span className="text-fg-faint">加载中…</span>
          </div>
        )}

        {error && (
          <div className="mx-3 mt-3 p-2 rounded-lg bg-del-bg text-err text-xs">
            {error}
          </div>
        )}

        {!loading && !error && reports.length === 0 && (
          <div className="empty-state">
            <div className="empty-state__icon">
              <FileText size={28} />
            </div>
            <div className="text-fg-faint mb-1">尚未生成报告</div>
            <div className="text-[11px] text-fg-faint/70 max-w-[200px]">
              尝试使用 / 命令或直接描述需求来生成土壤修复工程报告
            </div>
          </div>
        )}

        {reports.length > 0 && (
          <div className="flex flex-col gap-1 p-2">
            {reports.map((r) => {
              const source = getSourceTool(r.name);
              return (
                <button
                  key={r.name}
                  className="flex items-start gap-3 px-3 py-2.5 rounded-lg bg-bg-soft border border-border-soft text-left font-[inherit] text-fg-dim hover:text-fg hover:bg-bg-elev hover:border-fg-faint transition-all text-[12px] w-full"
                  onClick={() => void handleOpen(r.name)}
                >
                  <span className="shrink-0 mt-0.5">{getExtIcon(r.ext)}</span>
                  <div className="flex-1 min-w-0">
                    <div className="font-medium truncate">{r.name}</div>
                    <div className="flex items-center gap-2 mt-1 text-[10px] text-fg-faint">
                      <span className="px-1.5 py-px rounded-full bg-accent-soft text-accent text-[9px]">
                        {getSourceLabel(source)}
                      </span>
                    </div>
                  </div>
                  <span className="shrink-0 text-fg-faint/50 group-hover:text-fg-faint">
                    <ExternalLink size={12} />
                  </span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
