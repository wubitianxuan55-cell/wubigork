# 当日总账（2026-11-06 凌晨）：聊天板块核心优化第一批——众神殿人格论证四卫生切片全落地

> 依据众神殿两场连议收官结论（pan_1793883599669 + pan_1793885604229，议题「角色（带人格）
> 聊天胡乱说话、不作为」，用户已确认小结）：「胡乱说话/不作为」拆成来源污染、错误压制、
> 假成功、连续性缺口四类病因，首批四切片互相独立可单独回滚。主计划=进度计划/
> gaea-chat-persona-optimization-20261105.md。本批 5 笔提交，全部只读论证→代码落地，
> 压制文案包/续接/工具循环按裁决挂池未动。

## 提交索引（5 笔）

| 提交 | 刀 | 一句话 |
|---|---|---|
| 13b65f37 | 刀0 | 探针字段基建：TurnTrace L3 增 GateLevel/StrategyLevel/ClampInjected/InjectedFactIDs + whisperChatCore 收尾探针日志（raw/llm 长度哈希、历史面、终态、ttft）+ `chat-persona-baseline` tag |
| cfff7a0f | 刀1 | 空正文/取消守卫：非取消空正文有界重试一次仍空→显式 error；空轮/取消轮不推 WM/PromptHistory/不 FinalizeTurn/不入记忆；appendChatExchange 移到 cancelled 判定后；AppendExchange 空 assistant fail-loud |
| f429cecd | 刀2 | 生日显式化：删 nil→now 隐式默认；gaea 显式接 06-11（死常量首次接线+ParseAckemBirthdayMMDD）；存量等值行自愈（非 gaea 清 nil/gaea 置 06-11，幂等+先备份） |
| a931d216 | 刀3 | 搜索入口 raw/llm 双通道：Raw 进 TurnPlan/PreLLMTurn/WM/FinalizeTurn/记忆，增广块只进 llmUserMsg+PromptHistory；KV 缓存契约逐字节不变 |
| 16b52dda | 刀4 | 重启预热+发布序：先恢复后发布（并发首轮拿不到半成品）；chat_history 干净行 ≤4 轮回灌 PromptHistory（脏行五标记跳过，重启不再失忆） |

## 当场逮到的真问题（测试先行抓出）

1. **建连期取消走 callErr 早退**：首 token 前取消被误报为故障（error 终态）而非
   cancelled——原逻辑该分支在 cancelled 判断之前；且 ChatStreamMessages 错误路径
   cancel 返回 nil，直接调用=panic（守卫测试当场红）。
2. **sin 导出测试种数据撞防线**：AppendExchange 拒空后，历史空行改走 AppendMessage
   单条口种入（导出跳过旧库空行的测试语义保留）。
3. **heal 测试初版漏恢复序**：orch 未先 LoadCompanionStateFromDB 就 heal，等值门
   直接早退——生产序=先恢复后自愈，测试照此修正。

## 验收

- 新增测试 12 例全绿：空流/仅reasoning/取消空轮三终态用例+AppendExchange 防线、
  生日显式化三例+自愈迁移三例、双通道唯一标记两例、重启回灌+并发发布序两例
- 全量：`go build ./...`、`go vet`、`internal/app` 全包（~139s）、`internal/whisper/...`
  （prefix_stability 三契约/并发/生日）全绿
- 双通道验收口径：唯一标记只出现在本轮模型输入与 PromptHistory（后者=本批明确不改
  的既定口径，跨轮重放治理挂池）；WM/chat.db 全原文

## 余项（挂池，见主计划 §三）

五桶+15轮+重启差分实弹基线（刀0 字段已备，需本地引擎空闲+授权，先 baseline tag
后修复 HEAD 各跑一遍）→ 压制两臂实验（A′ 文案 vs 意图/分类门控，六条文案五代码位）
→ 来源字段契约立项 → 多轮续接（HEAD 复现门槛）→ 有界工具循环（真实办事坏例门槛）。
