# dsh-ops 蒸馏报告（0.5.1，MIT，T-Auto/dsh-ops）· 2026-11-10

> 纯调研未动刀。源码=`C:\AI\dsh\home\profiles\web\node_modules\dsh-ops`（Web profile 装机版，
> 上游 github.com/T-Auto/dsh-ops）。两路并行蒸馏：机制线（lib/ JS 宿主 3.1M）+文档内核线
> （docs/ 121 篇中文文档+native/ crates 结构）。所有结论供拍板，动刀须点名。

## 1. 产品定位

给 DSH 增加**有界文件与文档工具**：「ops = operations，不是运维产品」。核心动机是
**输出字节（token）节省与有界性**而非纯速度（BM-01：同场景 5,321B vs 内建 8,883B=40.1%↓；
BM-03 工具 schema 仅 241 tokens）。架构=JS 宿主适配层 + Rust 原生内核（一次性子进程，
stdin 2MiB JSON 请求 → stdout 128KiB JSON 结果，30s 超时，无 daemon）+ Windows 受限
Bash 供应链（Niubash/MSYS PE 导入表重定向）。分层原则「执行共享，投影分开」（Native 与
PTC 双投影进同一内核）。

**工具面 20 个 `ops_*`**：文件 15（inspect/read_files 批量 32/grep/glob/stat/diff/
write_file/edit_file/copy/move/remove/replace 跨文件批量替换/read_table/read_docx/read_pdf）
+ PTC 3（batch plan-resume 两阶段批量、result ref:// 结果存储、load_sdk）+ shell 1
（ops_bash 前台 only 明确无后台 job）+ 发现 1（discover_tools）。

## 2. 机制蒸馏（八件最有价值的机制）

1. **Revision CAS 写契约**：全部写操作强制 `expectedRevision`（SHA-256 或 "absent"
   create-only），CAS 原子发布；edit_file 1-100 补丁同原始字节唯一非重叠（file.js:15,54-58）。
   防「基于陈旧读的写」——行号不等于文件身份（设计决策 #424 全部不做，保留 CAS）。
2. **动态工具目录（resident/discoverable 两级）**：常驻仅 4-5 个，其余 discoverable
   初始不挂载；`discover_tools` 词法匹配→**下一模型步才激活**（可见性≠授权）；
   替换映射 read→inspect/glob/grep（discovery/controller.js:23-200）。省常驻 schema。
3. **结果预算体系**：并发泳道（2 并发/8 排队/CALL_LIMIT）+ 输出 64-128KiB cap +
   超限转 `ref://` 结果存储（≤10MiB/快照、TTL≤7 天、LRU）+ 有界开头预览 + 分页游标
   （scanComplete/pageComplete/STALE 语义，游标绑定会话+权限+观察状态）（结果预算.md）。
4. **失败即值**：错误为结构化值 `{code, phase, retryable, effectState, nextAction}`
   不抛异常；effectState=unknown（输出超限/超时）时 nextAction 明示「先检查后置条件」
   ——**副作用类工具的诚实失败语义**（file.js:113）。
5. **凭据围栏**：.ssh/.aws/.kube/.netrc/.env/id_rsa path-后缀+名字正则双轨拒绝，
   **连 danger-full-access 也拒**；遍历以 deny_roots 跳过（credential-policy.js:29-45）。
6. **会话私有 temp + TOCTOU 双校验**：session temp 按会话 id 哈希 0700 懒建绝不跟随
   链接；spawn 前后各 resolve 一次 policy 并全字段比对（AUTHORITY_CHANGED 即中止）。
7. **批处理 plan/resume**：plan=零副作用审计 manifest（≤512 操作）→ resume 只按 id
   执行既有 plan（防悄悄换输入）；副作用前持久化 unknown 状态到审计库；**无跨文件
   事务、无自动回滚**——保守恢复（native-ptc.md）。
8. **基准门禁文化**：五门禁（字节预算/CAS 拒绝/schema token/渐进发现时序/凭据围栏）
   进 opt-in CI；pinned 基线 blob 逐一校验；证据 JSON 拒绝覆盖；「timings never decide
   a CI gate」；历史失败判定不被后续决策改写。

## 3. 同构等价表（不立项）

