# gaea 项目记忆

> 本文件为项目长期记忆（文档记忆层级）。编码规范：**UTF-8 无 BOM**（历史遗留的 GBK/UTF-8 混合编码已清理）。
> 修改后请保持 UTF-8；.ps1 脚本需 UTF-8 带 BOM（见「沙箱环境备忘」）。

## 版本状态（顶部速览）

> 速览只留最近 3 版（2026-09-15 整段分流：水位根治，此前 14 版口径废止——10 条迁 archive 段首）；更早版本全部见 `docs/archive/agents-version-history-2026-09.md`（v4.49 及更早、v4.50–v4.145 仅 CHANGELOG/releases 有记录）。
- **最新发布：v4.454.0（2026-10-02）「纠偏：主角关系落本书项目角色，通用角色库回退」**——用户纠偏「角色库是通用库，不需要固化关系。我们是在补充增加小说项目的角色关系！」——v4.453.0 落错面（通用库），本版**整体回退库侧**+项目侧重做。**绑定面 744 零净变更**（-charlib 绑定+GenerateProjectProtagonistRelations〔novel〕）。【回退】v4.453 全部库侧改动还原到 v4.452（characterlib 字段/列/store、生成链收编、模板契约、编辑器/筛选栏/Inspector UI、api/桥接、测试；wailsjs 手改还原）。【项目侧】types.Character 增 ProtagonistRelation（protagonist_relation，2-8 字短语，**只落本书 characters.json**）；两路保护=AI 补全库投影链路回带原值+CharacterSyncProject 物化按 ID 回带；新模板 prompts/character-relation.json（角色摘要+主角锚点+已用关系避重+世界观；输出 {"relation"}；must 禁性格/禁剧情走向）+GenerateProjectProtagonistRelations(all/missing/one，无主角指路，protagonist-relation-progress 进度)；**消费**=characterSummaryLine 名册注入「·与主角：X」（非空才渲染，旧数据零变化）。【UI】角色面板头部 Dropdown（全部须确认/剩余全部）+详情抽屉个人钮（主角隐藏）+关系 Tag。【测试】Go 4 新例（矩阵/无主角指路/名册消费/补全+同步保字段）；app/characterlib/types 全包绿；tsc 绿。**坑复训**=mustJSON 与 image_handler_test 重名（辅助函数加前缀）/测试桩三件套（SetFeatureModel("characterlib")+characterAgent=character.New+a.ctx 超时）缺一即假红。**产物**=releases/gaea-v4.454.0.exe 51,918,336B SHA256=`06987474853b295e9bb357ae27a44880c1623126b9532703e2a0602ffb778f62`（冒烟 200 过；删 v4.449.0.exe；速览迁 1 条：451）。**文档**=releases/v4.454.0.md+CHANGELOG/README+releases/README+AGENTS 速览。
- **最新发布：v4.453.0（2026-10-02）「角色库『与主角的关系』：AI 随机生成，范围 全部/剩余全部/个人」**——用户指令「小说角色库增加AI随机生成与主角的关系功能」。**绑定面 743→744（+1：CharacterGenerateProtagonistRelations）**。【数据面】Character.ProtagonistRelation（protagonistRelation，2-8 字关系短语）+ SQLite protagonist_relation 列（migrateSchemaV2 幂等补列，旧行默认空串）+ store 四处 SELECT/upsert/scan。【生成链】字段收编 libFillKeys/libRandomKeys/labels/snakeFallback→编辑器「与主角关系」字段行+↻ 单字段随机免费获得；characterGenerate 注入【与主角关系】段（RoleType=protagonist 卡锚定姓名+性格摘要、库内已用关系避重、2-8 字短语口径；主角本人恒输出空串；无主角卡按世界观虚构兜底）；模板 output 契约+must。【批量绑定】mode=all（除主角与助手，覆盖重写）/missing（只补空白）/one（点名单个）；无主角卡诚实报错指路；主角本人请求友好报错；逐角色 character-fill-progress 进度（与一键补齐同通道）。【UI】筛选栏「AI 主角关系」Dropdown（全部须确认/剩余全部）+Inspector 个人钮+编辑器字段行。【接线七处】门面 gen_bindings 再生噪音按 HEAD 还原+手工补一行/bindingNames 744/spaceBindings play+分类锁 564→565/bridge charlib/api/wailsjs CharlibB+models.ts。【测试】Go 3 新例+前端 1 新例；charlib 包绿、Editor 36/36、spaceBindings 4/4、tsc -b 绿。**坑复训**=gen_bindings -names 只打印不落盘/门面再生噪音按 HEAD 还原+手工补行/测试桩缺 SetFeatureModel("characterlib",…) 撞真实登录态。**产物**=releases/gaea-v4.453.0.exe 51,917,824B SHA256=`2abf6a4caaeaa384b7d1b5b33087a249d73c2bfb67602548d3f0f7f1512f3108`（冒烟 200 过；删 v4.448.0.exe；速览迁 2 条：450/449）。**文档**=releases/v4.453.0.md+CHANGELOG/README+releases/README+AGENTS 速览。**v4.454.0 纠偏：本版库侧改动已整体回退，功能迁至本书项目角色（见 v4.454.0 条）。**
- **最新发布：v4.452.0（2026-10-02）「每 10 章一阶段 + 阶段内起承转合：分支与章节计划共用位置注入」**——用户指令「把分支剧情的阶段改到每10章一个阶段，章节计划的提示词也要像剧情分支一样，不同的时间段用不同的提示词，一个阶段的起承转合要做好」。**绑定面 743 零变更**（QuickBrainstormBranches 签名扩展 +chapterNum）。【阶段节奏】stageLength 20→10 全局一处改（合账提前到第 11/21/… 章、stage-recap.json 文案同口径）；分支/计划/生成合账共用同一 stageLength=一份真相。【阶段模型】stagePhaseOf：第1章全书开篇；11/21/…新阶段开篇；阶段内 2-3=起、4-6=承、7-9=转、10=合（收官）；<=0 未知。【分支教义】branchPositionContext 重写：起=三种开局推进路线（正面主攻/侧翼支线/暗线布局）；承=三种升级形态（赌注/对手/关系）；转=三种反转形态（揭底/倒转/爆雷）；合=收官兑现；每段带阶段区间。【计划教义】chapter-plan.json 增 story_position P0 槽+planPositionContext（各段任务各异+「不透支后段」纪律+must 约束）。【盲区根修】向导主链路此前恒传 0，位置教义在 UI 主流程形同虚设——CreatePage 按 覆盖章>0?覆盖章:prevChapter+1 下发目标章。【测试】Go 4 新/改例+20 节奏用例全改 10；前端调用点断言+1；app 全包绿、CreatePage 38/38、tsc 绿。**坑复训**=heredoc 反斜杠 n 工具层降级（第三次踩，一律 Edit 手写转置）。**产物**=releases/gaea-v4.452.0.exe 51,898,368B SHA256=`1106b61d9b84bd48f14165b96db7a57e5017d7b46211f6315dad589b23da651d`（冒烟 200 过；删 v4.447.0.exe）。**文档**=releases/v4.452.0.md+CHANGELOG/README+releases/README+AGENTS 速览。
- **最新发布：v4.437.0（2026-09-30）「P1 小刀包：对话停止入口 + 数字守卫 + 优化档关账三标注」**——优化方向档 P1-5 对话条+P1-7+三条关账。**绑定面 742→743（+1：ChatStreamCancel）**。**①停止入口**=后端 chatStreamCancels 登记表（**归 core**——裸 chat 测试桩不含 writingState，首放即 nil 嵌入 panic 的坑）+ChatStreamCancel(runID)（已生成部分落库+cancelled 终态=对齐章节生成取消语义；协程退出独占清表）；前端 stop+停止钮（sending 时发送变停止）+cancelled 帧（部分回复+标注不标 error）。**②数字守卫**=check-docs ⑤节自述 vs 实存对账——**首跑抓真账**：AGENTS 66.6KB 超预算→迁最老 6 条（v4.421~428.0）入 archive 回 47.4KB。**③关账**=P1-5 对话 ✅ 本版/小说 ✅（v4.429+430）/绘梦 ✅（v4.415）。**测试**=Go 3 新例（含取消传播受控端到端）+cancel 改坏反向验证；app 整包绿；全量 ci CI OK。**坑**=登记表字段放哪个 struct 要看消费方装配面（chat 域归 core）/端到端取消测试用受控协程（假站路由快速失败=毫秒级清表）/mutation 锚先 grep 再改（python 替换锚未中=假验证）。**产物**=releases/gaea-v4.437.0.exe 51,807,744B SHA256=`3458cc6740eb2500a8e61342d52b37a2df33b2945571a3d7e43d02c1c065bb87`（冒烟 200 过；删 v4.432.0.exe）。**文档**=releases/v4.437.0.md+CHANGELOG/README+releases/README+AGENTS 速览（迁 6 插 1）+progress+优化档三标注。**留池**=GenUI 预算合并（P2-2）/Get*Stats 字段用例尾池/停止真机走查。
- **最新发布：v4.436.0（2026-09-30）「P5 破防专项：Go >50KB 五文件拆分（纯搬移零功能）」**——用户指令「继续」（P5 复测在册挂池清账；七刀后实测五文件破防）。**零功能变更、绑定面 742 零变更**（同包搬移签名不动）：create_chapter_handler 67.4K→43.8K（拆 create_chapter_context.go）/image_comfyui 59.4K→36.1K（拆 image_comfyui_workflows.go）/image_handler 57.9K→39.3K（拆 image_comfyui_proc.go）/controller 53.2K→42.3K（拆 controller_session.go）/config 52.2K→41.7K（拆 config_paths.go）——**破防线归零**（>40KB 仍 7 个留池下季）。四包 build/vet/gofmt/golangci 0 issues+测试绿；随版收口 DigestInjection unlinkat 竞态；全量 ci CI OK（首轮在册 flaky OriginalSinPage.tabs 单跑绿重跑绿）。**坑**=大文件拆分机械配方（顶层声明清单定边界→整段行级搬移→编译器收敛 import→段尾注释回填）/切点选功能族聚拢/后台 shell cd 漂移再犯。**产物**=releases/gaea-v4.436.0.exe 51,804,160B SHA256=`bde9b4652abb9cee52488e4b73181837cf8e1c5b9471460ee6edd999f78754ac`（冒烟 200 过；删 v4.431.0.exe）。**文档**=releases/v4.436.0.md+CHANGELOG/README+releases/README+AGENTS 速览+progress。
- **最新发布：v4.435.0（2026-09-30）「长篇刀7：评测基线（七刀收官）」**——用户指令「继续」（七刀收官 P2）。规格 §4 最小闭环。**绑定面 739→742（+3：NovelEvalSnapshot/BaselineSet/Compare）**。**快照**=零 LLM 确定性聚合（AI 味 mean/max/P90/最差章+S1S2S3+伏笔回收率+结构 findings+文风 Delta+上下文合计+promptSetHash〔prompts sha256，变更即 stale〕）；**基线**=eval/baseline.json+snapshots 历史，body 无时钟稳定序（同状态两次逐字节相同=§4 验收线）；**对比**=逐指标 Δ+语义方向+hash 不一致整份 stale。**测试**=Go 4 新例+方向语义反向验证；**随版收口**=HTTP 断言测试的 Windows unlinkat 竞态（生成协程尾步写盘 vs TempDir 清理——**任何 CreateChapter 真链测试收尾必 waitGensDone**，v4.233 坑完整形态）；app 整包绿；全量 ci CI OK（三轮修 errcheck/unused/unlinkat）。**产物**=releases/gaea-v4.435.0.exe 51,803,648B SHA256=`f0262d5b4f57517512e4ac80e6900b6d1ec485e25a7895ef21570ceb336e4a4f`（冒烟 200 过；删 v4.430.0.exe）。**文档**=releases/v4.435.0.md+规格+CHANGELOG/README+releases/README+AGENTS 速览+progress。**长篇七刀全收官**（v4.422/430/431/432/433/434/435，§1.1~1.14 正面解全落地）。**留池**=回归夹具/盲评 bootstrap/张力聚合/真 A/B/七刀真机走查。
- **最新发布：v4.434.0（2026-09-30）「长篇刀6：上下文编译器」**——用户指令「继续」（长篇七刀次刀序 P2）。规格 §1.8 堆料三缺口收口。**绑定面 738→739（+1：NovelContextInventory）**。**①setting 入预算**=整篇设定超 3000 rune 截断+提示（此前无裁剪直进 P0 槽）。**②文风去重合并**=style.md+刀5 digest 并为单一「文风与表达」区段（偏好优先学习跟随；独立注入位移除）。**③世界观相关性**=命中本章计划/大纲的维度稳定前移（前移≠丢弃）；中文匹配 rune bigram。**④可见性**=Inventory dry-run 清单（区段/长度/合计）。**测试**=Go 4 新例（单标题去重验收+bigram 永假反向验证）+既有断言同步两处；app 整包绿；全量 ci CI OK（两轮修 unused 兼容壳+gofmt）。**坑**=worldview 测试直写 json sections（md 触发 legacy 迁移）/中文分词无效用 bigram/golangci unused 比 vet 严/后台 shell cwd 漂移复发。**产物**=releases/gaea-v4.434.0.exe 51,777,536B SHA256=`c87c5b06e6f88244225837683628d2d50ac1de0d05e14574869cde9353eb574a`（冒烟 200 过；删 v4.429.0.exe）。**文档**=releases/v4.434.0.md+规格+CHANGELOG/README+releases/README+AGENTS 速览+progress。**留池**=语义级检索/SceneBible 统一/Inventory UI。**刀序**=刀7 评测基线（P2，长篇七刀收官）。
- **最新发布：v4.433.0（2026-09-30）「长篇刀5：风格学习回灌」**——用户指令「继续」（长篇七刀次刀序）。规格 §1.9 指纹只打分不回灌的正面解。**绑定面 735→738（+3：NovelStyleDigestBuild/Get/Clear）**。**指令编译**=DigestOf 纯函数（指纹数值→可执行写作指令：句长节奏/段落/对话占比/低密度避免项/标点/词汇/签名词）。**摘要档**=style_digest.json 与指纹体检档独立；Build 复用指纹链（3 章 3000 字门槛）。**注入**=CreateChapter+逐场景流「作者风格约束」区段，零 digest 零注入（HTTP 断言）。**前端**=文风面板回灌区（状态+指令预览+构建/停用）。**测试**=Go 3 新例+永假反向验证+前端 2 新例；小说域 37 文件 327 例；全量 ci CI OK。**随版收口**=刀2 测试 flaky（SSE done 帧断言一次快照有竞态→waitFor 轮询）。**坑**=SSE 事件断言必须轮询/反向验证改阈值测不出单向放松须改永假/后台 shell cwd 漂移先 cd。**产物**=releases/gaea-v4.433.0.exe 51,764,224B SHA256=`b73ee31c530480f409d6b2d13c87688d6bf151756e4728efe1e6373f6b445228`（冒烟 200 过；删 v4.428.1.exe）。**文档**=releases/v4.433.0.md+规格+CHANGELOG/README+releases/README+AGENTS 速览+progress。**留池**=few-shot 样本句/真模型对照实验。**刀序**=刀6 上下文编译器（P2）。
- **最新发布：v4.432.0（2026-09-30）「长篇刀4：质量收敛闭环」**——用户指令「继续」（长篇七刀次刀序）。规格 §1.7 报告不闭环的正面解。**绑定面 733→735（+2：NovelChapterConvergePreview/NovelChapterConverge）**。**判据**=convergeCheck 纯函数（AI 味≤40 且 S1/S2 归零，S3 不阻断）。**闭环**=流式 converge-stream，每轮=轮前单元快照→定向修补（S1/S2 清单+AI 味要点指令整单元重写〔安全闸〕+rewriteUnit 句级）→复检→三态判停：converged 即停（零轮直答）/no-improve 按单元回滚本轮并停/max-rounds 剩余如实报告；每轮版本库留痕 mode=converge 可逐轮恢复；v4 无场景混合态章回落 blob 单元。**前端**=CreatePage rail「收敛修补」ConvergeModal（预览/目标可调/逐轮轨迹/终态如实）。**测试**=Go 6 新例（含 no-improve 回滚字节级+回滚函数直测〔语义改坏复验必红〕）+前端 4 新例；小说域 37 文件 325 例；全量 ci CI OK（漂移闸 735）。**坑**=反向验证语义 mutate 须真的执行到（安全闸下回滚 no-op→直测钉函数）/bash 双引号内 python f-string 花括号被吞/混合态章收集单元回落 blob。**产物**=releases/gaea-v4.432.0.exe 51,744,256B SHA256=`283cb4ee4664716b215459f50438e572cbf71bf40df0896cf230b263c43d8a71`（冒烟 200 过；删 v4.428.0.exe）。**文档**=releases/v4.432.0.md+规格+CHANGELOG/README+releases/README+AGENTS 速览+progress。**留池**=novelreview 并入判据/字段级修补/真机 no-improve 率观察。**刀序**=刀5 风格学习回灌。
- **最新发布：v4.431.0（2026-09-30）「长篇刀3：故事层骨架」**——用户指令「继续」（长篇七刀线程次刀序）。规格 §1.10 纵向结构承载物。**绑定面 729→733（+4：NovelStorySpineGet/Save/Health/Propose）**。**契约**=story_spine.json：主题论证（M12 报告制）/弧线 want·need·misbelief+水位点/节拍位（五预置+自定义）/支线开关表（MICE）/未解问题池，确定性校验整单拒绝。**体检**=七维度零 LLM：支线遗忘>10/问题超期>15/节拍滞后+3/中段塌陷（V2 强度中段<全书 75% 缺数据跳过）/弧线覆盖/终局未闭合/主题缺失。**注入**=CreateChapter+逐场景流同源切片（本章节拍/活跃线程/未解问题/弧线水位），空 spine 零注入（HTTP 断言）。**提案**=从主线+大纲+摘要提炼，只填表单落盘走保存。**前端**=CreatePage rail「故事骨架」面板。**测试**=Go 4 新例+阈值反向验证+前端 5 新例；小说域 36 文件 321 例；全量 ci CI OK（漂移闸 733）。**坑**=Go 源码 \n 字面量修改一律 Edit 工具（python heredoc 三重转义必降级）/node 插入核对声明顺序/产物 SHA 先跑后读勿预写。**产物**=releases/gaea-v4.431.0.exe 51,712,512B SHA256=`0d9023b372649056e54b4ead6c9157b0e0467f9bb6fb10da457712c01f14b94c`（冒烟 200 过；删 v4.427.0.exe）。**文档**=releases/v4.431.0.md+规格+CHANGELOG/README+releases/README+AGENTS 速览+progress。**留池**=M7 不并入/M8 三表/M9 DAG/弧线闸门/自动触发。**刀序**=刀4 质量收敛闭环。
- **最新发布：v4.430.0（2026-09-29）「长篇刀2：场景卡与场景级生成」**——用户指令「继续」（v4.429 后按台账次刀序开刀2）。规格 §1.3/§1.4 正面解。**绑定面 726→729（+3：NovelChapterScenesGenerate/NovelSceneRewrite/NovelSceneCardsPropose）**、spaceBindings 550→553，零功能删除。**数据契约**=SceneMeta 扩 6 创作学字段（Goal/Conflict/Turn/Outcome/Sequel/ExitHook，omitempty 零迁移；SaveSceneMeta 白名单 patch **指针语义**防旧调用方抹卡）+SceneRefs 启用（G10）。**生成链**=GenerateScene 卡注入+前场衔接（无卡 prompt 逐字节零回归）；NovelChapterScenesGenerate 整章按卡逐场生成（scene-gen-stream 流事件/每场即落盘/卡缺目标冲突写前闸可跳过=刀1 覆盖语义/Stitch→blob+SceneRefs/复用同章互斥与取消）；NovelSceneRewrite 单场景 whole 重写（快照+版本库 mode=scene，**他场字节不动**）；NovelSceneCardsPropose AI 拆卡**提案不落盘**（确认制）。**前端**=ⓘ 6 卡字段+弹窗内 AI 重写本场景/按卡生成全章（闸确认）+逐场进度/AI 拆卡审批弹窗。**测试**=Go 5 新例（含关闸反向验证+卡进 prompt 的 HTTP body 断言+他场字节不动）+前端 4 新例；小说域 35 文件 316 例；全量 ci CI OK（首轮挂 SA1012 nil ctx）。**坑**=bash 内联 node 模板串美元花括号被吞（补丁 .mjs 文件化）/CRLF 文件 python LF 锚 miss（node split/join 或 Edit 工具）/antd confirm title 双节点（findAllByText）/白名单 patch 扩字段用指针防抹除/wailsjs gitignore 生成物 fresh checkout 须 wails generate。**产物**=releases/gaea-v4.430.0.exe 51,645,440B SHA256=`74974d7ae9f1e1975dfe6d3363c2bac36380bd1f2cee8408939af29298ea8a11`（冒烟 200 过；删 v4.426.0.exe）。**文档**=releases/v4.430.0.md+规格+CHANGELOG/README+releases/README+AGENTS 速览+progress。**留池**=局部重写解禁/硬闸完整形态随刀3·刀4/逐场景生成+拆卡真机走查。**刀序**=刀3 故事层骨架。
- **最新发布：v4.429.0（2026-09-29）「小说板块收官刀：批2 观察池清账（修 15/关账 2/留池 2）」**——用户指令「继续优化 gaea，本次完成小说板块」。对齐 v4.427 池清账方法论：v4.425 §4.1 观察池 19 项逐项读码定性，**小说板块观察池清空**（G10 留刀2）。前端 8 文件+Go 14 文件+prompts 增删各一，绑定面 726 零变更。**三实锤**=①停止生成连接建立阶段显示「生成失败」（后端 ChatStream 建连 err 无条件 emit error；先判 ctx.Err() 走取消落盘发 cancelled）；②伏笔「失败重读×排队写」交错静默丢改动（op1 失败重读覆盖 itemsRef 后 op2 把磁盘旧内容写回；写链代际短路跳写+提示）；③设定 Agent 回填覆盖等待期手改（ChatPanel 已禁但编辑器不禁；回填前比对快照，已变走围栏+手动应用）。**Go**=缺口即停四合一（ForEachChapter+三处内联：上界改 MaxChapterNum 缺口续扫；定性修正=ForEachChapter 生产零调用）/分支摘要混入主线（ReadMainlineChapterSummaries，outline-continue 改主线口径）/迁移跳章 fail-closed（不落 v4 标记）/嵌套大纲四处递归（写前闸/章计划/场景标记/反推标题表——分卷书闸整体失明根除）/plans 锁/scene_gen 吞错上报 outlineWarning/chapter-review 模板补缺（fallback 提升为工坊真身）/删死资产 chapter-generate.json。**前端**=曲线代际守卫（含 loading 代际收口）/重写历史 projectPath 进 effect/Cmd+K 改受控数据流 updateScene（**微实验实证受控组件下 DOM 赋值+dispatch input 的 onChange 收到旧值**，GhostText 同款手法挂池待真机）/局部重写 reanchored 诚实化（±50 重锚提示）。**测试**=Go 新 8 例+前端新 9 例+四项反向验证改坏必红（曲线守卫切书场景断言恒真——svg 渲染条件被切书破坏，改「重跑体检」场景才有灵敏度）；小说域 35 文件 312 例；全量 ci CI OK（vitest 409 文件 3558 例/漂移闸 726）。**坑**=release.ps1 预检挡未跟踪文件（规格+发布说明先 docs commit）/测试期与后台全量 go test 并发改码=编译竞态自伤（反向验证须在 ci 冻结窗口外）/mutate→restore 锚不完整只回注释没回代码体（还原后必须复跑用例，本批 CancelDuringConnect 曾带病到全量 ci 才暴露）/python 跨行 replace 在 CRLF 文件上 LF 锚不匹配须 node。**产物**=releases/gaea-v4.429.0.exe 51,592,192B SHA256=`f90bee694ceaf3ee7cd624ef3633337f05bf35a56b7b9384d310d14e96a1e7a6`（冒烟 200 过；删 v4.425.0.exe）。**文档**=releases/v4.429.0.md+规格 进度计划/gaea-novel-watchpool-20260929.md+CHANGELOG/README+releases/README+AGENTS 速览+progress。**观察池**=GhostText 受控值疑点待真机/memory 分支摘要语义需拍板/零调用者绑定删除需拍板（宽口径 36）/G10 SceneRefs 随刀2。
- **最新发布：v4.428.1（2026-09-29）「走查沙箱隔离补漏：APPDATA 数据面同隔离（真机走查收官）」**——v4.425~428 三批原罪改动的真机走查（非版本刀）+走查实证安全缺口根修。Go 2 文件，前端零改动，绑定面 727 零变更。**走查全过**=ChatTabs tab 语义+方向键/手风琴 aria+ArrowDown/面板收起卸载+恢复/画廊描述回退链/消息分页（种 105 条：首拉恰 100+提示行+点击载入 105+提示消失）/流链路（回复渲染+console 零错误）；沙箱流可用（默认种子 herdsman）。**险情根修**=ApplyWalkthroughOverride 只重定向 workspace，sin/小说/角色库数据根在 %APPDATA%\gaea 未隔离——沙箱壳直连用户真实故事（全程只读，发现即停）；修=override 内 os.Setenv("APPDATA", Temp\gaea-walkthroughppdata)，全部 os.UserConfigDir 落沙箱；新增 TestWalkthroughOverrideIsolatesAppData（含用例顺序防串场归位）。**坑**=CDP dispatch 与读 DOM 须分两次 evaluate（React 批渲染旧读假象）/sin-acc-panel-* 是 id 不是类（类选择器查空差点误报 visible 失效）/走查先探沙箱数据落点（workspace 空+有故事=直连实盘信号）/修复后走查只需 GAEA_WALKTHROUGH=1。**门禁**=全量 ci CI OK；漂移闸 OK@4.428.1（727 一致）。**产物**=releases/gaea-v4.428.1.exe 51,582,464B SHA256=`380318d95e101f839e26d43061e21dc79c673a8e0744ca714b532a5e4072f746`（冒烟 200 过；删 v4.424.0.exe）。**文档**=releases/v4.428.1.md+CHANGELOG/README+releases/README+AGENTS 速览（迁 1 插 1：一百四十二迁 v4.419.0 入 archive）+progress。**观察池**=设定卡排队体验需 ComfyUI 在线补走/解释腔·工具误用率需真实生成人工评/novelstyle rewrite 链（需拍板）/跨实例族+TOCTOU。
- **非版本刀（2026-10-02）全仓审计第 15 批·簇 B 真 bug（3 条并行线）**——接续第 14 批（`295b6d2a`），线1 主代理直做（types+graph）+ 线2/线3 互斥子代理（agent / app 两文件）+ 主代理预核·逐线复核·**亲手重做反向变异**。**交付**=`docs/code-audit-2026-10-02/round-17-p1-batch15.md`（不抬版本）。**内容**=①**IN2-13** 删 `types.ConsistencyReport/Issue` 死定义（**全仓零消费**实证：`types.Consistency` 0 命中、`overall_note` 除定义 0 命中；graph 版=唯一活形态，注释标注防复活）——删除而非改名/兼容层；②**GA1-09** retry_until ephemeral 模式**同会话续跑接活**（原意判明=补真语义非盲删：doc 宣称累积+重试提示词预设子代理记得上次+死分支是会话不外传的接线未遂；`runSubSession` 回传会话+首试后包 ephemeral wrapper+`seed=nil`——**Seed 是替换语义**（session.go:52-56），不置 nil 会抹掉累积历史；4 项次生影响全 no-op 读码核实）；③**AP1-08** Apply/Restore 四段写回体收敛 2 helper（场景路 applying 分流：Apply=完整性校验+快照，Restore=**无条件还原**，语义差异钉子测试钉死；错误文案/校验顺序/快照时机逐字同原）+ **AP1-07** 三处 9 参取消出口收敛 `savePartial` 闭包。**测试**=+2 文件（`task_retry_session_test.go`：第二请求含首试回复+4 消息钉死 seed 重注；`novel_rewrite_writeback_pin_test.go` 7 Test：四路径+文案逐字+幂等+快照计数），既有测试**零改动**；钉子先绿→重构→一字不改复绿=行为不变时序证明。**反向证据**=线2 短路 wrapper 红 / 线3 拍丢快照红——**均主代理亲手重做**（「子代理跑过 ≠ 只信自述」）。**审计纠错 3 条**=IN2-13 危害偏重（零消费地雷非消费端拿错）/GA1-09 盲删选项被三证据链否决/AP1-07「三份」指调用出口非实现。**门禁**=前台 ci CI OK/exit 0（golangci 0 issues、vitest 427/3712）；守卫四份全绿（test-ctors **364 零新增**、primitives 无新增、contract 无新增、bindings@744）。**坑/教训**=①`Session.Seed` 替换非追加——弱直觉「防重复注入」漏掉「抹掉累积历史」真后果，读语义要读实现；②死分支处置先判原意（doc+调用方提示词+半成品形态三证据）再定删或接活；③小足迹主代理直做+大足迹子代理包互斥，三线并行 `go build ./...` 恒绿。**留池**=writeBackRewritten 远期合并（需统一错误文案）· cancelled 事件词汇两套（语义不同不硬并）；**下一批建议**=簇 C 口径单源（IN1-09/IN1-05/GA3-07/IN3-02/FE4-04/GA6 族）或簇 D 死代码（FE3-03 须按「前端驼峰名+mappings.ts 反查」实现）。
- **非版本刀（2026-10-02）全仓审计第 14 批·簇 A 安全（4 条并行线）**——接续第 13 批（`a08e9129`），4 条互斥子代理线（线1 路径/护栏 / 线2 hook 沙箱 / 线3 whisper 桌面操作 / 线4 前端两条真 bug）+ 主代理预核·独立实证·反向变异·收线缝合。**交付**=`docs/code-audit-2026-10-02/round-16-p1-batch14.md`+`testctors-baseline.json`（不抬版本）。**簇 A 四项**=①**AP2-04** 书封同处注入 `applyImageSafeMode`（**不改走统一入口**，理由写进注释：会接管 size 策略/落盘目录/并发槽 ⇒ 3:4 与 exports 落点回归）；②**AP5-09** 落**已有**叶子包 `internal/gaea/wspath`（`Within`/`ResolveRelWithin`，**30 调用点**，`builtin.within` 上提转调）——现场比审计**更严重**：**13 处真修复**含 `gaea_lint.go` 零校验、`gaea_preview.go` **伪拒绝（返回逃逸路径、赌 Stat 失败）**、docx/pptx/xlsx×6/crosslink×2/xlsx_chart **写侧零校验**（crosslink 还能在工作区外 `MkdirAll`）、`gaea_verify.go`（Journal 不可信面参与回滚写盘）、`gaea_schedule_file.go`（索引指针）、`gaea_deliverable_zip.go`（`\\` 形态漏拦），另修 `filewatch` 误丢合法「..foo」假阴性；③**GA4-06** hook 沙箱 `WithSandbox` 加法（`NewRunner` 签名与 14 处既有测试调用点零改动）+ `Runner.spawn` **唯一注入点** + `sandbox.Command` 的 bool 不再被丢 + 「enforce 但不可用」一次性诚实告警；④**IN4-05 收窄为 5 条真缺口**（黑名单通配符绕过 / `\\?\`·UNC 前缀绕过 / `pathTo` 空值静默落 CWD / 下载类策略盲区 / `Router==nil` 静默 no-op），**IN4-08/IN4-09 按现场撤销**（行号失效；控制组探针证明 `'`→`''` 转义有效）。**前端**=FE6-09 打字机守卫补齐（对齐 canonical `useChatStream.ts:216-230`）+ FE7-11 模块级游标出 `set` 更新器（`idSeq`+`pushHistory` **16 处**）并附带修 `setTaskAssignments` 拒绝路径仍清 redo 栈的真 bug。**反向证据**=线1 2 条（68 断言红）/线3 5 组/线4 4 条**由主代理执行**（含 SHA256 证明字节级还原）。**门禁（前台实跑）**=`go test ./... -count=1` exit 0（app 113.965s）· golangci v2.14.0 0 issues · frontend lint 0 error / build ok / **vitest 427 文件 3712 例全绿**（上批 425/3702）· E 系列 + check-docs OK；守卫五份（primitives --strict 无新增 / **bindings-drift OK@744**——改了 20+ 绑定实现、签名零变更 / contract-drift 无新增 / test-ctors **364** / check-docs OK）。**坑/教训（进在册，可复用）**=①**后台 job 拿不到 `.tmp` 写权限**：`ci.ps1` 放**后台任务**跑必在 `go test` 步大面积 `Access is denied`（两次实测、十个包），**同一命令前台跑全绿**（判别实验见 round-16 §3.1）⇒ **门禁一律前台跑**；②**反向验证的中间态必须还原**——线 2 树里的 `// REVERSE-TEST: bool discarded`+`sandboxed` 无条件置真被收线预检拦下（否则＝「把平台可用当本次已沙箱」的静默撒谎，本批主题正是消灭这类）；③**审计说的「已有安全助手」可能是伪防护**（`resolvePreviewPath` 的 `..` 分支返回逃逸路径）——「有判据」≠「判据有效」；④**子代理跑不了 ≠ 不验证**：其会话被管道 EPERM 挡住时，**配方由它给、变异与红/绿原文由主代理出**；⑤**守卫「放大器」档要真复核**：`matrixApp()` 原为裸 `&App{}`，而 `emit` 是 `*core` 方法 ⇒ 未来输入走到 `GaeaPreview` 非 `.md` 分支即 nil 嵌入 panic，收线改 `&App{core:&core{}}` 降档（bare→core-only，放大器计数不变）再刷基线；⑥默认 `GOCACHE` 在 workspace-write 沙箱下不可写，**`go build` 返回 0 可能只是全命中缓存**（靠文件探针定性）；⑦**删子代理临时物必须逐文件名**（`Remove-Item .tmp\*.cjs` 连带删掉 6 个旧临时脚本——`.tmp` 是共享暂存区）。**留池/拍板**=桌面 agent 接线（`RouterContext` 生产构造 + `DefaultAgentLoopRunner` 传 `Router`/`SessionID` + 自动批准设置——不接线则四层管道永远只经测试触达）· `download_and_install` 真修法（独立确认 UI）· `trashItem` 阻塞风险（两次实测不一致：成功移入回收站 / 挂死 600s）· `GaeaDocumentLint` 读根门 · FE6-09 话题 id 显式化 · `setBaseline` 同族残留 · 符号链接逃逸/两份 `validSceneID`/两份段级校验 · hook 真包裹在无 WSL2 机器测不到；**下一批建议**＝簇 B 余项（IN2-13 双定义、GA1-09 死分支已预核；AP1-07/08 属 `internal/app` 须单独一波）+ 簇 D 的 **FE3-03 死绑定检测**（**必须先按「前端驼峰名 + `mappings.ts` 反查」实现**，批次十三误判教训）。
- **非版本刀（2026-10-02）全仓审计第 13 批·P1 池清账 + 守卫精化（4 条并行线）**——接续第 12 批（`03e0fc0c`）的 4 线条并行清理。**交付**=`docs/code-audit-2026-10-02/round-15-p1-batch13.md`+`progress.md`+`testctors-baseline.json`（不抬版本、不动 CHANGELOG）。**内容**=①**真缺口 D5**：`upsert_task` 补「工期为负」闸（判据单源 `validateDuration(t,idInMsg)`，`Validate`/`upsert`/`patch` 三路径共用；TS 镜像 `negativeDurationError`；golden fixture **只加不改** 112/0，新增「upsert 负工期」+「patch 既有负工期只改 level 不误伤」两例）；②**守卫**：`ci.yml` 删 `internal/tts` 过期排除（理由双重不实：`TestEdgeTTS` 全仓零命中 + 真原因是 `sapi.go` 的 Windows-only `HideWindow`，`GOOS=linux` 交叉编译即报错）· 前端 flaky 分类器从零自测补到 8 组/14 断言并**堵掉「报告无 `testResults` 被放行成绿」**（现 PARSE-FAIL/exit 2）· 遮蔽闸「条数基线+字典序切片」→**显式清单+名字集合差**（新增/消失分开报、位置漂移仅提示）；③**前端收敛**：rel 第 4/5 份副本并单源 · 写类工具字段口径 **3 处 → `toolArgs.ts` 一处**（24 等价矩阵先固化现状再重构，重构后测试文件一字不改仍全绿）· 备份回滚部分失败由「只有 antd 瞬时 toast」→**页面内持久 Alert**；④**app**：索引入队去重**唯一入口**（判据与落库同 space、跨空间互不吞并）· 板块 ID 清单 **4 处硬编码 → manifest 派生** · 知识库迁移失败经既有 notice **启动即可见**（零新增绑定）。**验证**=`scripts/ci.ps1` = CI OK / exit 0（golangci 0 issues、vitest 全量）；主代理独立复跑 `internal/schedule` 1.03s / `internal/app` 106.7s / 前端 6 文件 86 passed；守卫五份全绿（test-ctors 基线 359→**363** 已评审）。**坑/教训（进在册，可复用）**=①**绑定名 grep 必须「前端驼峰名 + `mappings.ts` 反查」**：前端写 `app.DataBackupRollback()`，运行时经 `gaeaToGaea[prop] ?? prop`（`proxy.ts:158`）改写为 Go 名 `GaeaDataBackupRollback`——只按 Go 名 grep 会**系统性漏报**（主代理本批即误判「GA4-11 前端零消费面」，被线 3 用调用点/mock/分面证据推翻）；**这条直接决定下一批 FE3-03「死绑定检测」的实现口径**；②**并发子代理的中间产物不是原子交付**（承接第 12 批）——本批任务书要求各线「接近上下文上限/长时间无进展时先用 `send_message` 交阶段性结论」，结果**四线全部交齐报告**（上批两条线崩溃丢报告），此纪律继续沿用；③**子代理在局部契约上可以比主代理更懂**：线 1 驳回主代理「patch 闸宽外溢」建议并给出依据（`ops` 契约=只校验本次改动，外溢会让与工期无关的补丁被拒、错误文案指向用户没碰的字段、**同一条 op 在不同计划上随机成败**），主代理采纳并以两侧探针 + fixture 用例锁定；④**golden fixture 只加不改**：本批 `numstat=112/0`（既有 65 条逐字节不变），收线时应把「既有期望被动过」当红旗追问；⑤**审计因果链要按现场复核**：AP4-07 引的 `novel_import_task.go:55` 提交的是 `KindOutlineReconstruct`，全仓 `KindFileIndex` 仅 3 个提交点且当前无 play 生产者 ⇒「正在抖动」不成立，按「埋着的地雷」定性（修法照旧落地）。

