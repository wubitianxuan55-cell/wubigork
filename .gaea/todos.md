# 任务进度

> 最后更新: 2026-11-28 凌晨（AI 反馈批已收口：五笔提交 + 包 gaea-ai-feedback-20261128-093c83c5 装机 PID 53224，未 push；接续单=进度计划/gaea-ai-feedback-20261127-batch.md §9）

| 状态 | 任务 |
|------|------|
| ✅ | AI 反馈批（2026-11-27，20 条）收口：xlsx 体检 bash 提示 + 两包测试补齐；门禁全过（app 1391 例、agent 全量、前端 510/511）；五笔提交 a78c83a7/1b628dc7/0ae43cb3/73739a34/093c83c5；干净树打包装机 PID 53224，包版四件回写 build/bin；**未 push** |
| ⏳ | AI 反馈批真机走查：造价技能新主路（样表上编→用户检查→入库）、载荷走文件、bash 写出 xlsx 缺缓存值提示、session_search 留空列会话、PDF all_pages |
| ⏳ | 测试卫生两条：`modelcenter/SchedulingSection.test.tsx` 切换开关一例偶发（waitFor 超时）；control 包测试写脏夹具 `resumed.jsonl.meta` |
| ⏳ | 反馈 #4：move_file 覆盖正式件留上一版快照；被拦的「改写已有正式件」自动暂存到 .gaea/work |
| ⏳ | 反馈 #5'：proposal_save 回执「全库共 N 份（含录入中）」+ 本方案进度行 |
| ⏳ | 反馈 #11：outline 增 display_title（匹配用 title，入库用 display_title） |
| ⏳ | 反馈 #12：proposal_save「辅助认标题」（带行号的候选标题清单）或 format_convert 标题形态识别 |
| ⏳ | 反馈 #8：cost_project quotas/resources 增 keyword 精确匹配开关 |
| ⏳ | 反馈 #14/#15②：会话交接卡（最小=最近会话带末条结论）；断点恢复块从持久化日志重建 |
| 🗳 | 待拍板：#10 矢量图补图（引 PDF 库依赖）；277 条章节数据恢复；测算样表「按数据扩骨架」填表脚本；样表三处旧文案是否用 gaea 改 |
| ✅ | 落码 12 条（造价线 / bash / 文件闸 / read_file 字节窗口 / session_search / PDF 页数 / xlsx 体检）——已提交装机，见接续单 §9 |
| ✅ | 勘查 docx 结构：正文段落折行情况、图片落点、软换行、表格跨页续张规律 |
| ✅ | 编写 docx→md 重建器：单元格折行重拼、gridSpan/vMerge 展开、空段落剔除、跨页续表合并、图片独占行 |
| ✅ | 原型验证典型表格（合并表头/跨页续表/目录表/长文本表/图配表）逐格比对 docx 与 PDF 原件 |
| ✅ | 全量转换并机检：表格列数一致、无残留锚点、无空首行、图片引用齐全 |
| ✅ | 重算标题清单 outline（1~4 级），丢弃目录表等噪声，与旧 440 节点对齐校验 |
| ✅ | 分批 proposal_save 替换入库同名方案，复核节点数与字数 |
| ✅ | PDF 原件对账（n-gram 命中）+ 补 11 张缺图 + 剔除章节目录表 + 修 15 处文字错误 |
