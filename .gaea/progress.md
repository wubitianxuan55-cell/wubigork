# 任务进度

> 最后更新: 2026-08-16 20:05:00

| 状态 | 任务 |
|------|------|
| ✅ | 读取校正版0816表结构与实际数值 |
| ✅ | 按校正口径对齐备注/断链公式并重算 |
| ✅ | 更新记忆与事实底座为校正后单价和总价 |
| ✅ | 小说：设定页「应用到设定」手动按钮 + 创作面板字号调节 + 默认 story-deslop + 剧情后台构思后弹窗确认 |
| ✅ | 模型中心：修复右侧统计与资源（内存采集/CPU 采样/AMD GPU 显存识别/趋势时间轴/失败可见+重试+30s 刷新） |
| ✅ | 绘梦：模板库重设计（内置 7 类 + herdsman 12 类 = 19 类 231 个，gen-herdsman-templates.mjs 生成）+ ComfyUI WebSocket 实时进度 + 单图铺满画布 |
| ✅ | 角色剧照：远程 URL 本地化（保存+启动迁移）、助手/项目角色同步、前端裂图占位兜底 |
| ✅ | 打包发布 v3.0.3：版本三处统一 3.0.3，Go 全量测试 + 前端 507/509（2 个既有 launcher 基线失败）通过，发布 gaea-v3.0.3.exe + SHA256SUMS + v3.0.3.md + CHANGELOG |
| ✅ | 办公板块「任务目标 → 验收清单」升级：RequirementView 扩展 items/autoPursue + 5 个新绑定；目标卡/待办卡拆分重设计（GoalCard 始终展开 + TodoCard 默认折叠、未完成在前已完成收尾分组）；恢复会话时把目标写入 agent goal gate；Go +2、前端 +11 用例，全量测试零回归 |
| ✅ | 角色库文件交互修复：补齐/随机管线支持五维人格（按性格生成 T/I/S/O/R）；「加入项目」弹窗选择任意小说（新增 CharacterAssociateTo 绑定，校验书架目录）；加入后物化 characters.json + 小说角色面板读取自愈（旧版只写关联表的角色自动补齐可见）；Go 全量测试通过、前端 tsc/eslint 0 errors、vitest 521 通过（2 个既有 launcher 基线失败） |
| ✅ | 小说阅读体验优化：阅读面板铺满主视区（根容器 flex 修复 + 去掉 46rem 限宽）；新增排版面板（字号/行距/版宽，全局持久化，默认铺满）；段落首行缩进 + 两端对齐 + 章节标题；顶部阅读进度条 + 每章滚动位置记忆；←/→ 上一章/下一章快捷键；readingSettings 3 用例，前端 tsc/eslint 0 errors、vitest 524 通过（2 个既有 launcher 基线失败） |
| ✅ | 小说阅读 P0（调研驱动）：阅读主题（跟随/米黄/护眼绿/夜间）+ 亮度滑杆（70-120%，持久化）；书签（图钉面板，当前滚动位置「＋ 此处」带段落摘录，列表跳转/删除，按项目持久化）；自动滚屏（播放/暂停、Aa 面板 5 档速度、滚轮手动暂停、到底自动停）；readingSettings 扩展 + readingBookmarks 3 用例，tsc/eslint 0 errors、vitest 527 通过（2 个既有 launcher 基线失败） |
| ✅ | 小说阅读 P1-划线/高亮/想法：划词浮动工具条（四色高亮 + 想法，单段内选择，滚动收起）；高亮按摘录文本回渲染（冲突跳过、点击编辑/删除）；本章批注面板（颜色点 + 摘录 + 想法标记，点击定位/删除）；想法弹窗（摘录引用 + 多行编辑）；readingAnnotations 3 用例，tsc/eslint 0 errors、vitest 530 通过（2 个既有 launcher 基线失败） |
| ✅ | 小说阅读 P1-AI 伴读：章节顶部「AI 摘要」折叠卡（展开生成 3-5 条要点，按章缓存 + 失败重试，仅本地文本）；划词工具条「问书」（摘录引用 + 提问 + AI 回答弹窗）；后端 NovelReadingAsk（summary/ask，正文 rune 截断 9000，novel 功能路由，内联提示词），novel 门面 67→68，绑定漂移 469 一致；Go +1 守卫用例，tsc/eslint 0 errors、vitest 530 通过（2 个既有 launcher 基线失败） |
| ✅ | 小说阅读 P1-朗读同步 + 全文搜索：阅读页脚 TTS 听书（逐句蓝色高亮 + 平滑滚动居中，停止自动清除）；全文搜索（放大镜弹层，标题+正文、防抖 300ms、一章一命中带 ±40 字片段，点击打开章节阅读模式并黄色定位首个命中）；后端 NovelSearch（大纲顺序扫描、分支章节 ReadChapterBranch、上限 100），novel 门面 68→69，绑定漂移 470 一致；Go +1 守卫用例，tsc/eslint 0 errors、vitest 530 通过（2 个既有 launcher 基线失败） |
| ✅ | 修复：项目上下文跨板块残留——切到绘梦/办公/聊天等板块时，顶栏面包屑和底栏遥测不再显示小说标题/进度/章数/字数；项目上下文仅在项目锚点板块（小说）展示，其他板块顶栏只显示板块名、底栏保留系统遥测；前端 tsc/eslint 0 errors、vitest 530 通过（2 个既有 launcher 基线失败） |
| ✅ | 导出标签页合并进阅读面板：删除小说板块「导出」tab（子导航/类型/懒加载/ExportPage.tsx 移除），导出逻辑抽为 ExportPanel 组件，阅读页脚新增导出按钮 + 弹窗（一键导出 TXT/Markdown/EPUB，结果列表与空态保留）；NOVEL_NAV 与 NovelInspector 导出说明同步清理；tsc/eslint 0 errors、vitest 530 通过（2 个既有 launcher 基线失败） |
| ✅ | 书架优化 + 导入成品小说：移除首卡 hero 双列效果，全部卡片固定等高 288px（封面/信息/操作对齐）；「导入小说」按钮（原生文件选择 TXT/MD/EPUB）→ 弹窗填书名/题材/文风 → 后端 ImportNovelBook 解析章节（第X章/Chapter N/序章/Markdown 标题切分，GBK/GB18030 兜底，EPUB spine 顺序读正文去 HTML）新建项目并自动打开；同名目录自动加序号；Go +4 用例，novel 门面 69→70、绑定漂移 471 一致，tsc/eslint 0 errors、vitest 530 通过（2 个既有 launcher 基线失败） |
| ✅ | 打包发布 v3.0.4：版本三处统一 3.0.4（sync-version + package.json）；CI 全绿（go build/vet/test + 前端 build/vitest/eslint + E 系列守卫修复——ChatComposer/useChatStream/chat utils/ChatB.d.ts/GraphView 陈旧断言）；wails build 发布版 35.18MB；冒烟测试通过（/api/health 200）；发布 gaea-v3.0.4.exe + SHA256SUMS-v3.0.4.txt + v3.0.4.md + CHANGELOG-v3.0.4.txt + README 索引更新（v2.38-v2.40 归档到 releases/archive/） |
| ✅ | 首页任务指挥中心改版（参照 DeepSeek 首页风格）：Hero 左文右卡（公告 pill + 大标题 + 双行动卡「开始创作/和 gaea 对话」黑色描边）+ 右侧深色渐变 AI 视觉卡（细网格纹理 + 星云流光 + 语音晶核呼吸环 + 活跃模型 pill）；AI 状态细条改 4 列透明无边框；「全部模块」分区：办公大卡固定 280px 居左 + 其余 8 卡 4×2 网格（角色库/设置不再单独成行），模块区不被压缩——修复设置卡被底部信息条横框遮挡；PageRegistry 补注册 ProgrammingPage（编程卡恢复可见）；语音晶核 canvas 粒子星云在 WebView2 下 rAF 被节流只剩首帧（静止图片），setTimeout 驱动实测不稳，最终取消粒子星云恢复发光球；tsc 0 errors、vite build 通过 |
| ✅ | 打包发布 v3.0.5：版本统一 3.0.5（sync-version 三处 + package.json）；CI 全绿（go build/vet/test + 前端 build + E 系列守卫）；wails build 发布版 35.2MB；冒烟测试通过（/api/health 200）；发布 gaea-v3.0.5.exe + SHA256SUMS-v3.0.5.txt + v3.0.5.md + CHANGELOG-v3.0.5.txt + README 索引更新（v3.0.0 归档到 releases/archive/）；git 提交 + tag v3.0.5（v3.0.2–v3.0.4 发布资产随本次一并入库） |