- **AGENTS.md 八迁分流（2026-09-13，非版本刀，纯文档）**——CI 卫生守卫 WARN（59926B 逼近预算）触发：v4.261.0~v4.255.0 六条入 archive（八迁记录更新），主文件恢复 14 版，水位 59926→47518B；迁移完整性三项断言全过。坑=CRLF 行尾下 node indexOf 锚点不带行尾+写入前验 includes。




- **v4.174–v4.178（2026-09-09）瘦身 P3 版3/P4 三刀与造价条目匹配键刀**——全文迁入 docs/archive/agents-version-history-2026-09.md（2026-09-11 三迁腾预算，check-docs 指令预算守卫）。
- **非版本刀（2026-09-23）源码包归档推送**——wubigork-v4.399.0-source.tar.gz（13028977B/4744 files，SUMS 双行）+ 29 个未推 commit 与 v4.373~v4.399 tag 全量上 remote；画室编辑力八连发里程碑打包。坑：git archive 的 -o 是全局选项须放 pathspec 前（--output= 形式最稳）。

## 执行纪律：默认并发子代理（2026-09-03 用户强化习惯）

用户已把「并发子代理」从点名指令固化为**默认习惯**：后续任务默认按此执行、
无需再次点名「并行使用子代理」。

1. **开工先拆线**：任务含 ≥2 条互不相交的线时，先列出「线 × 文件足迹」再动工；
   2-4 条独立线用并发子代理分头执行（运行环境 4 并发槽位），主代理负责定契约、
   跑全量门禁、集成收口——v4.54-v4.59 的「三线并行 + 主代理收口」即标准形。
