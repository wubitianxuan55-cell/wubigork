// CadPanel.tsx — 右侧工作台「CAD 图纸」卡（v4.524 一期，可选使用）。
// 打开 DWG/DXF/PDF → 图框列表（passRate 质量闸）→ 分幅 PNG 翻页查看 +
// 矢量文本清单 +「问图纸」（分幅 PNG 喂识图管线，结果可发对话流续问）。
// 红线（docs/gaea-cad-tool-master-plan-2026-10.md）：三绑定只读零写入；
// 「问图纸」复用 RecognizeImage 零新绑定；图纸与提取结果不入库。
import { useCallback, useMemo, useRef, useState } from "react";
import { Alert, Button, Input, Spin, Tag } from "antd";
import { useT } from "../../lib/i18n";
import { Layers } from "../../icons";
import { app } from "../../lib/bridge";
import type { CadOpenResult, CadRasterResult, CadTextsResult, CadCountResult, CadText } from "../../lib/bridge/cad";
import { cadRecentList, cadRecentPush, type CadRecent } from "./cadRecent";
// 解析缓存（§7.3）：ToolSheet 关闭卸载本组件，openRes/raster/texts/count 留在
// cadToolCache，重开（最近图纸/重复导入同路径）不重跑解析。
import {
  cadCacheGet,
  cadCachePutOpen,
  cadCachePutRaster,
  cadCachePutTexts,
  cadCachePutCount,
} from "./cadToolCache";
import "./cad-panel.css";

/** 导入扩展名白名单（与 pickFile.ts 的 fail-closed 后置复核同源：宽容模式下
 *  系统对话框过滤可绕过，前端必须复核扩展名）。 */
const CAD_EXTS = ["dwg", "dxf", "pdf"];

/** 问图纸默认加宽重渲宽度（PoC：真实图幅 3000px 首屏够看，6000px 小字可读）。 */
const ASK_WIDTH = 6000;

function extOf(name: string): string {
  const i = name.lastIndexOf(".");
  return i >= 0 ? name.slice(i + 1).toLowerCase() : "";
}

/** CAD 链路失败上报 gaea.log（走 LogFrontendError 统一通道；面板内同时可见，
 *  日志留痕供实测诊断——首版实测曾因错误只走 UI 而零日志痕迹）。 */
function reportCadError(where: string, e: unknown): void {
  const msg = e instanceof Error ? e.message : String(e);
  try {
    void app.LogFrontendError?.(`[CadError] ${where}: ${msg}`)?.catch(() => {});
  } catch {
    /* 日志通道不可用不掩盖原错误 */
  }
}

