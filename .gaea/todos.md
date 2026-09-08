# 任务进度

> 最后更新: 2026-09-09（状态大清算：本表只留**仍开放**项；已完成历史见 .gaea/AGENTS.md 版本速览与 progress.md，逐版全勾者不再列出。旧表 2026-09-04 条目已全部完成/被取代，清空）

## 当前开放（按域）

| 状态 | 任务 |
|------|------|
| ⬜ | 进度计划·真机池：mspdi 刀4 Project/WPS/斑马走查、2013+ 资源/分配键位（等样本）、list_projects 真机、E1 负数对称（动逆推才收口） |
| ⬜ | 办公·pptx：刀2 真机走查（mock 无 pptx 预览分支）；刀4 修改队列泛化（待使用反馈拍板）；Verifier 通道 B 对 docx_apply 复核口径 |
| ⬜ | 办公·思维导图/多维表：B2 后半（字段类型面板/画廊视图/可选 validate 工具）待拍板（M2 已于 v4.108 落地） |
| ⬜ | 造价·刀路池（docs/gaea-cost-domain-survey-2026-09.md §缺口）：条目匹配键/定额编码 → 组价流式分步+多方案 → 五算与版本快照贯通 → 询价库级异常扫描 → 检索索引主动维护+清单级测评集 → 行业含量对照基线；组价依据区 mock 恒空真机走查挂池 |
| ⬜ | DSH 蒸馏：3c 沙箱浏览器多开、阶段四真实终端、阶段五侧边对话——全部「拍板后/若做」门控 |
| ⬜ | genui 审计唯一开放项：resume 槽位口径统一（明确不做，另行排期） |
| ⬜ | 小说域：GenerationGate 闭环/刀7 续/刀8（v4.77 批次后未推进） |
| ⬜ | 图域：T1+（T0 契约 v4.98 已落地） |

## 收敛计划 W1（2026-09-08 立项，权威= docs/gaea-convergence-plan-2026-09.md）

| 状态 | 任务 |
|------|------|
| ✅ | 1.1 package-lock 入库 + CI npm ci（v4.164.0；npm 11 lockfile 缺漏坑已沉淀） |
| ✅ | 1.2 vitest 抗抖：testTimeout 15s + CI 前端 flaky retry（v4.164.0） |
| ✅ | 1.3 DeliverablesPanel 过期 id 潜伏 bug 复现钉死（v4.164.0：office→gaea，+1 回归锁） |
| ✅ | 1.4 WebView2 壳内残留面扫描 → docs/webview2-shell-audit-2026-09.md（P0×2/P1×7/刀序A-D；修复刀A-C 另立版本，刀D 真机取证） |
| ✅ | 拍板：性质路线=A 终极个人工具（产品化=期权）；LICENSE=私有 All Rights Reserved（v4.165.0 落档） |

## 收敛计划 W2+（下一批）

| 状态 | 任务 |
|------|------|
| 🔄 | 审计刀A（P0×2）：基线台账 CSV+角色库参考图壳内修复（v4.165.0 ✅） |
| ✅ | 审计刀B（下载类×4）：ImageGen 两处下载 / NovelSetting 导入导出 / 办公 md 分支（瘦身 P1 v4.166.0） |
| ✅ | 审计刀C（上传类×3）：ControlPanel / VisionTrial / SkillModal 接 pickFile util（瘦身 P1 v4.166.0） |
| ⬜ | 审计刀D（真机取证）：printSvg iframe print / 拖拽 / 粘贴（Filters 可选参数已代码化完成 = v4.167.0） |
| ⬜ | W2 知识外化：30 分钟上手文档 / progress.md 瘦身归档 / 个人路径清洗 |

## 瘦身长期总规划（2026-09-08 立项，权威= docs/gaea-slim-masterplan-2026-09.md）

| 状态 | 任务 |
|------|------|
| ✅ | P0 基线落档（v4.167.0）：docs/gaea-slim-baseline-2026-09.md——七面四表实测（dist 10.0MB/entry 1192.79kB/exe 46.22MB/npm 直依赖 29）+ 功能等价快照（13 板块×核心动作，knowledge 纠正为后端 D7 被过滤）+ IA 走查（rail/默认空间/首页陈列/双空间并列未表达实证） |
| ✅ | P1 快赢轮子（v4.166.0）：W1 b64 收口（≈20 处含 inShell 合一）+ W2 slug×5（strutil.TitleSlug+legacy golden matrix）+ W3 novel diff（DiffReview 迁 lib/diff LCS）+ 刀B/C + locale 死键（1445 键 0 死）+ 依赖验活（26 依赖逐个统计，codemirror 顶包死重移除、@codemirror/* 四子包显式化） |
| ⬜ | P2 形态（2版）：白名单解耦刀0 ✅（v4.167.0）→schedule 并入办公刀1/2→双空间并列落地（工位/乐园平级切换器+分域导航）→home 空间感知化→壳内走查（待证项：rail code 双入口/后端 nav 子项差异/knowledge 孤儿页） |
| ⬜ | P3 结构（2版）：office 抽核+巨文件首批+bridge 双轨退役启动 |
| ⬜ | P4 性能（1版）：懒初始化+exe strip+MemoryHub chunk+entry 拆解 |
| ⬜ | P5 维持：季度复测+观察池审判（cost 文档化/虚拟滚动/内嵌） |

## 文档整理（2026-09-09）✅

✅ docs/ 全域状态大清算：43 份文档状态行对齐 git log 终态（schedule 两设计「待拍板」→已收官、M2 勘误 v4.108 已落地、dsh-univer/genui/edit-tools/unsloth 补收官头）；research-2026-09-* 八目录归档至 docs/archive/ 并修正全仓引用；go-port 方案归档（未采纳）；docs/README.md 索引重写为终态。