2. **单线也倾向派活**：一条独立成刀的任务（调研/实现/测试/文档）若体量超过一轮，
   优先派子代理并发执行，而不是排队串行。
3. **足迹互斥铁律**：线间文件足迹不相交；契约/生成类文件（types.ts / bridge.ts /
   mock.ts / 三语字典 / gen_bindings 产物）必须指定单一负责人，生成动作由主代理在
   所有后端子代理完成后统一执行，防止并发写覆盖。
4. **每刀回写**：刀末把本次分线/收口经验（含教训）写回本文件与 `.gaea/progress.md`，
   让习惯持续强化；若某刀必须串行，收口时说明原因。

## 交互纪律：不许用「等待」换时间（2026-09-10 用户拍板）

用户原话（对上一轮执行的批评）：「后台运行的东西你为什么要等待」「在浪费我的时间」。
耗时本身不可怕，**干等与重复**才是浪费。后续每一轮工作按下办：

1. **后台任务不阻塞**：起后台任务后立刻去做别的事（改码/读文档/跑定向用例/写文档），
   收到完成通知再取结果。**禁止起完就守着等**——那几分钟是白扔的。
2. **重活先报 ETA 再跑**：实测耗时——全量 vitest ≈ **140s**（324 文件 2799 例）、
   `scripts/ci.ps1` 全门禁 ≈ **5–7 分钟**（Go 全量 + 前端 lint/build/vitest）、
   `go test ./...` ≈ **2–4 分钟**。开跑前一句话说清「跑什么、约多久」，用户可否决。
   **「CI 绿」口径（2026-09-27 定）**：写门禁结论必须带定语——「本地快闸」=
   `scripts/ci.ps1`（构建/vet/golangci v2.14.0/Go 全量/漂移/前端/E系/卫生）；
   「Actions」=.github/workflows/ci.yml 三 job（backend+race+frontend，race 检测
   只在 Actions：本机无 gcc）。禁无定语「CI 绿」。
