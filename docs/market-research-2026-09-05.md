# gaea × GitHub 生态市场调研合成（2026-09-05）

> 三路并行子代理原始稿：`docs/research-2026-09-05/`（agent-workbench.md /
> context-observability.md / desktop-assistant.md）。本文为合成结论 + 对
> gaea 长期规划的回填建议。数据为 2026-09-04/05 GitHub 快照，未核实项均
> 已在原始稿标注。

## 一、三条调研线的一句话结论

| 线 | 对象 | 一句话结论 |
|---|---|---|
| A 代理工作台 | codex/claude-code/opencode/Cline/goose/crush/gemini-cli（+Roo-Code 停运样本） | 2026 代理工作台收敛为五条 UX 共识：上下文常驻可视化、终止级联、工具结果富媒体化、文件活动面板化、一引擎多前端 |
| B 上下文可观测 | langfuse/Phoenix/AgentOps/ccusage + 图片 token 公式 | 计量与定价解耦、token 互斥拆桶、图片 token 走 patch 精确式（⌈w/28⌉×⌈h/28⌉）、聚合防父子重复计数、聚合→逐步→单调用三级下钻 |
| C 桌面 AI 助手 | lobe-chat/cherry-studio/chatbox/open-webui/jan/anything-llm | **六家无一具备 docx/xlsx 真编辑**——全场停在"解析→RAG→文字回答"；微信渠道已被 lobe-chat 破独占，但"微信收件→改好文档回传"闭环无人做 |

## 二、gaea 现状对标（拿手感）

**已对齐或领先**：逐请求上下文构成 + 逐类 signed delta（B 线"防父子重复
计数"——gaea 的 Go 权威逐请求 delta 是天然优势，Phoenix#3164 的通病
gaea 规避了）；文件活动 ±行增量/操作日志跳转（A 线"文件活动面板化"）；
任务强杀+两击确认（A 线"终止"半步）；Office 真编辑（C 线全场空白）。

**明确缺口**（按 B/A 线共识排序）：
1. 工具结果图片缩略卡 + 图片 token 估算（2.5b 后半；公式已核实：Anthropic
   现行 ⌈w/28⌉×⌈h/28⌉，先档位缩放再计，旧 /750 系统性高估大图）；
2. 成本费率 hover + 缓存/推理 token 互斥拆桶（B 线共识；ccusage 的
   LiteLLM 定价快照 + 用户覆盖单价是现成口径）；
3. 「剩余上下文 %」常驻状态栏（codex 式；gaea 已有完整数据，只缺常驻位）；
4. 终止级联收口：主任务中止→子代理/后台命令连带清理（claude-code/cline
   均为此打过修复仗；gaea v4.78 强杀只到单任务）；
5. 失败子代理以「可恢复 task_id」呈现（opencode 式，失败≠终点）。

## 三、长期规划回填建议（融合三线）

### 近期（蒸馏规划既有队列，1-2 版）
- **2.5b 后半**：图片缩略卡（vision 链路不动的红线只约束识图执行链，
  结果展示层不受限）+ 图片 token 估算按 ⌈w/28⌉×⌈h/28⌉ 官方口径显示
  「缩放后尺寸→实际计费 token」成对信息（B 线第 3 条）。
- **2.5e /context 弹层**（待拍板）可与 A 线"上下文常驻化"合并考虑：
  拍板项升级为「常驻剩余上下文% 徽标 + 点击弹层」二合一。
- 1c HTML 沙箱预览照旧。

### 中期（新增候选，按价值排序）
1. **成本费率 hover + token 拆桶**（B 线蓝本：互斥桶 total=求和、
   regex 模型匹配费率表、用户覆盖层接 v4.59 自定义引擎用户价目）。
2. **终止级联**（A 线）：OnForceKill 钩子扩展到子代理树+后台命令注册表。
3. **失败子代理可恢复呈现**（A 线）：TaskCenter 失败卡给"重试/续跑"
   入口（follow-up 管道 v4.64 已有底座）。
4. **三级下钻联动收尾**（B 线第 5 条）：聚合（Token 卡）→逐步（趋势）→
   单调用（深读面板）——v4.79~v4.82 已建成大半，补 Token 卡→趋势锚点。

### 战略层（C 线，营销与路线措辞）
- **Office 真编辑是全场空白**：宣传口径从"AI 助手"改为"AI 改好的文档"，
  招牌场景=框选→改→落盘回写；实现红线不变（换壳不换芯）。
- **微信卖点升级**："能接微信"不再稀缺（lobe-chat Channels 官方支持），
  卖点改为"微信收到→改好 docx/xlsx 回传"的 IM 闭环交付（文件收发 B 刀
  抓包前置工作应提速）。
- **合规叙事**：jan 提示"数据不出本机"有市场；gaea 本地优先 + 自研
  （无 AGPL/保品牌传染）可作对外表述。
- **生态风险**：Roo-Code 停运样本——gaea 不依赖单一宿主生态的路线
  得到反向印证。

## 四、信息来源

- 原始稿三份（含全部仓库链接、星数、license、公式出处）：docs/research-2026-09-05/
- 关键外部锚点：platform.claude.com/docs/en/build-with-claude/vision（图片
  token 官方口径）、ccusage/ccusage（本地用量聚合口径）、lobehub.com
  Channels 文档（微信渠道支持列表）。
