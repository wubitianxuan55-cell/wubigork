// mock/cost.ts — 成本库/价格域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
import type { AppBindings } from "../bridge";
import type { CostCategory, PriceFetchRecord, PriceSource } from "../types";
import {
  costCategoriesMock,
  costMock,
  priceFetchMock,
  priceSourcesMock,
  setCostCategoriesMock,
  setPriceFetchMock,
  setPriceSourcesMock,
  taskView,
} from "./shared";
import type { MakeMockState } from "./state";

type CostMethods = Pick<
  AppBindings,
  | "CostList" | "CostSearch" | "CostCategories" | "CostCategorySave" | "CostCategoryDelete"
  | "CostGet" | "CostSave" | "CostDelete"
  | "CostImportPreview" | "CostImportAIParse" | "CostImportVisionPreview" | "CostImportApply"
  | "PriceSources" | "PriceSourceSave" | "PriceSourceDelete"
  | "PriceFetch" | "PriceFetchAll" | "PriceFetches" | "PriceFetchApply" | "PriceFetchIgnore"
  | "PriceHistory" | "CostCompare"
>;

export function buildCost(_s: MakeMockState): CostMethods {
  return {
    async CostList() {
      return costMock;
    },
    async CostSearch(query: string, category: string, status: string) {
      const q = (query ?? "").toLowerCase();
      return costMock.filter((e) => {
        const path = e.categoryPath || e.category || "";
        if (category && category !== "all" && path !== category && !path.startsWith(category + "/")) return false;
        if (status && status !== "all" && e.status !== status) return false;
        if (!q) return true;
        return [e.name, e.title, e.spec, e.source].some((s) => (s ?? "").toLowerCase().includes(q));
      });
    },
    async CostCategories() {
      return costCategoriesMock;
    },
    async CostCategorySave(parentId: number, name: string, sort: number, id: number) {
      if (id > 0) {
        const walk = (nodes: CostCategory[]): CostCategory | null => {
          for (const n of nodes) {
            if (n.id === id) {
              n.name = name;
              n.sort = sort;
              return n;
            }
            const hit = walk(n.children ?? []);
            if (hit) return hit;
          }
          return null;
        };
        walk(costCategoriesMock);
        return id;
      }
      const nextId = Math.max(0, ...costCategoriesMock.flatMap((n) => [n.id, ...(n.children ?? []).map((c) => c.id)])) + 1;
      const node: CostCategory = { id: nextId, parentId, name, sort, count: 0, children: [] };
      if (parentId === 0) {
        costCategoriesMock.push(node);
      } else {
        const walk = (nodes: CostCategory[]): boolean => {
          for (const n of nodes) {
            if (n.id === parentId) {
              n.children = [...(n.children ?? []), node];
              return true;
            }
            if (walk(n.children ?? [])) return true;
          }
          return false;
        };
        walk(costCategoriesMock);
      }
      return nextId;
    },
    async CostCategoryDelete(id: number) {
      const walk = (nodes: CostCategory[]): CostCategory[] =>
        nodes.filter((n) => {
          if (n.id === id) return false;
          n.children = walk(n.children ?? []);
          return true;
        });
      setCostCategoriesMock(walk(costCategoriesMock));
    },
    async CostGet(name: string) {
      const e = costMock.find((c) => c.name === name);
      return e ? { ...e, body: "", createdAt: "" } : null;
    },
    async CostSave() {
      // mock: no-op——浏览器开发环境无持久化成本库（真实实现写 CostLibrary 表）。
    },
    async CostDelete() {
      // mock: no-op——同上，无库可删。
    },
    // ── 成本库导入（mock：对已知文件返回样例候选）──
    async CostImportPreview(path: string) {
      return {
        path,
        fileName: path.split(/[\\/]/).pop() ?? path,
        columns: ["材料名称", "规格型号", "单位", "单价(元)", "供应商"],
        unmapped: ["备注"],
        rows: [
          {
            name: "hp300", title: "HP300 高频液压振动锤", category: "机械", unit: "台班",
            price: 3200, spec: "300kW", source: "XX租赁", status: "现行",
            existingName: "hp300", existingPrice: 3000, matchNote: "将覆盖更新（现价 ¥3,000）",
            raw: "HP300 高频液压振动锤 | 300kW | 台班 | 3200 | XX租赁", skip: false, skipReason: "",
          },
          {
            name: "", title: "P.O 42.5 水泥", category: "材料", unit: "吨",
            price: 480, spec: "", source: "海螺", status: "现行",
            existingName: "", existingPrice: 0, matchNote: "新增",
            raw: "P.O 42.5 水泥 | | 吨 | 480 | 海螺", skip: false, skipReason: "",
          },
        ],
        message: "",
        aiUsed: false,
      };
    },
    async CostImportAIParse(path: string) {
      const pv = await this.CostImportPreview(path);
      pv.aiUsed = true;
      pv.message = "AI 智能解析完成，请核对后确认导入。";
      return pv;
    },
    // ── PDF/图片报价单导入（mock：source=pdf_text 的样例候选）──
    async CostImportVisionPreview(path: string) {
      return {
        path,
        fileName: path.split(/[/]/).pop() ?? path,
        columns: ["材料名称", "规格型号", "单位", "单价(元)", "备注"],
        unmapped: [],
        rows: [
          {
            name: "rebar", title: "热轧光圆钢筋", category: "材料", unit: "t",
            price: 3750, spec: "HPB300 Φ12", source: "供应商报价单.pdf", status: "现行",
            existingName: "rebar", existingPrice: 3000, matchNote: "将覆盖更新（现价 ¥3,000）",
            raw: "热轧光圆钢筋 HPB300 Φ12 | t | 3750", skip: false, skipReason: "",
          },
          {
            name: "", title: "螺纹钢", category: "材料", unit: "t",
            price: 3420, spec: "HRB400 Φ20", source: "供应商报价单.pdf", status: "现行",
            existingName: "", existingPrice: 0, matchNote: "新增",
            raw: "螺纹钢 HRB400 Φ20 | t | 3420", skip: false, skipReason: "",
          },
        ],
        message: "PDF 文字提取解析完成，请核对后确认导入。",
        aiUsed: true,
        source: "pdf_text",
      };
    },
    async CostImportApply() {
      return 0;
    },
    // ── 价格源（mock）──
    async PriceSources() {
      return priceSourcesMock;
    },
    async PriceSourceSave(src: PriceSource) {
      const i = priceSourcesMock.findIndex((s) => s.id === src.id);
      if (i >= 0) priceSourcesMock[i] = src;
      else priceSourcesMock.push(src);
    },
    async PriceSourceDelete(id: string) {
      setPriceSourcesMock(priceSourcesMock.filter((s) => s.id !== id));
    },
    async PriceFetch(id: string) {
      const src = priceSourcesMock.find((s) => s.id === id);
      const rec: PriceFetchRecord = {
        id: "fetch-" + Date.now(), sourceId: id, sourceName: src?.name ?? id,
        url: src?.url ?? "", period: "758", fetchedAt: new Date().toISOString(), status: "pending",
        candidates: priceFetchMock[0]?.candidates ?? [],
      };
      setPriceFetchMock([rec, ...priceFetchMock.filter((f) => f.id !== rec.id)]);
      if (src) src.lastFetchAt = rec.fetchedAt;
      return taskView("price_fetch", "抓取 " + (src?.name ?? id), { count: rec.candidates.length, fetchId: rec.id });
    },
    async PriceFetchAll() {
      const enabled = priceSourcesMock.filter((s) => s.enabled);
      for (const src of enabled) {
        await this.PriceFetch(src.id);
      }
      return taskView("price_fetch_all", "一键抓取全部价格源", { fetched: enabled.length, failed: 0 });
    },
    async PriceFetches() {
      return priceFetchMock;
    },
    async PriceFetchApply(fetchId: string, titles: string[]) {
      const rec = priceFetchMock.find((f) => f.id === fetchId);
      if (rec) rec.status = "applied";
      return titles.length;
    },
    async PriceFetchIgnore(fetchId: string) {
      const rec = priceFetchMock.find((f) => f.id === fetchId);
      if (rec) rec.status = "ignored";
    },
    async PriceHistory(name: string) {
      return [
        {
          name, title: "热轧光圆钢筋", unit: "t", price: 3181, source: "四川造价信息网",
          period: "758", fetchedAt: new Date().toISOString(), note: "价格源更新",
        },
        {
          name, title: "热轧光圆钢筋", unit: "t", price: 3000, source: "手动录入",
          period: "", fetchedAt: "", note: "",
        },
      ];
    },
    // ── 比价（mock：现价/历史/价格源抓取 3 行多源对比；字段对齐 Go CostCompareRow）──
    async CostCompare(name: string) {
      if (!name.trim()) return [];
      const now = new Date().toISOString();
      return [
        {
          source: "成本库", period: "", price: 3000, diffPct: 0,
          fetchedAt: "", kind: "current",
        },
        {
          source: "四川造价信息网", period: "757", price: 2980, diffPct: -0.7,
          fetchedAt: now, kind: "history",
        },
        {
          source: "供应商报价单", period: "758", price: 3750, diffPct: 25,
          fetchedAt: now, kind: "fetch",
        },
      ];
    },
  };
}