| dsh-ops | gaea 对应 | 判定 |
|---|---|---|
| ops_* 文件工具 15 | 内建 read/write/edit/ls/grep | 同构（gaea 缺 CAS/富格式读，见候选池） |
| ops_bash（前台 only） | bash（有 jobs 后台） | 同构，gaea 更广 |
| ops_batch plan/resume | jobs 任务中心（持久/断窗续跑/取消/重试）+DAG | 强同构，gaea 覆盖更广 |
| ops-index 名字索引+三平台 watcher（默认关+Ready 门槛） | fileindex 语义索引（每 10 分钟增量） | gaea 更激进；dsh 特色是 AB 证据文化与默认关 |
| ops-documents 有界只读观察 | docmd 解析+format_convert | 目标相反（观察 vs 转换），不复刻 |
| 下载守卫 / 审阅快照 CLI | PublishGuard（名近义远）/无 | 下载守卫按需可参考，非本轮 |
| publication/（工具发布进会话表面） | PublishGuard | **名字撞车语义不同**，防混淆 |

## 4. 候选池（gaea 缺位，动刀须点名）

- **A1 结果预算体系**（推荐首位）：工具回执字节 cap+溢出转引用存储（gaea 版可落
  `.gaea/work/results/`，TTL+LRU+会话绑定）+分页游标+STALE 语义。直击 gaea 工具大
  输出爆上下文（zcode 4MB cap 老病同族）；BM-01 已证 40% 字节节省。影响高/代价中。
- **A2 动态工具目录**：办公 agent 20 工具 schema 常驻（含 office 工具族新增）→
  resident/discoverable 两级+下步激活。与阶段九「上下文即预算」哲学同源；BM-03
  241 tokens 证据。影响中高/代价中（动 boot 装配+agent schema 导出面）。
- **A3 revision CAS**：write_file/edit_file 增 expectedRevision（stat.sha256）可选参数
  +冲突拒绝。与证据链 BaselinePath 互补；防多执行体同树 stale-Edit（双会话合流坑在册）。
  影响中/代价低-中。
- **B1 凭据围栏**：read/grep/bash 前置 path-后缀+名字正则 deny_roots（.env/id_rsa/
  .ssh 等），连 danger 面也拒。小而硬的安全块，影响中/代价低。
- **B2 失败即值**：工具错误结构化 `{code,retryable,effectState,nextAction}`（至少 bash/
  写类工具），effectState=unknown 语义进 agent 纪律提示。影响中/代价低-中。
- **B3 基准门禁文化**（观察池）：字节预算门禁+证据不可覆盖——呼应 P0 门禁线程，暂不立项。
- **C 不抄**：Rust 原生内核重写（Go 内建够用，重写=L 成本）；Niubash/MSYS PE 补丁
  供应链；PTC run_code SDK（DSH 宿主机制）；全文索引/grep 走索引（上游自己都没开）；
  跨文件事务/自动回滚（上游明确不做，gaea DAG+Journal 已覆盖该域）。

## 5. 上游诚实文化摘录（值得继承的工作纪律）

- 「既有测试只证明它们真实断言的行为，不能用来宣称未实现的功能已通过未来验收矩阵」
- 索引 AB 基准结论=「无法测出收益→保持默认关」——**负结果也入档、不用前景故事顶替**；
  速度数字只作证据从不用作 CI 判决
- 磁盘回收「必须有归属证据，绝不按目录名或修改时间猜测」
- 设计决策文档把「不做+重启所需证据」写透（#424/#459/#534 有条件 no-go 各配重启门槛）

## 6. 关键源锚（深读入口）

- 机制：`lib/capabilities/registry.js`（目录）、`lib/authority-paths.js`（围栏）、
  `lib/execution/protocol.js`（envelope）、`lib/kernel/call-budget.js`、
  `lib/discovery/controller.js`、`lib/bash-ipc.js`（MSYS 供应链）
- 文档：`docs/{内部架构,原生内核,结果预算,搜索覆盖范围,文件批量操作,索引AB基准,
  基准门禁,设计决策-20261102,磁盘与缓存,动态工具目录}.md`
- 上游钉版：0.5.1（本机 Web profile 装机版）；基准基线 pinned `4b51b63`（0.3.1）