3. **定向优先，全量收尾**：改动后先跑目标文件（3–10s）确认；全量只在交付前跑一次。
   一轮改动 = 中间定向 + 收尾全量，**同一验证不重复跑**（上一轮全量跑了 4 遍=反例）。
   **工具（2026-09-27 立）**：`scripts/test-related.ps1` = git 改动 → Go 包
   （吃结果缓存，不带 -count=1）+ `vitest related`，几秒~分钟级；`ci.ps1 -Quick`
   = 静态闸（build/vet/lint/卫生守卫）+ 受影响面测试，实测 ~1 分钟（vs 全量
   5-10 分钟）。**全量档（ci.ps1 不带参）保留在 push 前/发版前跑**；
   用户对「几分钟」的抱怨=2026-09-27 拍板用两档制化解，勿再无脑全量。
4. **超时上限按实际需要写**：不要写 900s 这种夸张数字——那是上限不是耗时，
   只会造成「要卡你十几分钟」的观感。
5. **不把「等待」当进度表达**：进度由已完成的具体产出说话，不由轮询次数说话。
   用户催问时先答「在跑什么、还要多久」，再继续。
6. **能并行就并行**：长任务是可并行的（截图取证 / 读码 / 单测互不依赖），
   串行排队本身就是一种浪费。

