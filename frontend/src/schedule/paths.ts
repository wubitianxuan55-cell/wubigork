/**
 * schedule/paths.ts — 进度计划缺省工程文件 rel（单源常量，FE7-09）。
 *
 * 历史：同一路径字面量曾在三处各写一份——schedule/gschedSummary.ts
 * （`SCHEDULE_FILE_PATH`）、schedule/store.ts（`DEFAULT_SCHEDULE_PATH`，注释
 * 自陈「此处镜像 … 而不直接转引」）、gaea/lib/changes.ts
 * （`SCHEDULE_DEFAULT_REL`）；第四份在 Go 侧 internal/schedule/project.go 的
 * `DefaultRelPath`（不在前端单源范围）。改一处漏两处即静默漂移：计划卡
 * `isCurrentPlan` 判据、store 水合初值与 agent 变更聚合各自认不同路径。
 *
 * 当年不转引的理由是模块环：gschedSummary 已 import store 的
 * `normalizeProject`，store 再 import gschedSummary 即成环，先加载方求值
 * `const` 时 TDZ 崩溃。本模块**零依赖**（不 import 任何模块），转引它不产生
 * 任何边，环的前提消失 ⇒ 三处消费点统一以 import 取值。
 *
 * 取值口径：正斜杠 rel（与 Go 侧 `filepath.ToSlash` 归一后的 rel 同口径），
 * 须与 internal/schedule/project.go `DefaultRelPath` 保持同字面量（跨语言
 * 漂移由后端 rel 校验 + ScheduleDiffCard 路径比对兜底，见余量登记）。
 */
export const SCHEDULE_DEFAULT_REL = '进度计划/当前计划.gsched.json'
