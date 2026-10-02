import { useCallback, useEffect, useState } from "react";
import { app } from "../../lib/bridge";
import type { PriceFetchRecord, PriceSource } from "../../lib/types";

const FREQ_LABEL: Record<number, string> = {
  0: "仅手动",
  6: "每 6 小时",
  24: "每天",
  168: "每周",
};

export function freqLabel(h: number): string {
  return FREQ_LABEL[h] ?? `每 ${h} 小时`;
}

export function timeText(s: string): string {
  if (!s) return "从未抓取";
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return s;
  return d.toLocaleString("zh-CN", { hour12: false });
}

// usePriceSources 价格源数据装载（价格源页 PriceSourcesPanel 与阅览仓库
// PriceSourcesRepository 共用，FE2-06 抽取）：app.PriceSources()（管理面板再
// 加 app.PriceFetches()）→ 8 秒超时兜底 + loadFailed 三态。
// v4.362：加载失败可见化——失败不再伪装成空列表，由容器渲染失败态 + 重试。
// freqLabel/timeText 为两容器漂移的文案工具单源（卡片与抓取记录共用）。
export function usePriceSources(opts?: { withFetches?: boolean }) {
  const withFetches = opts?.withFetches ?? false;
  const [sources, setSources] = useState<PriceSource[]>([]);
  const [fetches, setFetches] = useState<PriceFetchRecord[]>([]);
  const [loading, setLoading] = useState(true);
  // 加载失败可见化：bridge 拒绝（非超时兜底）时置位，容器出「读取失败 + 重试」。
  const [loadFailed, setLoadFailed] = useState(false);

  // 后端调用偶发卡住时兜底：最多等 8 秒，避免「加载中…」永久转圈。
  const withTimeout = useCallback(<T,>(p: Promise<T>, fallback: T): Promise<T> => {
    return Promise.race([p, new Promise<T>((res) => setTimeout(() => res(fallback), 8000))]);
  }, []);

  const load = useCallback(() => {
    setLoading(true);
    Promise.all([
      withTimeout(app.PriceSources(), []),
      // 阅览仓库只读陈列，不取抓取记录（不发起 PriceFetches 调用）。
      withFetches ? withTimeout(app.PriceFetches(), []) : Promise.resolve([] as PriceFetchRecord[]),
    ])
      .then(([s, f]) => {
        setSources(s ?? []);
        setFetches(f ?? []);
        setLoadFailed(false);
      })
      .catch(() => {
        // 加载失败可见化——原（仓库侧）伪装成空列表。
        setSources([]);
        setFetches([]);
        setLoadFailed(true);
      })
      .finally(() => setLoading(false));
  }, [withTimeout, withFetches]);

  useEffect(() => {
    load();
  }, [load]);

  return { sources, fetches, loading, loadFailed, load };
}