> 与上一节的关系：并发是**手段**（把时间省下来），本节是**约束**（别把省下的时间又等回去）。

## 工装：CDP 走查与前端调试（2026-09-10 增补）

- `scripts/cdp-walk.mjs` 支持 **`--target <url 片段>`**（多标签时选定目标页，缺省取第一个
  page）与 **`@文件路径` 传入 JS 表达式**（PowerShell 5.1 传原生命令参数会吃掉内层引号，
  长表达式一律写文件再 `@` 传入）。真机走查配方：无头 Edge
  `--headless=new --remote-debugging-port=9333 --window-size=1440,920 --user-data-dir=<tmp>`
  + Vite dev（`?mock=1`）→ 本脚本 eval/截图。
- **真机走查沙箱开关（v4.418，必用）**：`GAEA_WALKTHROUGH=1` 启动壳=workspace 强制指向
  `%TEMP%gaea-walkthroughworkspace`（覆盖用户/项目配置），会话/记忆/交付物全落沙箱，
  走查后删沙箱目录即清场——首条消息自动装配+自动恢复会把回合接进真实会话，此开关是
  唯一根治（2026-09-26 险情防复发）。未设时零行为变化。
- **Vite dev 缓存坑**：同一 CSS/源文件在**同一秒内的两次编辑**，mtime 粒度相同会让
  HMR/转换缓存认不出改动（表现为浏览器仍跑旧样式，reload 也无用）。处置=改完
  `(Get-Item file).LastWriteTime = Get-Date` 再 reload。