export function CadPanel({ onAsk, variant = "pane" }: { onAsk?: (text: string) => void; variant?: "pane" | "sheet" }) {
  const t = useT();
  const [file, setFile] = useState<{ path: string; name: string } | null>(null);
  const [openRes, setOpenRes] = useState<CadOpenResult | null>(null);
  const [sheet, setSheet] = useState(1); // 1 基，与 CadOpen 的 sheets 序一致
  const [raster, setRaster] = useState<CadRasterResult | null>(null);
  const [rasterWidth, setRasterWidth] = useState(0); // 当前 raster 的渲染宽度（问图纸红线：喂图须 ASK_WIDTH）
  const [busy, setBusy] = useState(false); // 打开/切图框/重渲共用忙态
  const [error, setError] = useState("");
  const [errorKind, setErrorKind] = useState<"open" | "raster" | "texts" | "count" | "ask">("open");
  const [texts, setTexts] = useState<CadTextsResult | null>(null);
  const [textsBusy, setTextsBusy] = useState(false);
  const [count, setCount] = useState<CadCountResult | null>(null);
  const [countBusy, setCountBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  const [textsCopied, setTextsCopied] = useState(false);
  const [recent, setRecent] = useState<CadRecent[]>(cadRecentList);
  const [question, setQuestion] = useState("");
  const [askBusy, setAskBusy] = useState(false);
  const [answer, setAnswer] = useState("");

  // 异步结果只允许回填同一「图纸→图框」代际。所有路径/图框切换先同步递增
  // generation，再等 React 状态提交；迟到的 Open/Raster/Texts/Count/Ask 结果
  // 一律丢弃，绝不把 A 图数据挂到 B 图。
  const generationRef = useRef(0);
  const activePathRef = useRef("");
  const activeSheetRef = useRef(1);
  const busyRef = useRef(false);
  const askBusyRef = useRef(false);
  const countBusyRef = useRef(false);
  const textsBusyRef = useRef(false);
  const seqRef = useRef({ raster: 0, texts: 0, count: 0, ask: 0 });

  const fetchRaster = useCallback(async (path: string, idx: number, width: number) => {
    const generation = generationRef.current;
    const seq = ++seqRef.current.raster;
    busyRef.current = true;
    setBusy(true);
    setError("");
    try {
      const next: CadRasterResult = JSON.parse(await app.CadRaster(path, idx, width));
      if (
        generation !== generationRef.current ||
        seq !== seqRef.current.raster ||
        path !== activePathRef.current ||
        idx !== activeSheetRef.current
      ) return;
      setRaster(next);
      setRasterWidth(width);
      // 默认宽度渲图入缓存（加宽图不缓存：dataUrl 太大）；迟到的结果不入。
      if (width === 0) cadCachePutRaster(path, idx, next);
    } catch (e) {
      if (
        generation !== generationRef.current ||
        seq !== seqRef.current.raster ||
        path !== activePathRef.current ||
        idx !== activeSheetRef.current
      ) return;
      setRaster(null);
      setRasterWidth(0);
      setErrorKind("raster");
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (generation === generationRef.current && seq === seqRef.current.raster) {
        busyRef.current = false;
        setBusy(false);
      }
    }
  }, []);

  const load = useCallback(
    async (path: string, name: string) => {
      if (busyRef.current || askBusyRef.current) return;
      const generation = ++generationRef.current;
      seqRef.current.raster++;
      seqRef.current.texts++;
      seqRef.current.count++;
      seqRef.current.ask++;
      activePathRef.current = path;
      activeSheetRef.current = 1;
      busyRef.current = true;
      setBusy(true);
      setError("");
      setFile({ path, name });
      setSheet(1);
      setTexts(null);
      setTextsBusy(false);
      textsBusyRef.current = false;
      setAnswer("");
      setQuestion("");
      setRaster(null);
      setRasterWidth(0);
      setCount(null);
      setCountBusy(false);
      countBusyRef.current = false;
      setCopied(false);
      setTextsCopied(false);
      setOpenRes(null);
      // 缓存命中：图纸只读，openRes/raster/texts/count 原样回填，零绑定调用
      // （§7.3 重开不重跑解析——ToolSheet 关开之间不付 CadOpen+分幅渲染的等待）。
      // 代际/路径守卫照常生效（上面已递增代际并指向本 path）。
      const cached = cadCacheGet(path);
      if (cached) {
        busyRef.current = false;
        setBusy(false);
        cadRecentPush(path, name);
        setRecent(cadRecentList());
        setOpenRes(cached.openRes);
        activeSheetRef.current = cached.bestSheet;
        setSheet(cached.bestSheet);
        const hit = cached.rasters.find((r) => r.sheet === cached.bestSheet);
        if (hit) {
          setRaster(hit.res);
          setRasterWidth(0);
        } else {
          void fetchRaster(path, cached.bestSheet, 0);
        }
        if (cached.texts) setTexts(cached.texts);
        if (cached.count) setCount(cached.count);
        return;
      }
      try {
        const res: CadOpenResult = JSON.parse(await app.CadOpen(path));
        if (generation !== generationRef.current || path !== activePathRef.current) return;
        setOpenRes(res);
        // 最近图纸（内存态单源）：打开成功才入列，关卡再开仍可一键重开。
        cadRecentPush(path, name);
        setRecent(cadRecentList());
        // 首选质量最好的图框（passRate 是 PoC 实测的图框质量闸：真图幅 0.77+，
        // 误判巨框 <0.1；PDF 恒 -1 取第 1 页）。
        let best = 1;
        let bestRate = -2;
        res.sheets.forEach((s, i) => {
          if (s.passRate > bestRate) {
            bestRate = s.passRate;
            best = i + 1;
          }
        });
        activeSheetRef.current = best;
        setSheet(best);
        cadCachePutOpen(path, res, best);
        await fetchRaster(path, best, 0);
      } catch (e) {
        if (generation !== generationRef.current || path !== activePathRef.current) return;
        reportCadError("CadOpen", e);
        setErrorKind("open");
        setError(e instanceof Error ? e.message : String(e));
        setFile(null);
        activePathRef.current = "";
      } finally {
        if (generation === generationRef.current) {
          busyRef.current = false;
          setBusy(false);
        }
      }
    },
    [fetchRaster],
  );

  const import_ = useCallback(async () => {
    if (busyRef.current || askBusyRef.current) return;
    setError("");
    try {
      const picked = await app.PickFiles(CAD_EXTS.join(","));
      const f = picked?.[0];
      if (!f) return;
      if (!CAD_EXTS.includes(extOf(f.name))) {
        setErrorKind("open");
        setError(`仅支持 ${CAD_EXTS.map((e) => `.${e}`).join("/")} 文件`);
        return;
      }
      await load(f.path, f.name);
    } catch (e) {
      reportCadError("PickFiles", e);
      setErrorKind("open");
      setError(e instanceof Error ? e.message : String(e));
    }
  }, [load]);

  const selectSheet = useCallback(
    (idx: number) => {
      if (!file || idx === activeSheetRef.current || askBusyRef.current) return;
      generationRef.current++;
      seqRef.current.raster++;
      seqRef.current.texts++;
      seqRef.current.count++;
      seqRef.current.ask++;
      activeSheetRef.current = idx;
      setSheet(idx);
      setAnswer("");
      setRaster(null);
      setRasterWidth(0);
      setCountBusy(false);
      countBusyRef.current = false;
      setTextsBusy(false);
      textsBusyRef.current = false;
      setError("");
      void fetchRaster(file.path, idx, 0);
    },
    [file, fetchRaster],
  );

  const loadTexts = useCallback(async () => {
    if (!file || texts || textsBusyRef.current) return;
    const generation = generationRef.current;
    const path = file.path;
    const idx = activeSheetRef.current;
    const seq = ++seqRef.current.texts;
    textsBusyRef.current = true;
    setTextsBusy(true);
    try {
      const next: CadTextsResult = JSON.parse(await app.CadTexts(path));
      if (
        generation !== generationRef.current ||
        seq !== seqRef.current.texts ||
        path !== activePathRef.current ||
        idx !== activeSheetRef.current
      ) return;
      setTexts(next);
      cadCachePutTexts(path, next);
    } catch (e) {
      if (
        generation !== generationRef.current ||
        seq !== seqRef.current.texts ||
        path !== activePathRef.current ||
        idx !== activeSheetRef.current
      ) return;
      setErrorKind("texts");
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (generation === generationRef.current && seq === seqRef.current.texts) {
        textsBusyRef.current = false;
        setTextsBusy(false);
      }
    }
  }, [file, texts]);

  const loadCount = useCallback(async () => {
    if (!file || countBusyRef.current) return;
    const generation = generationRef.current;
    const path = file.path;
    const idx = activeSheetRef.current;
    const seq = ++seqRef.current.count;
    countBusyRef.current = true;
    setCountBusy(true);
    setCopied(false);
    try {
      const next: CadCountResult = JSON.parse(await app.CadCount(path));
      if (
        generation !== generationRef.current ||
        seq !== seqRef.current.count ||
        path !== activePathRef.current ||
        idx !== activeSheetRef.current
      ) return;
      setCount(next);
      cadCachePutCount(path, next);
    } catch (e) {
      if (
        generation !== generationRef.current ||
        seq !== seqRef.current.count ||
        path !== activePathRef.current ||
        idx !== activeSheetRef.current
      ) return;
      reportCadError("CadCount", e);
      setErrorKind("count");
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (generation === generationRef.current && seq === seqRef.current.count) {
        countBusyRef.current = false;
        setCountBusy(false);
      }
    }
  }, [file]);

  /** 分布紧凑显示：图框序号(名称)→数量；无图框归属显示「图外」。 */
  const sheetLabel = useCallback(
    (idx: string) => {
      const n = openRes?.sheets[Number(idx) - 1]?.name;
      return n ? `${idx}:${n.length > 8 ? n.slice(0, 8) : n}` : `#${idx}`;
    },
    [openRes],
  );

  const copyCount = useCallback(async () => {
    if (!count) return;
    const head = ["名称", "块", "数量", "分布", "置信度"].join("\t");
    const rows = count.items.map((it) =>
      [it.name || it.block, it.block, String(it.count),
        Object.entries(it.perSheet).map(([k, v]) => `${sheetLabel(k)}×${v}`).join(" ") || "图外",
        it.confidence > 0 ? it.confidence.toFixed(2) : "-"].join("\t"),
    );
    const layers = ["", "图层线长", "图层", "长度", "实体数"].join("\t") + "\n" +
      count.layers.map((l) => ["", l.layer, String(Math.round(l.length)), String(l.lines)].join("\t")).join("\n");
    try {
      await navigator.clipboard.writeText([head, ...rows, layers].join("\n"));
      setCopied(true);
    } catch {
      setCopied(false);
    }
  }, [count, sheetLabel]);

  /** 交 AI 算量（用户拍板：gaea 提供 CAD 识图工具，算量表交给 AI 模型）：
   *  把识别数据打包成一条消息发进对话流，AI 按 cad-takeoff 技能出算量表
   *  与组价（复用既有造价测算能力）。无 onAsk 通道时降级复制到剪贴板。 */
  const handoffToAI = useCallback(async () => {
    if (!file || !openRes || countBusyRef.current) return;
    const generation = generationRef.current;
    const path = file.path;
    const current = () =>
      generation === generationRef.current &&
      path === activePathRef.current;
    let c = count;
    if (!c) {
      const seq = ++seqRef.current.count;
      countBusyRef.current = true;
      setCountBusy(true);
      try {
        const loaded: CadCountResult = JSON.parse(await app.CadCount(path));
        if (!current() || seq !== seqRef.current.count) return;
        setCount(loaded);
        cadCachePutCount(path, loaded);
        c = loaded;
      } catch (e) {
        if (!current() || seq !== seqRef.current.count) return;
        reportCadError("CadCount", e);
        setErrorKind("count");
        setError(e instanceof Error ? e.message : String(e));
        return;
      } finally {
        if (current() && seq === seqRef.current.count) {
          countBusyRef.current = false;
          setCountBusy(false);
        }
      }
    }
    if (!c) return;
    let textsBlock = "";
    try {
      let ts = texts;
      if (!ts) {
        const seq = ++seqRef.current.texts;
        const loaded: CadTextsResult = JSON.parse(await app.CadTexts(path));
        if (!current() || seq !== seqRef.current.texts) return;
        setTexts(loaded);
        cadCachePutTexts(path, loaded);
        ts = loaded;
      }
      const shown = ts.texts.slice(0, 80).map((x: CadText) => `[${x.layer}] ${x.text}`).join("\n");
      textsBlock = `\n## 图上文字（共 ${ts.total} 条，取前 80；全量可再取）\n${shown}\n`;
    } catch {
      // 文本清单失败不阻塞交接（符号统计是算量主体）
    }
    const sheetLines = openRes.sheets.map((s, i) => `${i + 1}. ${s.name}（清晰度 ${s.passRate >= 0 ? s.passRate.toFixed(2) : "—"}）`).join("\n");
    const itemLines = c.items
      .map((it) => `${it.name || "（待命名）"}|${it.block}|${it.count}|${it.confidence > 0 ? it.confidence.toFixed(2) : "-"}|` +
        Object.entries(it.perSheet).map(([k, v]) => `${k}×${v}`).join(" "))
      .join("\n");
    const layerLines = c.layers.map((l) => `${l.layer}|${Math.round(l.length)}`).join("\n");
    const msg =
      `【CAD 图纸算量】${file.name}（${openRes.kind.toUpperCase()}，${openRes.sheets.length} 张图）\n` +
      `请按 cad-takeoff 技能出算量表与组价：先列待确认清单，再出算量表（数量只来自下方工具数据），` +
      `组价匹配定额/清单宁缺勿误。\n\n` +
      `## 图纸清单\n${sheetLines}\n\n` +
      `## 符号统计（块引用计数=精确数量；名称=邻近文字众数自动命名，置信度随行；「待命名」行需先确认）\n` +
      `名称|块|数量|置信度|分布\n${itemLines}\n\n` +
      `## 图层线长（几何长度 mm，施工系数需你按惯例给并明示）\n图层|长度\n${layerLines}\n` +
      textsBlock;
    if (!current()) return;
    if (onAsk) {
      onAsk(msg);
    } else {
      try {
        await navigator.clipboard.writeText(msg);
        setCopied(true);
      } catch {
        /* 剪贴板不可用则静默（面板已有复制表格路径） */
      }
    }
  }, [file, openRes, count, texts, onAsk]);

  /** 文本清单全文复制（TSV：图框序号/图层/内容；交 AI 或贴表格用——
   *  面板只展示当前图框前 100 条，剪贴板给全量）。 */
  const copyTexts = useCallback(async () => {
    if (!texts || texts.texts.length === 0) return;
    const lines = [
      ["图框", "图层", "内容"].join("\t"),
      ...texts.texts.map((x) => [String(x.sheet), x.layer, x.text].join("\t")),
    ];
    try {
      await navigator.clipboard.writeText(lines.join("\n"));
      setTextsCopied(true);
    } catch {
      setTextsCopied(false);
    }
  }, [texts]);

  const ask = useCallback(async () => {
    if (!file || !question.trim() || askBusyRef.current) return;
    const generation = generationRef.current;
    const path = file.path;
    const idx = activeSheetRef.current;
    const seq = ++seqRef.current.ask;
    const current = () =>
      generation === generationRef.current &&
      seq === seqRef.current.ask &&
      path === activePathRef.current &&
      idx === activeSheetRef.current;
    askBusyRef.current = true;
    setAskBusy(true);
    setAnswer("");
    setError("");
    try {
      // 问图纸喂图红线（PoC §五）：按 ASK_WIDTH 加宽重渲（小字可读）再喂识图；
      // 已是加宽图时直接复用，不重复渲染。
      let pngPath = raster && rasterWidth === ASK_WIDTH ? raster.pngPath : "";
      if (!pngPath) {
        const r: CadRasterResult = JSON.parse(await app.CadRaster(path, idx, ASK_WIDTH));
        if (!current()) return;
        setRaster(r);
        setRasterWidth(ASK_WIDTH);
        pngPath = r.pngPath;
      }
      const prompt =
        `请依据这张 CAD 图纸的渲染图回答问题。问题：${question.trim()}。` +
        "要求：图上文字按原文引用；数值与结论说明图中依据；图中没有依据时明确说明，不要猜测。";
      const reply = await app.RecognizeImage(pngPath, prompt);
      if (!current()) return;
      setAnswer(reply.trim());
    } catch (e) {
      if (!current()) return;
      reportCadError("Ask", e);
      setErrorKind("ask");
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      if (generation === generationRef.current && seq === seqRef.current.ask) {
        askBusyRef.current = false;
        setAskBusy(false);
      }
    }
  }, [file, question, raster, rasterWidth]);

  const sendToChat = useCallback(() => {
    if (!onAsk || !file || !answer) return;
    const sheetName = openRes?.sheets[sheet - 1]?.name ?? "";
    onAsk(`【${t("cad.askPrefix")}：${file.name}${sheetName ? `｜${sheetName}` : ""}】\n问：${question.trim()}\n答：${answer}`);
  }, [onAsk, file, answer, openRes, sheet, question, t]);

  const sheetTexts = useMemo(() => {
    if (!texts) return [];
    return texts.texts.filter((x) => x.sheet === sheet || x.sheet === 0);
  }, [texts, sheet]);

  return (
    <div
      data-testid="cad-panel"
      data-variant={variant}
      className={`cad-panel ${variant === "sheet" ? "cad-panel--sheet" : "cad-panel--pane"} h-full min-h-0 gap-2 p-2 text-[12px]`}
    >
      <div className="cad-head">
        <div className="cad-import-row flex items-center gap-2">
          {(variant === "pane" || file) && (
            <Button
              size="small"
              data-testid="cad-import"
              disabled={busy || askBusy}
              onClick={() => void import_()}
            >
              {t("cad.import")}
            </Button>
          )}
          {file && <span className="truncate text-fg-dim" title={file.path}>{file.name}</span>}
        </div>
        {(variant === "pane" || file) && <div className="cad-hint text-[11px] text-fg-dim">{t("cad.importHint")}</div>}
        {recent.length > 0 && (
          <div className="flex flex-wrap items-center gap-1" data-testid="cad-recent">
            <span className="text-[11px] text-fg-dim">{t("cad.recent")}</span>
            {recent.map((r) => (
              <Button
                key={r.path}
                size="small"
                type={r.path === file?.path ? "primary" : "default"}
                data-testid={`cad-recent-${r.name}`}
                title={r.path}
                disabled={busy || askBusy}
                onClick={() => void load(r.path, r.name)}
              >
                {r.name}
              </Button>
            ))}
          </div>
        )}
      </div>

      <div className="cad-alerts">
        {busy && <Spin size="small" />}
        {error && (
          <Alert
            type="error"
            showIcon
            message={`${t(`cad.error.${errorKind}`)}：${error}`}
            data-testid="cad-error"
          />
        )}
        {openRes?.warnings.map((w, i) => (
          <Alert key={i} type="warning" showIcon message={w} />
        ))}
      </div>

      {!file && !busy && (
        <div className="cad-empty">
          <div className="cad-empty-mark"><Layers size={20} /></div>
          <b>打开一张图纸</b>
          <p>{t("cad.emptyHint")}</p>
          <Button type="primary" size="small" onClick={() => void import_()}>{t("cad.import")}</Button>
          <span>支持 DWG / DXF / PDF · 解析与分幅全程本地</span>
        </div>
      )}

      {openRes && (
        <>
          <div className="cad-side">
            <div className="text-[11px] text-fg-dim flex flex-wrap gap-x-2">
              <Tag>{openRes.kind.toUpperCase()}</Tag>
              <span>{openRes.entityCount} 实体</span>
              <span>{openRes.textCount} 文本</span>
              <span title={openRes.layers.join("、")}>{t("cad.layers", { count: openRes.layers.length })}</span>
            </div>

            <div className="flex flex-wrap gap-1" data-testid="cad-sheets">
              {openRes.sheets.map((s, i) => (
                <Button
                  key={i}
                  size="small"
                  type={i + 1 === sheet ? "primary" : "default"}
                  data-testid={`cad-sheet-${i + 1}`}
                  title={s.passRate >= 0 ? t("cad.sheetQuality", { rate: s.passRate.toFixed(2) }) : s.name}
                  disabled={askBusy}
                  onClick={() => selectSheet(i + 1)}
                >
                  {s.name || `#${i + 1}`}
                  {s.passRate >= 0.3 && s.passRate < 0.5 ? "!" : ""}
                </Button>
              ))}
            </div>
          </div>

          <div className="cad-main">
            {raster && (
              <div className="cad-canvas">
                <div className="flex justify-between items-center">
                  <span className="text-[11px] text-fg-dim">
                    {openRes.sheets[sheet - 1]?.passRate >= 0
                      ? t("cad.sheetQuality", { rate: (openRes.sheets[sheet - 1].passRate ?? 0).toFixed(2) })
                      : ""}
                  </span>
                  <a
                    data-testid="cad-rerender"
                    className="text-[11px] cursor-pointer"
                    aria-disabled={askBusy}
                    onClick={() => !askBusy && file && void fetchRaster(file.path, sheet, ASK_WIDTH)}
                  >
                    {t("cad.rerender")}
                  </a>
                </div>
                <img
                  data-testid="cad-raster-img"
                  src={raster.dataUrl}
                  alt={openRes.sheets[sheet - 1]?.name ?? ""}
                  className="cad-raster max-h-[38vh] object-contain border border-border-soft rounded bg-white"
                />
              </div>
            )}

            <div className="cad-ask flex flex-col gap-1">
              <div className="flex gap-1">
                <Input
                  size="small"
                  data-testid="cad-ask-input"
                  placeholder={t("cad.askPlaceholder") ?? ""}
                  value={question}
                  onChange={(e) => setQuestion(e.target.value)}
                  onPressEnter={() => void ask()}
                  disabled={!file}
                />
                <Button
                  size="small"
                  type="primary"
                  data-testid="cad-ask-btn"
                  loading={askBusy}
                  disabled={!file || !question.trim()}
                  onClick={() => void ask()}
                >
                  {t("cad.ask")}
                </Button>
              </div>
              {answer && (
                <Alert
                  data-testid="cad-answer"
                  type="info"
                  message={
                    <div className="flex flex-col gap-1 whitespace-pre-wrap">
                      <b>{t("cad.askAnswer")}</b>
                      <div>{answer}</div>
                      {onAsk && (
                        <a data-testid="cad-ask-send" className="cursor-pointer" onClick={sendToChat}>
                          {t("cad.askSend")}
                        </a>
                      )}
                    </div>
                  }
                />
              )}
            </div>
          </div>

          <div className="cad-inspector">
            <div className="cad-section flex flex-col gap-1 min-h-0">
              {texts ? (
                <div className="min-h-0 overflow-auto" data-testid="cad-texts">
                  <div className="flex justify-between items-center mb-1">
                    <span className="text-[11px] text-fg-dim">
                      {t("cad.texts", { count: texts.total })}
                      {texts.truncated ? ` · ${t("cad.textsTruncated", { count: texts.texts.length, total: texts.total })}` : ""}
                    </span>
                    <a data-testid="cad-texts-copy" className="text-[11px] cursor-pointer" onClick={() => void copyTexts()}>
                      {textsCopied ? t("cad.countCopied") : t("cad.textsCopy")}
                    </a>
                  </div>
                  {sheetTexts.slice(0, 100).map((x, i) => (
                    <div key={i} className="flex gap-2 leading-5">
                      <span className="text-fg-dim shrink-0 max-w-[7em] truncate" title={x.layer}>
                        {x.layer || "-"}
                      </span>
                      <span className="break-all">{x.text}</span>
                    </div>
                  ))}
                </div>
              ) : (
                <Button
                  size="small"
                  loading={textsBusy}
                  disabled={!file || openRes.kind === "pdf" || askBusy}
                  onClick={() => void loadTexts()}
                >
                  {t("cad.texts", { count: openRes.textCount })}
                </Button>
              )}
            </div>

            <div className="cad-section flex flex-col gap-1 min-h-0">
              {count ? (
                <div className="min-h-0 overflow-auto" data-testid="cad-count">
                  <div className="flex justify-between items-center mb-1">
                    <span className="text-[11px] text-fg-dim">
                      {t("cad.count")} · {count.items.length}
                    </span>
                    <span className="flex gap-2">
                      <a data-testid="cad-handoff" className="text-[11px] cursor-pointer" onClick={() => void handoffToAI()}>
                        {t("cad.handoff")}
                      </a>
                      <a data-testid="cad-count-copy" className="text-[11px] cursor-pointer" onClick={() => void copyCount()}>
                        {copied ? t("cad.countCopied") : t("cad.countCopy")}
                      </a>
                    </span>
                  </div>
                  {count.items.slice(0, 60).map((it, i) => (
                    <div key={i} className="flex gap-2 leading-5" data-testid={`cad-count-row-${i}`}>
                      <span className="shrink-0 min-w-[10em] truncate" title={it.block}>
                        {it.name || <span className="text-fg-dim">{t("cad.countUnnamed")}</span>}
                      </span>
                      <span className="text-fg-dim shrink-0 max-w-[9em] truncate" title={it.block}>
                        {it.block}
                      </span>
                      <b className="shrink-0">×{it.count}</b>
                      <span className="text-fg-dim truncate">
                        {Object.entries(it.perSheet).map(([k, v]) => `${sheetLabel(k)}×${v}`).join(" ")}
                      </span>
                    </div>
                  ))}
                  {count.items.some((it) => !it.name) && (
                    <div className="text-[11px] text-fg-dim mt-1">{t("cad.countNoNameHint")}</div>
                  )}
                  <div className="text-[11px] text-fg-dim mt-1">{t("cad.countLayers")}</div>
                  {count.layers.slice(0, 8).map((l, i) => (
                    <div key={i} className="flex gap-2 leading-5">
                      <span className="text-fg-dim shrink-0 max-w-[9em] truncate" title={l.layer}>
                        {l.layer}
                      </span>
                      <span>{Math.round(l.length).toLocaleString()}</span>
                    </div>
                  ))}
                  {count.warnings.map((w, i) => (
                    <Alert key={`w-${i}`} type="warning" showIcon message={w} className="mt-1" />
                  ))}
                </div>
              ) : (
                <div className="flex gap-1">
                  <Button
                    size="small"
                    data-testid="cad-count-btn"
                    loading={countBusy}
                    disabled={!file || openRes.kind === "pdf" || askBusy}
                    onClick={() => void loadCount()}
                  >
                    {t("cad.count")}
                  </Button>
                  <Button
                    size="small"
                    data-testid="cad-handoff-pre"
                    disabled={!file || openRes.kind === "pdf" || askBusy}
                    onClick={() => void handoffToAI()}
                  >
                    {t("cad.handoff")}
                  </Button>
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

