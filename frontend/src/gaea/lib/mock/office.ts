// mock/office.ts — 办公/文件/任务域聚合入口（T6-10.1 拆分自 lib/mock.ts，
// P4 结构刀2 再分：本文件瘦身为纯 re-export，< 5KB 红线达标）。
// 实现按域拆分至 mock/office/ 目录：
//   types.ts         OfficeMethods 共享类型（OfficeBindings 的 mock 实现契约）
//   state.ts         会话内走查态（mockFileBodies/mockXlsxState/pptx seed）
//   schedule.ts      进度计划域（状态+辅助函数+Schedule* 方法，因 TS2632
//                    导入绑定不可重赋值而聚合同文件，含 window.__mockScheduleFile 钩子）
//   methods_xlsx.ts  OfficeEditText/Docx*/Pptx*/Xlsx* 真编辑方法
//   methods_office.ts 其余办公方法（浏览/搜索/预览/收藏/任务/Git/证据链/备份等）
//   build.ts         buildOffice 组装入口
// 导出名与 mock.ts 需求逐一同（buildOffice）；状态/方法签名/返回形状零改动。
// 检索/索引/测评（SemanticSearch/UnifiedSearch/RetrievalEvalRun/FileIndexRebuild/
// FileSemanticSearch）因行数约束拆分至 retrieval.ts，见该文件头注释。
export { buildOffice } from "./office/build";