- **`vitest` worker 上限**：`maxWorkers` 必须按**物理核**（≈逻辑核/2，夹 [4,8]）而不是
  逻辑核——超配会把单用例墙钟拉到 CPU 时间的数倍，重组件首个 `await import` 直接
  撞 `testTimeout` 假红（详见 `frontend/vite.config.ts` 注释）。

## 长期规划（权威，2026 定稿）

- **下一阶段规划 = `docs/gaea-next-stage-plan-2026-09.md`（2026-09-10 活跃指导）**：接棒长期规划（阶段一~四已收官）——阶段五 记忆 OS+上下文编译（主轴）/阶段六 书斋纵深——办公主角（可审计默认化·记忆驱动项目本体·多文件 DAG），进度（DCMA 体检·蒙特卡洛工期带）与跨域 EVM 为两翼/板块并行池/拍板池（信息价接入·MCP 翻案·壳外 computer use）/维持轨。提方向前先对其 §0 在册边界（GoalCard v3.6.0 撤下、MCP/平台化不进规划、不做算量、DSH 独立窗口等）。
- **用户拍板（2026-09-08，收敛计划 §0）：性质路线=A「终极个人工具」**——产品化
  降为期权不作承诺，不为想象中的用户写代码；**LICENSE=私有 All Rights Reserved**
  （根目录 LICENSE），未来开源须另行发布开源许可证覆盖对应模块并与私有部分区隔。
- **瘦身长期总规划（2026-09-08 立项）= `docs/gaea-slim-masterplan-2026-09.md`**：七面
  （认知/资产/结构/轮子/运行/产物/数据知识）×六阶段（P0 基线→P5 维持）；治理规则仍在
  收敛计划。防复发规约自 P1 起生效（新下载走 saveExportBlob/新选取走 pickFile/新 diff
  复用 lib/diff/新 slug 用 strutil.TitleSlug/新解码用 b64ToBytes/NAVIGATE 用 manifest id/
  新域先问「能否是文档」）。
- **审计衍生优化方向（2026-09-26）= `docs/gaea-optimization-direction-2026-09.md`（候选池；**P0-1 门禁落地 + P0-2 comfyUI 竞态已于 2026-09-27 落地**，见该档 §3 状态注记）**：
  本次全仓审计（三路并行模块审计 + 本机实测复现）——综合 7.0/10 + 分维与 14 模块评分 + P0~P2
  优化项（门禁落地 / internal-app 竞态 / 台账自动化 / 上帝包拆分 / 281 遮蔽绑定 / 平行实现收敛 /
  效果验证闭环 / 甘特库评估 / 仓库体积）+ 本次整理清单 + 未验证项观察池。**排刀前先读它的 §3 与 §5**；
  与既有权威路线冲突时，仍以本段规划为准。
