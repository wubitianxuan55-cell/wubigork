// mock/office/state.ts — office 域会话内走查态（P4 结构刀2 拆分自 mock/office.ts）。
// 走查态（v4.108）：浏览器 mock 从「静态样例」升级为「会话内可变」——
// XlsxSetCell/WriteFile 真实改动内存状态，Preview 回读即所见，支持
// 看板拖拽/导图编辑保存的全链真浏览器走查（不落盘，刷新即复位）。
// 进度计划多工程走查态（mockSchedule*：SCHEDULE_DEFAULT_REL/mockScheduleFiles/
// mockScheduleMeta/mockScheduleCurrent 及 mockScheduleRegister 等）同属会话内存，
// 但 `mockScheduleCurrent` 由多个方法直接重赋值——TS 禁止对导入绑定重赋值
// （TS2632），故进度计划状态与 Schedule* 方法聚合在 schedule.ts，不抽到本文件。
import { MOCK_XLSX_BODY } from "../shared";

export const mockFileBodies: Record<string, string> = {};
export const mockXlsxState = JSON.parse(MOCK_XLSX_BODY) as {
  sheets: { name: string; rows: { ref: string; value: string }[][] }[];
};

// pptx 真编辑刀2 走查态：rel → 每页段落全文（段落粒度 = PptxApplyEdit 的
// target 粒度）。首次访问播种两页四段演示数据，此后 ApplyEdit 真实改内存，
// PptxSlideText 回读即所见（与 XlsxSetCell 同构；不落盘，刷新即复位）。
const mockPptxTexts = new Map<string, { index: number; paragraphs: string[] }[]>();
export function mockPptxSeed(rel: string): { index: number; paragraphs: string[] }[] {
  let slides = mockPptxTexts.get(rel);
  if (!slides) {
    slides = [
      { index: 1, paragraphs: ["季度经营总结", "营收同比增长 12%，成本结构持续优化"] },
      { index: 2, paragraphs: ["下季度计划", "重点推进三件事：拓客、提效、降本"] },
    ];
    mockPptxTexts.set(rel, slides);
  }
  return slides;
}
// 1x1 占位 PNG（与 image mock 同源的最小 PNG dataUrl）。
export const MOCK_PPTX_PAGE_PNG = "data:image/png;base64,iVBORw0KGgo=";