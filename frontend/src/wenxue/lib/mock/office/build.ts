// mock/office/build.ts — office 域组装入口（P4 结构刀2 拆分自 mock/office.ts）。
// buildOffice 按方法域合并两个分组（methods_office 杂项 / methods_xlsx 编辑）
// 为完整 OfficeMethods；图文板块进度计划/DAG 面已随 2026-11-15 刀0 撤除。
import type { MakeMockState } from "../state";
import type { OfficeMethods } from "./types";
import { officeMethods } from "./methods_office";
import { xlsxMethods } from "./methods_xlsx";

export function buildOffice(_s: MakeMockState): OfficeMethods {
  return { ...officeMethods, ...xlsxMethods } as OfficeMethods;
}