- **用户再拍板（2026-09-08）：工位与乐园并列平级**——办公不是乐园的上位核心，瘦身
  不是「收乐园保办公」；认知面问题=13 板块平铺单一导航面、并列结构未表达。P2 措辞
  已由「乐园折叠」改为「双空间并列落地」，乐园板块在乐园空间内一级可见。
- **唯一权威路线图 = `docs/gaea-nextgen-roadmap-2026.md`**（11 个子代理调研合成）：
  8 板块竞品调研（办公/造价/AI 底座/编程/小说/绘梦/轻语/微信+语音）+ WorkBuddy×灵犀
  深度拆解（§12）+ **版本重定义"双空间"（§10）** + 四层落地（§13 后端/前端/UX/UI）+
  执行计划（§14 阶段 0 地基 → 阶段 1 双空间内核 → 阶段 2 双空间壳 → 阶段 3+ 领域包）。
- **用户拍板：工作与娱乐分开、互不干扰**——工位（办公/造价/资料+工作记忆）与
  乐园（轻语/小说/绘梦/阅读）双空间硬隔离；记忆分区互不检索、模型策略各配各的、
  上下文永不跨界；跨空间仅用户显式发起（如"把乐园封面放进报告"）。
  （编程板块 v4.439 经用户拍板「删除编程板块」整块移除：manifest/页面/Go 绑定
  五方法/暗色 mock 面全部退役，independent 分面归零但机制保留。）
  ~~"陪伴×办公融合"（旧 v4.3）已删除~~。
- **本轮关键纠正**：灵犀 = 金山 WPS 独立 AI 办公 Agent（非阿里/通义系）；
  WorkBuddy = 腾讯云 CodeBuddy 全场景 AI 办公工作台（非 Kimi 系；Kimi Work 是月之暗面的）。
- **i18n 决策（2026 追加）**：采用审计 §405「诚实 zh-only」选项——**壳层 chrome +
  设置外壳三语**（已完成，消灭壳层混合语言根因）；**页面内容层保持 zh 单语**，
  不再逐页铺 en/zh-TW 字典（个人中文工具无国际化受众，~5000 字符边际价值≈0；
  未来需国际化时按 S2.3b WireShape 模式整页迁移）。
- **文档纪律**：docs/ 旧调研/已落地计划已归档至 `docs/archive/`（见其 README）；
  后续会话以本文件 + 长期规划 + `.gaea/progress.md` 为权威，勿引用 docs/archive 结论。
  **新文档必须登记 `docs/README.md`**——守卫 `node scripts/check-docs.mjs` 四查（孤儿登记 /
  `docs/` 悬空引用 / **本文件字节预算 ≤ 65536 B**，超了尾部纪律段就对会话不可见 /
  **含非 ASCII 的 .ps1 必须带 UTF-8 BOM**——2026-09-10 实撞：编辑 ci.ps1 丢 BOM，整条门禁语法错静默不跑），已接入 `scripts/ci.ps1`。
- **执行审计（2026-08-30）**：`docs/archive/audit-2026-08-30-v4-execution-review.md` 记录
  v4.x 全量「承诺 vs 代码」对照——裁决=最小版执行（骨架真、纵深欠账）。红线缺口
  三条（记忆注入跨空间未接线 / 任务分账未启用 / 事件过滤仅 1 处）与补课刀序见该文
  §B/§E；后续每刀验收新增「纵深检查」，发布说明必须列欠账清单。
- **执行状态（v4.8.0 后）**：审计欠账大面收账——读屏纵深（多显示器/OCR 本地
  摘要/截图留档）、intent LLM 兜底分类器（默认关，白名单+置信门+硬超时）、
  生图产物 CardPath 接通、iLink 微信通道离线收敛（限频/下载防线/识别管线/
  防御解析/SendFileCard seam）、全局离线模式总开关（EngineType.IsLocal +
  routeModel 云过滤）、成本知识图谱可视化（BuildGraph + CostGraphView 第 8
  模块，绑定面 533）、实时语音 Realtime S0 铺底（internal/realtime seam）。
  剩余欠账（Realtime S1/S2、iLink 真机窗口、离线模式设置 UI、权限升级请求+
  stubGate 竞态、XlsxPreview 虚拟滚动/生命库可写化=观察项）见
  `releases/v4.8.0.md` 欠账清单。
- **欠账收尾小步（2026-08-30，v4.8.3 后）**：VoiceStart realtime 门小修
  （端到端回复走服务端 response 事件，whisperChatFn=nil 也可启动，拼接
  管线双门逐字节保留）；持久化套件统一（desktop_session 原子写 + archive
  JSONL 单次 Write 落整行）；XlsxPreview 大表格行虚拟滚动（观察项收账，
  300 行以上只渲染可见窗口 ±overscan）。Go 全量绿、vitest 809/809、
  tsc/eslint 0、绑定面 535 不变。
- **下一执行**：v4.8.3 已发布（微信图片双向真协议）；剩余——Realtime 真机
  验证轮（真 key 下端到端对话/打断体感/AEC 实效，S2 骨架已就绪待真机数据）；
  手写体识别质量复测（多模态 Qwen 升级后）；iLink 语音/视频等未探明 item
  维持宁漏勿误静默跳过；生命库可写化=观察项。

## 版本状态（历史存档）

> 2026-09-09 整理：本段逐版历史（v2.x~v4.135，约 2100 行）整体迁往 `docs/archive/agents-version-history-2026-09.md`，按需检索；当前动态见顶部速览，发布物全文见 releases/。

## 项目定位

gaea 是 Windows 桌面端「通用办公」AI 助手（Wails v2：Go 1.26 后端 + React/TypeScript/Vite 前端）。
核心能力：文档撰写、表格处理、格式转换（docx/xlsx/pdf → Markdown）、图表生成、报告拼装、
知识库与记忆中枢、方案编写。品牌定位已从「土壤修复工程办公」全面转为「通用办公」。

## 技术栈与关键约定

- 桌面框架 Wails v2.13（Go + WebView2）；后端事件总线 + 前端 zustand 桥接（bridge.ts → window.go.app.App）
- **绑定面（v2.17.0 起）**：App 不再直接绑定 Wails；429 个导出方法拆 10 个板块门面
  （internal/app/bindings_*.go：CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB，
  纯委托零逻辑改动）。改绑定面方法后用 `go run ./scripts/gen_bindings` 重新生成 +
  `TestBindingsCompleteness` 兜底；前端调用经 gaea/lib/bridge.ts（按方法名路由门面）或
  api/bridge.ts 的 window.go.app.App 兼容代理；旧 wailsjs 导入走 src/wailsjsCompat 重导出；
  wails build 会重生成 wailsjs/go/app/<门面>.js
- 单模型架构：一个 executor 完成规划与执行，无独立规划器；任务/技能子代理走 `task` / `run_skill`
- 内置工具精简为 17 个核心工具（v2.4.3 起）：文件/命令、网络、任务、记忆/知识、技能、format_convert、chart_gen
- 文档能力交给 ModelScope 技能：docx / pdf / xlsx（安装在 `~/.codex/skills` 与 `.gaea/skills`），
  转换引擎共用 `internal/office/docmd`（format_convert 工具与预览面板同一实现）
- 内置子代理技能：format-convert / chart-builder / doc-assemble
- 记忆系统：SQLite（`%APPDATA%\gaea\Hephaestus.db` facts 表，按项目 slug 隔离）+ 文档记忆（AGENTS.md 层级）
- 环境依赖：LibreOffice（soffice）、node 全局 docx、Python 3.13（lxml/openpyxl/pypdf/pdfplumber/reportlab/pandas/matplotlib 等）
- 本地 AI 底座：**Herdsman**（localhost:8080/v1，~110GB 模型：35B 对话 ×2、zimage-turbo、voxcpm2、
  mineru、embedding/reranker、paddleocr、sherpa-onnx 等）；gaea 的聊天/视觉/检索/OCR/解析/ASR/TTS/生图/翻译
  全部依赖它，herdsman 升级可能破坏契约——用 App.HerdsmanProbe 启动探测

