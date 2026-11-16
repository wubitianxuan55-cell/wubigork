# 聊天消息分支版本树设计（A 批）

- 日期：2026-11-16
- 上游参照：deepseek-ai/chat-demo `branches.ts`（MIT，全量通读台账
  docs/gaea-deepseek-chatdemo-distill-2026-11.md §1.1）
- 结论先行：**存储零迁移**（分支字段进 `chat_messages.extra` JSON），**叶子回溯**
  替代根前向走（分页友好），**首次分支操作触发整话题规范化**（新旧行共存）。

## 1. 数据模型（零 schema 迁移）

消息 `extra` JSON 新增可选键 `bt`（branch tree）：

```json
{ "bt": { "p": 123, "c": 456 } }
```

- `p`（parent）：本消息的父消息 id；根消息为 `0`。
- `c`（selected child）：**选择指针挂在父消息上**（DeepSeek 同位语义）——
  指向当前选中的子消息 id；`0`/缺省 = 无子或按 seq 最大的子兜底。

规范化不变量（对齐 normalizeSession）：话题内**全部**消息都有 `bt`。旧行为零：

- 未规范化的线形话题（所有消息无 `bt`）：可见路径 = seq 序，一切照旧；
- 首次分支操作（编辑提问/重新生成）触发 `normalizeTopicBranches(topicID)`：
  按 seq 序回填 `bt`（父=前一条，子的 `c` 指向后一条，末条 `c=0`），逐条
  merge 写回（保留 extra 既有键 search_summary/answered_by 等）。

## 2. 可见路径：叶子回溯（分页友好）

DeepSeek 从 root 前向走（`getVisibleMessages`）；我们**从叶子反向走 `bt.p`**：

1. 叶子 = 加载集合内「没有兄弟选它」的最新消息：从 seq 降序找第一条
   「未被任何已加载消息的 `bt.c` 指向」的消息；
2. 沿 `bt.p` 反向走到集合边界（父不在已加载集合 → 停，窗口外内容随
   loadOlder 追加自然延长链）；
3. 反转得可见路径。

选择指针变更（`bt.c` 改写）只发生在可见路径上的父消息 → 父必在已加载集合，
无需全量。切版本后保持「每条分支各自的选择」（`bt.c` 挂父，切走不丢）。

## 3. Go 侧（internal/chat + internal/app）

新增（`internal/chat/branches.go`，纯函数可测）：

- `ParseBT(extra) BT` / `MergeBT(extra, bt) string`：extra JSON 读写合并（保留既有键）。
- `NormalizeNeeded(msgs) bool`：任一消息缺 `bt` 即需规范化。
- `BuildLinearBT(msgs) map[int64]BT`：seq 序线性回填表。
- `VisiblePath(msgs) []Message`：叶子回溯（§2），未规范化话题按 seq 直通。

Store 层新增：

- `UpdateMessageBT(id int64, bt BT) error`：merge 写回单消息 extra。
- `SetSelectedChild(parentID, childID int64) error`：改父消息 `bt.c`（事务内
  校验 child 与 parent 同话题且 `bt.p` 确实指向 parent——防越枝改写）。

App 层（chat_service.go / bindings_chat.go）：

- `runChatStreamPlain` 与 `appendChatExchange` 分支感知：话题规范化后，
  落库的 user/assistant 行写入 `bt`（user.p=路径叶/指定父，assistant.p=userID，
  父行 `c` 指向新行）。
- 新绑定三个：
  - `ChatStreamPlainAt(topicID, parentMsgID, message, mode, search, thinking, force)`：
    编辑提问——在 parent 下挂新 [user, assistant] 分支（parent 的 `c` 指新 user）；
  - `ChatRegenerate(topicID, questionMsgID, mode, search, thinking, force)`：
    在问题下挂新 assistant 版本（不落新 user 行）；
  - `ChatSelectVersion(topicID, msgID)`：改父 `bt.c`。
- 历史装配分支感知：`plainChatHistoryMessages` 装载前走可见路径过滤——
  未规范化话题直通（现状），规范化话题只进路径上的轮次；
  `ChatStreamPlainAt` 装配到 parent 截止；`ChatRegenerate` 装配到 question 截止
  （question 本身作为末条 user 重发）。

## 4. 前端（frontend/src）

- `pages/chat/branches.ts`：branches.ts 移植（`bt` 类型别名 + buildVisiblePath
  叶子回溯 + buildVersions 一次遍历兄弟分组，未变复用旧对象）。
- `BranchSwitcher` 组件：`‹ i/n ›`，busy 禁用；挂用户行与助手行 actions 区；
  切换后锚点恢复滚动（版本切换不跳屏，DeepSeek 同款；v1 可后置）。
- ChatPage：
  - 渲染前 `buildVisiblePath(messages)`；`extra.bt` 缺省话题直通；
  - 用户行加「编辑」钮 → 行内 textarea → 提交走 `ChatStreamPlainAt`；
  - 助手行加「重新生成」钮 → `ChatRegenerate`；
  - 流式行仍走既有 streamKey/streamText 通道（新行流式协议不变）。
- 分页：loadTopic/loadOlder 不变（拉的是含分支的全集），展示层过滤。
  hasMore 按 DB 全集计数——含隐藏分支时可能「翻页翻出不可见行」，v1 接受，
  台账记档。

## 5. 明确不做（本批）

- 续写 continue：引擎 prefill 支持度未验证（OpenAI 兼容 API assistant 末条
  prefill 各引擎行为不一），挂拍板池；
- 分支删除/剪枝；树形多根 UI；跨页锚点恢复滚动的完整版（v1 只做切版本
  不跳屏的 anchor 补偿，视实现成本可后置）。

## 6. 风险与回滚

- extra 被第三方消费方（sin 插图 extra 映射、answered_by）共享：`bt` 键
  merge 写入不触碰其他键；UpdateMessageExtra（整字段覆盖）调用方盘点过
  （sin 板块已退役，残留调用在测试里）。
- 规范化写放大：每话题一次 O(n) UPDATE，n≤ 千级，一次性可接受。
- 回滚：`bt` 键无人消费时可见路径计算完全跳过——回滚即删前端走 paths 分支。
