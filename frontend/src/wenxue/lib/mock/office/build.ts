// mock/office/build.ts — office 域组装入口（P4 结构刀2 拆分自 mock/office.ts）。
// buildOffice 按方法域合并三个分组（methods_office 杂项 / methods_xlsx 编辑 /
// schedule 进度计划）为完整 OfficeMethods；方法与返回形状与拆分前逐字节一致，
// mock.ts 的 `import { buildOffice } from "./mock/office"` 无需改动。
import type { MakeMockState } from "../state";
import type { OfficeMethods } from "./types";
import { officeMethods } from "./methods_office";
import { xlsxMethods } from "./methods_xlsx";
import { scheduleMethods } from "./schedule";

export function buildOffice(_s: MakeMockState): OfficeMethods {
  return { ...officeMethods, ...xlsxMethods, ...scheduleMethods } as OfficeMethods;
}