## 发布流程（2026-08-14 修订：补版本资源步骤）

1. 更新 CHANGELOG.md / README.md（版本表）/ wails.json（productVersion）/ releases/README.md（版本表）
2. **同步版本资源**：`build/windows/info.json` 是 Wails 生成版本信息的模板（fixed 段必须含
   `product_version`，否则 exe 的 ProductVersion 为 0.0.0.0）；根目录 `versioninfo.rc` 是遗留物，一并更新以免误导
3. 构建（本沙箱：`cd frontend; npm run build` → `wails build -s`；本机：`cmd /c build.bat`），
   产物 build/bin/gaea.exe（同时复制到桌面）；本机 build.bat 已内置真实退出码检查 +
   默认自动冒烟（.tmp 临时副本 → scripts/smoke.ps1，18999 /api/health 200，失败即停；
   `build.bat skip-smoke` 可跳过，发布前不得跳过）
4. 复制 exe 到 `releases/gaea-v<版本>.exe`，生成 `releases/SHA256SUMS-v<版本>.txt`；
   **本地产物只保留最近 5 版**（2026-09-10 用户拍板）——发版后删掉第 6 新的那一版，
   更早版本的身份以 `SHA256SUMS-vX.Y.Z.txt` 为准（旧 exe 不入库）
5. 写 `releases/v<版本>.md` 发布说明（含 SHA256 与冒烟结果），更新 releases/README.md 版本表
6. 冒烟：`scripts/smoke.ps1 -ExePath releases\gaea-v<版本>.exe`（/api/health 200 即通过）
7. 更新 `.gaea/progress.md` 进度记忆与本文件（版本状态）

## 沙箱环境备忘（2026-08-14 整理，详细版见 docs/2026-08-14-sandbox-environment-notes.md）

**防止重蹈覆辙的四条铁律**：
1. `go telemetry off` 已持久生效；构建缓存写入问题随 danger-full-access 策略解除，无需再覆盖 GOCACHE
2. **wails build 前端编译会挂起**（wails 捕获前端输出走管道）——必须 `cd frontend && npm run build`
   再 `wails build -s`（-s = 跳过前端编译，9s 完成）
3. `go test ./...` 单进程树会被 harness 终止、个别测试二进制偶发 `fork/exec Access is denied`——
   逐包验证 + `scripts/test-all.ps1`（逐包/重试/状态续跑）；exec 拒绝用 `go test -c` 手动运行证明代码无恙
4. .ps1 脚本必须 UTF-8 带 BOM（否则 powershell.exe 按 GBK 解析报错）；npx 用 `& 'C:\Program Files\nodejs\npx.cmd'`

## 本地 TTS 引擎（重要记忆，勿遗忘；2026-08-09 整理）

> ⚠️ **VoxCPM2 已于 v2.6.9 移除**：实测不达标（耗时长、音色男女混乱、克隆不稳定）。
> 下方 VoxCPM/Vulkan 相关方法保留为「已废弃教训」，勿重新安装；当前本地 TTS 为 CosyVoice2。
> 注：herdsman 侧实测 voxcpm2 可用（冷启动约 50s，不支持预设音色），qwen3-tts-* 未安装。

本机（Radeon 8060S 核显 / 64GB 统一内存 / Windows）本地 TTS 有两条引擎线，gaea 只连 OpenAI 兼容 8020/8010。

### ~~VoxCPM2~~（已移除 v2.6.9，以下为废弃记录）

- `8030`：主后端 `C:\AI\llama-omni\build\bin\llama-tts-server.exe`（llama.cpp-omni，C++/ggml + Vulkan）
  - 模型：`C:\AI\llama-omni\models\VoxCPM2-BaseLM-Q8_0.gguf`（1.65GB）+ `VoxCPM2-Acoustic-F16.gguf`（1.74GB）
  - 8060S 识别 `KHR_coopmat + bf16`，全量 29 层 offload Vulkan0，加载约 2s
- `8021`：备胎 ROCm PyTorch（`C:\AI\voxcpm\server.py` + `VOXCPM_PORT=8021`）
- `8020`：适配层 `C:\AI\voxcpm\adapter.py`（FastAPI，gaea 入口，契约不变）
- 一键启动：`C:\AI\voxcpm\start_voxcpm_stack.ps1`（8030/8021/8020）

### CosyVoice2（端口 8010）

- `C:\AI\cosyvoice\server.py`：LLM 段 GGUF + Vulkan（`gguf\cosyvoice_f16.gguf`），flow 段 ONNX + DirectML（5 步）
- 启动：`C:\AI\cosyvoice\start_cosyvoice.bat`；约 14s 加载+预热，短句 ~1.5s

### 音色（两引擎统一 4 个，火山引擎 Speech-AI-Forge-spks 录音室样本）

- 中文女 `zh_female.wav`（f0≈221Hz）、中文男 `zh_male.wav`（f0≈133Hz）
- 英文女 `en_female.wav`（f0≈191Hz）、英文男 `en_male.wav`（f0≈109Hz）
- 参考音频 ~7s / 16kHz；转写在 `C:\AI\voxcpm\voices\_meta.json`

### 本次优化方法（AMD 核显提速教训，勿重蹈覆辙）

1. 不要再用纯 ROCm PyTorch 追赶速度：iGPU 共享内存架构下 ROCm 与 CPU 基本相同（RTF ≈1.06~1.12）；
   Vulkan + ggml 的 GEMM/coopmat 才是突破口（克隆 RTF 0.65~0.84）
2. 构建：MSYS2 UCRT64，`cmake -B build -DGGML_VULKAN=ON -DGGML_NATIVE=ON`
3. 坑 1（端口绑不上）：server-voxcpm2 会构造 SSLServer，空证书导致 is_valid_=false 任何端口 bind 失败；
   本地回环不需 TLS，改普通 httplib::Server
4. 坑 2（克隆近静音）：AudioVAE 参数特征必须 frame-major（`ggml_cont(latent)`），
   不能 `cont(transpose(latent))`（dim-major）
5. 坑 3：llama.cpp-omni 的 CLI `-r` 克隆偶发偏静音，HTTP server 路径正常；生产走 server
6. 坑 4：VoxCPM Python 长文本 CFG 2.0 会「静音+整段重试」（RTF 4.8~7.8），CFG 1.5 稳定；
   C++ server 端用 max_steps 限制解码上限
7. 网络：HuggingFace LFS 直连/hf-mirror 都不通，ModelScope 直链快（8.6MB/s）
8. 实测：短句克隆 RTF 0.65~0.84（6 步）、语音设计 0.57~0.60；同 seed 输出确定

### 详细记录

- `docs/archive/2026-08-09-voxcpm2-integration.md`（VoxCPM2 全部历程）
- `docs/archive/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`（CosyVoice GGUF 提速）

### 自动启动（当前仅 CosyVoice2）

- gaea 启动时后端 ensure cosyvoice；模型中心 TTS 模型卡片「启动」按钮 → `App.StartLocalTTSService(engineId)`；
  引擎连接测试兜底 ensure（等约 8s）；TTS 合成前兜底 ensure
- 实现：`internal/app/tts_service.go`（core.ensureLocalTTSService 幂等 + 异步轮询，emit `tts-service-status`；
  CosyVoice 直接 python server.py，隐藏窗口 CREATE_NO_WINDOW）
- 端口探测：CosyVoice2 `8010/v1/models`

## 已知注意

- 角色库剧照默认跟随绘梦（ImageBackend/ImageModel），可在模型中心单独绑定
- 文生视频依赖本地 ComfyUI 安装 LTX-Video 模型
- 里程碑：2026-08-12 完成通用办公全面打磨（显示/布局/安全三线：成本库/记忆/知识库/技能写入全部硬性确认
  含 yolo、子代理路径；子代理不再继承持久化写入工具）
