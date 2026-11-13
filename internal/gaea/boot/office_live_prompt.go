package boot

// office_live_prompt.go — COM 实时协同档的系统提示词分工段（波3.5 preferLive
// 引导回，dsh-office 蒸馏；上游 promptText 常驻段同形态）。仅当本机探测到
// Office/WPS（comhelper.ProbeSuiteAvailable）时由 boot 拼接进执行纪律之后；
// 文案融合上游 promptText 与 redirectText 的引导语义：Word/PPT 优先实时档，
// xlsx 数据分析仍走代码/技能，批量转换走 office_batch。
//
// 注意：SingleModelPrompt 常量有逐字节兼容红线（work 缺省输出被既有测试
// 锚定），本段是 boot 侧追加，不触碰该常量。

// OfficeLivePromptSection 返回实时协同分工段（测试钉死文案用；boot 内部
// 拼接走常量本体）。
func OfficeLivePromptSection() string { return officeLivePromptSection }

const officeLivePromptSection = `## Office 实时协同（本机已装 Office/WPS）
- 本机注册了 office_ 工具族（office_open/office_read/office_edit/office_render/office_save 等）：Word 与 PowerPoint 文档的创建和编辑**优先用 office_open 打开后经 office_edit 修改**——文档在应用窗口里实时成形，格式由 Office 自己排版不走样，用户可以边看边协同；文件正被占用（用户开着）也能改。**不要用 docx/pptx 技能经 run_skill 造文件**：脚本生成的文件用户看不到过程、文件被占用时写不进、公式与版式易走样。
- 确实不适合实时档时再回到技能/脚本路线：批量格式转换用 office_batch；应用打不开的旧格式、无窗口的无人值守交付用技能造文件。
- Excel 数据分析（清洗/透视/聚合大表）仍优先代码与 xlsx 技能；数据要在 Excel 窗口里呈现（图表/格式/透视表/用户边看边改）时才用 office_open，批量清洗可用 xlsx_profile 先看列概况再 xlsx_apply。
- 实时档纪律：编辑前先 office_read（用户可能刚改过，Word 段落传 expect 防错位）；改动尽量合并进一次 office_edit 的 ops；交付前 office_render {overview:true} 总览核对再 office_save；AI 只在自己打开的文档上调 office_close，用户开着的文档不碰。
- 公式：文本里写 $...$ / $$...$$ LaTeX，Word/WPS 会转成原生公式。`
