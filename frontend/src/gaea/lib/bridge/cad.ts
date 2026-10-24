// cad.ts — CadBindings（AppBindings 分域接口之一，Go OfficeB 门面）：
// 办公「CAD 图纸」卡（一期 P0，docs/gaea-cad-tool-master-plan-2026-10.md）。
// 三绑定全只读；「问图纸」复用 RecognizeImage（image.ts）零新面。
// 返回 JSON 字符串（Go 侧以 string 承载），组件层解析——与 VisionRoute 同口径。

export interface CadSheet {
  /** 图框名（图号-图名组合或兜底「图 N」；PDF 为「第 N 页」）。 */
  name: string;
  /** 世界坐标包围盒 {minX, minY, maxX, maxY}；PDF 全 0。 */
  box: [number, number, number, number];
  /** 文字清晰达标率 0~1（图框质量闸：PoC 真图幅 0.77+，误判巨框 <0.1）；PDF 恒 -1。 */
  passRate: number;
  /** 归属该图框的文本条数。 */
  texts: number;
  /** 分幅 PNG 落盘路径（可直喂 RecognizeImage）。 */
  png: string;
}

export interface CadOpenResult {
  kind: "dwg" | "dxf" | "pdf";
  version: string;
  entityCount: number;
  skipped: number;
  textCount: number;
  layers: string[];
  /** 解析警告（天正代理实体/无图框兜底等，前端原样展示）。 */
  warnings: string[];
  file: { name: string; size: number };
  sheets: CadSheet[];
}

export interface CadRasterResult {
  pngPath: string;
  dataUrl: string;
}

export interface CadText {
  text: string;
  layer: string;
  x: number;
  y: number;
  h: number;
  sheet: number;
}

export interface CadTextsResult {
  total: number;
  truncated: boolean;
  texts: CadText[];
}

export interface CadCountItem {
  /** 原始块名（未命名块为内部编码）。 */
  block: string;
  /** 自动命名（邻近文字众数；空=待人工命名）。 */
  name: string;
  /** 命名置信度 0~1（众数占已配文字实例比）。 */
  confidence: number;
  count: number;
  /** 图框序号(1 基)→数量。 */
  perSheet: Record<string, number>;
  /** 首实例坐标（定位用）。 */
  sample: [number, number];
}

export interface CadLayerLen {
  layer: string;
  /** 图纸单位长度（mm 图 1:1 即毫米；几何长度≠施工量，系数归组价口径）。 */
  length: number;
  lines: number;
}

export interface CadCountResult {
  items: CadCountItem[];
  layers: CadLayerLen[];
  warnings: string[];
}

export interface CadBindings {
  // CadOpen 打开 DWG/DXF/PDF：解析摘要 + 全图框分幅 PNG（产物落 .gaea/cad/ 缓存）。
  CadOpen(path: string): Promise<string>;
  // CadRaster 取第 sheet 个图框/页 PNG（1 基；width>0 加宽重渲，PDF 忽略 width）。
  CadRaster(path: string, sheet: number, width: number): Promise<string>;
  // CadTexts 矢量文本全量清单（\U+ 转义已解码；PDF 不支持，报错引导「问图纸」）。
  CadTexts(path: string): Promise<string>;
  // CadCount 通用符号统计（块计数精确 + 邻近文字众数自动命名 + 图框归属 + 层线长）。
  CadCount(path: string): Promise<string>;
}
