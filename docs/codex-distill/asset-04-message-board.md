# 资产 04 · 多智能体消息板（channel / thread / post / subscription）

> 源码：`codex-rs/message-board/`（19 个实现文件：`api.rs` 8.5 KB / `types.rs` 3.4 KB / `permissions.rs` 3 KB /
> `in_memory.rs` 15.8 KB / `local/*` 22 KB + 查询层 12.5 KB / `tools/spec.rs` 7.8 KB）
> 与 `codex-rs/ext/agent-message-board/`（扩展装配）、`core/src/agent_message_board.rs`（装配入口）。
> 二进制侧证据：工具名字面量 `create_channel / get_channels / list_threads / search_posts / read_thread /
> read_post / subscribe / unsubscribe / post` 在 `evidence/strings.txt` 与模型目录字段区同时出现。

---

## 一、数据结构（`types.rs` 原文结构）

```rust
PostMetadata { message_id: Uuid, channel_name: String, author: AgentPath, thread_id: Uuid, created_at }
PostPreview   { ..PostMetadata, text_preview, n_chars, truncated }
PostContent   { ..PostMetadata, text, n_chars, next_offset_chars }   // 分页读正文
ThreadSummary { thread_id, root_post: Preview, reply_count, last_activity_at, latest_reply: Option<Preview> }
ChannelSummary{ permissions: Option<ChannelPermissions>, channel_name, description: Option<String>,
                created_at, created_by: AgentPath, message_count, last_message_id: Option<Uuid> }
ChannelPage   { channels: Page<ChannelSummary>, board_permissions: Option<BoardPermissions> }
ThreadPage    { root_post: Preview, replies: Page<PostPreview> }
SubscriptionState { channel_name, thread_id: Option<Uuid>, target_agent: AgentPath, enabled, last_message_id }
```

**四个设计要点**

1. **预览与正文分离**：列表只给 `PostPreview`（截断 + `n_chars` 标注截了多少），
   要正文用 `read_post` 拿 `PostContent`，并按 **`next_offset_chars`（字符偏移，不是字节）** 续读。
2. **线程是一等对象**：帖子必有 `thread_id`，`ThreadSummary` 携带根帖预览 + 回复数 + 最近回复。
3. **分页在序列化时才补元数据**：`Page<T>` 只存 `results` + `next_cursor`，
   序列化时补 `n_returned` / `has_more`——模型不需要自己判断「还有没有」。
4. **作者是 `AgentPath`**：不是 user/assistant 二值，而是多智能体路径——这是它能把「谁说的」表达清楚的前提。

## 二、权限模型（`permissions.rs`）

```rust
BoardPermissions   { create_channel, edit_metadata, manage_permissions }
ChannelPermissions { read, post, reply, edit_metadata }
```

注释写明两条纪律，值得直接照抄：

- **「Effective actions，不是 ACL 定义」**：返回的是「当前调用者能做什么」，不是权限矩阵；
- **`reply` 不蕴含 `post`**：能在帖子里回复，不等于有权新开线程——权限不可越级推导。

另外 `PermissionDenied` 的序列化很讲究：`{code: "permission_denied", message, action?, resource?}`，
并通过 `model_response(budget)` **保证错误 JSON 塞进预算**——起始 limit 512 字符，超预算就**折半重试**，
并对被截字段置 `truncated: true`，最小预算取 `budget.min(768)`；绕不开就返回可读的结构化错误而不是裸 panic。
**这个「拒绝也要装进预算」的做法，比大多数实现都认真。**

## 三、API 面（`api.rs`）

```
identity()
create_channel(request)            list_channels(query, recent_first, limit, cursor)
post(caller, request)              append(...)                     // 追加到已有线程
list_threads(channel, sort, recent_first, limit, cursor, max_chars_per_post)
search_posts(channel, query, limit, cursor)
read_thread(thread_id, ...)        read_post(message_id, offset_chars, limit)
set_subscription(channel, thread?, target_agent, enabled)
```

工具面（`tools/spec.rs`）把这 8 个能力直接暴露给模型：`create_channel / get_channels / list_threads /
search_posts / read_thread / read_post / subscribe / unsubscribe / post`——**能力面 = 工具面**，没有额外抽象层。

## 四、订阅语义

`SubscriptionState{channel_name, thread_id?, target_agent, enabled, last_message_id}`：
订阅是「某个 agent 对某频道/某线程」的开关，并记住**最后读到的消息 ID**，
因此新消息判定是「ID 之后的增量」，不是「时间窗内」。

## 五、与 gaea 众神殿的对照

| 维度 | codex 消息板 | gaea 众神殿（`internal/pantheon`，60 文件） | 判断 |
|---|---|---|---|
| 协作形态 | **异步**：频道/线程/订阅/回放 | **同步四席**：一次并跑、共享窗口、分歧台账 | 形态不同，不是优劣 |
| 发言归属 | `AgentPath` + 频道 + 线程 | 席位 ID + 观点记录 | gaea 够用 |
| 增量消费 | `last_message_id` 指针 | 无（一次性汇总） | codex 更强 |
| 分页与预算 | 预览/正文分离 + 字符偏移续读 + 错误预算化 | 无 | **值得借** |

## 六、三档落地建议（按需求验证结果选）

**A 档（只借结构，1–2 天）**：把众神殿的观点/分歧记录换成「预览 + 正文」两段结构
（列表给预览与 `n_chars`，点开才取正文），并给分页补 `n_returned/has_more`。
> 收益：四席长发言不再一次性灌进上下文；前端渲染也更快。

**B 档（加增量消费，3–5 天）**：引入 `last_message_id` 指针 + `enabled` 订阅开关，
让「某席位只看某频道的新消息」成为可能；配合 `search_posts` 做历史检索。
> 收益：长会话里每个席位不必重读全部历史。

**C 档（完整异步协作，2 周+）**：频道/线程/订阅全套 + 回放。
> **只在真实需求出现时做**：如果 gaea 的多智能体一直是「一次性并跑四席出结论」，异步消息板属于过度设计。
> 判断标准：是否出现过「席位 A 的结论需要等席位 B 后续补充」的场景。

## 七、不建议照搬

- **完整 ACL 后端**：`BoardPermissions/ChannelPermissions` 的服务端计算在本地单用户应用里没有对手方，
  保留「拒绝要结构化且进预算」这一条即可；
- **UUID + cursor 分页**：gaea 的会话数据量级用 offset 就够，UUID 主键可以换成现有 ID 体系。

## 八、验收（A 档即可断言）

- 众神殿一次四席发言后，上下文里只出现预览（每条 ≤ N 字符）+ `n_chars`，正文需显式读取；
- 列表返回 `n_returned` / `has_more` 与实际条数一致；
- 权限拒绝的返回是结构化 JSON（`code/message/action/resource`），且在预算内（超长自动折半 + `truncated` 标记）。
