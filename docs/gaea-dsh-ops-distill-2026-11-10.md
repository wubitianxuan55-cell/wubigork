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
  **✅ 2026-11-10 落地销项（四刀）**：`tool.Failure` 结构化错误+信封扩
  retryable/effect_state/next_action（纯加法）+新码 conflict；bash 超时/取消/
  非零退出（effect 恒 unknown+退出码结构化）、revision CAS 三冲突=conflict+
  none、凭据围栏=denied 分类接线；executeOne 收口按字段渲染+写族默认
  unknown（复用 IsDeliverableTool 账本）+**内层失败信封塌缩**（bash json
  非零退出/browser validation「成功包着失败」根修）；**顺带根修三处盲计**
  （steer/complete_step/回合摘要按裸 "error:" 前缀数失败，但普通工具失败自
  V8.9 起就是信封 JSON——真实失败从未被数到）；纪律提示两句（系统提示原则
  节+compact 描述）。未引入 phase 字段（gaea 无相位概念，如实记档）。
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

## 7. Windows 适配专项（2026-11-10 补刀，第三路蒸馏）

### 7.1 受限 Bash 供应链（Windows 特有，必要恶级别）

- **选壳 fail-closed**：受限模式只认 Niubash（niu.exe）或 pinned MSYS；系统 Git Bash/
  自定义 bashPath 直接 `EXECUTOR_UNAVAILABLE`——「审批永远修不好一个不兼容的受限
  运行时」（shells/resolve.js:168-193、bash-ipc.js:51-58）。
- **MSYS PE 导入表重定向**（bash-pe.js:2-48）：手写 PE 解析把 import name 表的
  `kernel32.dll`/`ntdll.dll` 字符串**原位等长改写**为 `opsipc.dll`/`opsnt.dll`（不改
  IAT 不改指令→sha256 供应链校验仍成立）；代理 DLL 只拦 `CreateNamedPipeA`/
  `NtCreateNamedPipeFile`/`NtCreateSection`——MSYS 的 named-pipe IPC 在受限 Low 令牌
  沙箱里起不来的唯一解法（#389）。**Niubash 与 MSYS 是两个东西**：Niu=独立原生受限
  bash（默认），winuxcmd 153 个命令名=一个 exe 的硬链接 argv0 faces（零数据拷贝）。
- **代际缓存**（Windows 运行中 exe 不可覆盖→rename-only 发布）：stage→校验→manifest
  →rename 发布；坏代改名隔离 `.bad-*` 绝不覆盖；`renameWithRetry` [100..1000]ms
  预算背后是杀软持句柄实测 1.6s 窗口（#352）；lease 锁 mkdir 原子**从不偷锁**，
  「rename 成功或 PID 死亡不证明 shell 后代已退出」是第一公理。
- **执行树**：同卷 hardlink 克隆（786MiB 零拷贝）→seal（一遍 `icacls /T /deny`，
  313 文件一次进程创建 vs 每文件数千 ms）→每次 spawn 前 re-seal+stat-only verify
  （旧版每调用 1.85GiB SHA-256 已去掉）。

### 7.2 文件系统适配矩阵

| Windows 差异 | 处理 |
|---|---|
| NTFS 名字 WTF-8 非法代理项 | 可逆编码+折叠键（NFC∘lower∘NFD）只作匹配、**原始字节作身份**（大小写不敏感用折叠表达） |
| 长路径 | 不主动加 `\?\`；主干遍历用 cap-std 句柄相对打开绕开 MAX_PATH |
| junction/symlink | **一线拒绝**（REPARSE_POINT 0x400 判 link，明说含 junction）；junction 删除只摘条目不递归 |
| READONLY/ADS | 属性白名单 capture（EFS/压缩/sparse→fail-closed）；命名流 `METADATA_UNSUPPORTED`、只收 `::$DATA`；readonly 删除先验 nlink==1 再清位（绝不 chmod 共享硬链接） |
| 共享冲突/锁 | per-name sidecar 锁；**原始错误码分类**：raw 1/50/120→`LOCK_UNSUPPORTED` vs 32/33→`REVISION_CONFLICT`（一行代码避免把「FS 不支持锁」误报成「别人在写」） |
| 0700 落地 | Windows 无 `open(O_CREAT|0700)` 等价物→**NtCreateFile 创建即带 protected DACL**（字节落地前私有，创建后 SDDL 逐字验证失败即删） |
| 8.3 名/大小写绕过 | 凭证围栏 canonical+lowercase 后缀匹配，8.3 拼写缺失叶子也命中（防路径等价拼写绕过 deny 列表） |
| 原子发布 | replace=rename+DACL capture/restore（AR→AI 控制位转换）；失败标 `effect_state=applied`「文件已发布，别盲目重试」 |

### 7.3 Windows 索引栈 W1-W4（默认关的克制）

W1=MFT 冷枚举（FSCTL_ENUM_USN_DATA，需 Administrator；ReFS 128-bit id 放不进
64-bit FileKey→整源 Unsupported）→W2=USN journal 有界重放（2²⁰ 条/8 轮）+
CursorStore 判 replay/rescan（**绝不主动创建 journal**——副作用纪律）→W3=raw FSCTL
fallback（默认不用，usn-journal-rs 表达力缺口时才降级）→W4=硬链接对账（**不需要
Administrator**；「不完整枚举绝不当作空集应用」会误删仍有链接的文件）。默认关的
原因：MFT crate 会把 COM 拖进 worker 导入表（NullSource 不链接）+无 watcher 则
state 永不 Ready——四层切片全部前置服务未来 watcher，纪律可敬但今天零用户价值。

### 7.4 WSL 委托（opt-in 只读）

`\wsl.localhost` 工作区 + `DSH_OPS_WSL_READS=1` → glob/grep/read_files 走
`wsl.exe --exec <linux worker>`（减少 9p 重复元数据访问）；每次调用重解析盘符挂载
（「mount mappings 是活权威不是缓存」）+sha256 核对 Linux worker 字节；存储引用
绝不内联（Linux worker 永不获得结果存储授权）；游标 `w1:` 绑定 distro+摘要，跨
后端续页明确 SEARCH_STALE 不混快照域。

### 7.5 得失评与 gaea 可借鉴清单

**必要恶**：patchMsys（不改指令字节才让 sha256 校验成立）、rename-only 代际发布、
icacls 树级 deny+READONLY（Windows 无 immutable 位）、NtCreateFile 创建即 DACL、
reparse 全线拒绝、凭证路径 8.3/大小写兼容。
**超额工程**：W3 fallback（交付时未接线）、整个索引栈前置（今天零用户价值）、
adopt 旧代逻辑、每调用三重 verify（文档自认只防意外漂移）。
**gaea 可直接借鉴**（Go 平移）：①WTF-8+「折叠键匹配/原始字节身份」二分；②索引
失败三态 `Unsupported/Unavailable/Incomplete(kind)` +「不完整枚举绝不当作空集」；
③Windows 原始错误码分类表（1/50/120 vs 32/33）；④renameWithRetry 有界预算+实测
数据引用格式（#352）+「cleanup 失败永不覆盖业务失败分类」；⑤失败分类学元模式
（每 stage 带 {retryable, remedy}）；⑥argv0 faces+hardlink 克隆（os.Link 等价）；
⑦NtCreateFile 创建即 DACL 模式（若做私有缓存）；⑧USN cursor 17 字节定长+无副作用
纪律。另注：**「哈希相同不代表安全元数据相同」**——静态哈希比对不查令牌/完整性
标签/杀软决定，这个认知对 gaea 装机校验同样适用。

## 8. 第二轮深审（2026-11-10 晚，用户令重审；三路并行，纯调研）

三路：lib/ 未挖子系统逐个读（第一轮只覆盖 §6 锚定的 6 个文件）、native/crates
算法级深读（第一轮只盘结构）、docs/ 118 md 全目录扫描（第一轮只蒸 10 篇）。
与 §2/§7 已覆盖清单核对无重复。**全局发现：lib 根下 scoped-ownership 等 6 个
文件全是 2 行转发 shim，真身在 publication/ 与 execution/——机制密度最高处。**

**文档过时段警示（5 条，蒸馏以较新篇为准）**：常见问题.md 的锁 sidecar
「从不删除」（#319 已改为持有者释放时删）、macOS /tmp 整体拒绝（父链规范化后
后代可用）、shellPolicy deny-host-shell（遗留键）、使用指南 §0.5.0 cwd 回退
（0.5 面向宿主 0.2.2 不回退）、native-ptc「24 小时过期」（新引用会话期）。

### 8.1 输出与读取契约（Rust 线，密度最高）

- **序列化字节预算三件套**：预算按**转义后 JSON 字节**计（`"`/`\n` 2B、
  <0x20 6B——32MiB 文件曾炸 200MB stdout 的 NUL 6 倍乘数，file_output.rs:5-11）；
  **信封最坏预留先行**（先构造含最长标志的完整信封再逐 item 扣 room，「不存在
  事后才发现信封超大」，:41-112）；**行内截断+hex 恢复**（装不下的行给
  bytesTotal/bytesOmitted/truncatedInLine+nextAction=按 16 字节行分页续读，
  nextOffset 回退到被切行——截断点对模型可见可恢复不丢后缀，:78-110）。
  gaea 对位：truncateToolOutput 按原文字节截断，未计 JSON 转义成本，截断点
  不可恢复——**缺位**。
- **自校验续读游标**：token=`s1:{binding}:{snapshot}:{next}:{digest}`，binding
  覆盖查询参数+root dev/ino+权限集，快照哈希 feed 每个访问路径的
  (path,dev,ino,len,mtime,is_dir)（scan.rs:119-135、paging.rs:28-114）；
  **续读不授予任何权限**、**不完全遍历不发 continuation**（paging.rs:210-226）；
  walk(s1)/索引(s2) 快照域隔离互不认。gaea 对位：A1' grep offset 游标无绑定
  无自校验——**缺位（A1' 直接补强）**。
- **三态跳过+完备性三元组**：excluded（选择不看）/gap（没看到，计 failed 与否
  分档）两义分开（scan.rs:108-115）；complete/scanComplete/pageComplete 三值
  分离+读前后 (size,mtime) 比对→FILE_CHANGED（pdf.rs:21-98）——「结果是下界
  还是全集、哪部分没看」模型可判。**缺位**。
- **read_files 聚合帽+双层续读**：1-32 文件共用 96KiB−最坏信封总帽逐文件扣
  room，nextIndex（文件序）+每文件 nextOffset 双层恢复（file_many.rs:19-49）；
  错误文件也是条目不整批失败（:88-102）。
- **防伪行号**：只有物理 LF 是行界（CRLF 一行、裸 CR 不计）；正文 JSON 式可逆
  转义——源文本无法伪造行号/文件头。gaea readfile 42→ 前缀+0-based，无防伪。
- **glob/grep 增量**：glob pattern 数组+`!` 排除+modified 稳定排序；grep
  definitions 模式（词法声明行分类）、exactTotal=false 早退只给 matchedObserved
  下界（row-limit 截断明示「总数请重查」）、上下文行零行表字节切片定位。

### 8.2 权威与生命周期（JS 线）

- **作用域所有权仲裁**（publication/ownership.js:4-52）：scope 链**最近者赢**、
  同层最后挂载者赢、卸载自动恢复前一行权威、重入计数防自触发——gaea 双执行体
  同树权威问题的直接对位（stale-Edit CAS 之外的编排层）。**缺位**。
- **超代对账**（publication/reconciliation.js:2-52）：新一轮先 cancel 旧
  AbortController；wait 双续延观察迟到 rejection（既不发布也不 unhandled）；
  drain 循环到静止防「drain 期间又来一轮」。**缺位**。
- **工具变更归属传播**（publication/changes.js:49-63+replacements.js:185-215）：
  每条 tools/change 归因到作用域，受影响者重算、欠账不清、突发合并——O(live)
  压成 O(affected)。**缺位**。
- **多等待点权限再校验**（execution/jobs.js:52-180）：admission() 在**每个
  await 后**重查（≥5 处）；沙箱准备后 sameAuthority 重比对；spawned 标志分界
  RESULT_UNKNOWN（效应不明）vs EXECUTOR_UNAVAILABLE（零效应）。
- **跨风味 workdir 锚定**（session-shells.js:56-106）：会话工作区拼写决定
  Windows/POSIX 规则；无盘 POSIX 绝对/跨盘 C:child/空占位各有专门拒绝——
  拒绝一切借宿主进程当前盘解析的输入（诱饵目录陷阱）。
- **权限模型全景 L0-L8**（docs 综合重建）：宿主三档为唯一授权源；固定文件操作
  是可信围栏非代码隔离；read-only 连 dryRun 也拒；逐命令 allowed-once+批准后/
  沙箱准备后双重复核；降权中止在途 worker；grant 撤销即否决已捕获 toolSet；
  可见性≠授权（restrict 管不到自有注册）。gaea 缺 L4-L6 动态复核层。
- **清理纪律三件**：清理债务账本（首个存储句柄发出**之前**先落持久记录，清理
  失败推迟不遗忘，kernel/result-cleanup-registry.js:1-90）；白名单自底向上 GC
  （名单外一个未知名字跳过整个 root、rmdir 自底向上让并发写者赢，
  result-cache-gc.js:1-50）；按归属证据回收（绝不按名字/mtime/死 PID）。

### 8.3 写与批量（Rust 线）

- **bind-before-publish sidecar 锁**（file_update_lock.rs:46-154）：锁住
  sidecar 后**重开名字**比对 (dev,ino)，Bound 才发布、Moved 丢弃重试≤8 次；
  释放时持锁验证身份再 unlink（防删别人重建的锁）；锁名带点前缀天然被
  glob/grep 跳过。
- **结构化树哈希**（paths.rs:207-287）：类型标签+文件字节+**排序后**子项
  (len,name)——同树稳定，batch 执行前 revalidate 用。
- **batch 写前 unknown WAL+resume 只验不重放**（ops-batch lib.rs:204-286）：
  每项碰文件系统前先 durable 落 unknown，崩溃恢复不可能误当 pending 重放；
  resume 只跑 postcondition（dest 树哈希==计划 revision）。
- **move=hardlink 原子 no-clobber+双侧 CAS**，跨设备报 CROSS_DEVICE 不降级
  copy-delete；**发布后 fsync 失败=SYNC_FAILED+effect_state=applied**「文件
  已发布，别重复，去验证」（file.rs:604-623）——绝不把已生效写报成 none。
- **replace 全计划溢出预检**：每文件 32MiB 上限 dryRun 也算，任一超限整次
  OUTPUT_LIMIT **零变更**；多文件=一串文件级原子发布非事务（A 成 B 败保留 A
  回执+pending）；changed 计替换次数非文件数。

### 8.4 有界解析（Rust 线，与「整文件解析再截断」的差异）

- **ZIP preflight 先于解压库**（archive.rs:42-147）：尾部 65557 窗口反找 EOCD
  且注释长度精确闭合，拒多 EOCD/ZIP64/多盘/加密/重复名/重叠 entry——「ZIP
  metadata 先于库从攻击者可控计数分配内存而 bounded」。
- **DOCX SAX 窗口物化**（docx.rs:154-192）：End("p") 即 push、窗口外只计数
  不进 items（解析即分页，内存只有当前页）；单段 1MiB 帽、10 万段帽。
- **PDF 严格子集全定长**：仅 classic xref 单 trailer，拒 Prev/XRefStm/Encrypt/
  ObjStm；直接 Length 拒 indirect；引用链≤64 防环；16MiB/32 页全定长；Flate
  单滤镜+zlib 终结符+total_in==raw.len() 拒尾部数据。
- **CSV 流式状态机**：逐字符严格解码+流式 sha256 revision；恰 MAX_ROWS 行也
  能证明 EOF（「One bounded lookahead proves EOF」）。
- **XML 走界**：深 128/属性 64/命名空间 512/30s 逐事件；DTD 与自定义实体拒。

### 8.5 错误码全表与配置面（docs 线）

- **错误码按层全表**（gaea 现 9 粗码对比）：OUTPUT_LIMIT/PATH_CHANGED（枚举中
  目录身份变化不给部分结果）/LOCK_UNSUPPORTED（曾被误报 REVISION_CONFLICT 的
  教训）/ERR_INVALID_ENCODING/EXIT_CODE（重试策略 after-input-change）/
  RESULT_PARTIAL/RESULT_UNKNOWN/AUTHORITY_CHANGED/SEARCH_STALE；EXECUTOR_
  UNAVAILABLE 带子阶段（staging 失败重试一次 vs IPC 不匹配重试无效）——**子码
  直接回答「重试有没有用」**。effect_state 除 none/unknown 外还有 **applied**
  （发布已生效，验证而非重试）。
- **参数纪律**：未知键拒绝并列出可接受键（不静默忽略）；可选字段空占位值
  （""/null/[]）worker 运行前剥离，必填/replacement:""/false/0 原样。
- **配置面**：迁移内存优先生效+只写 patch 层（永不整树写回——会把其他插件行
  烘进根文件）+失败降级内存迁移下个激活重试；设置保存带 revision CAS 冲突保
  草稿；退役行守卫防双实例。

### 8.6 候选池第二波（D 系，动刀须点名；gaea 侧对位断言已核）

- **D1 输出预算三件套**（推荐首位）：转义成本预算+信封最坏预留+行内截断
  hex 恢复。纯函数~150 行，read/grep/多文件读全受益，消灭「截断了还超限」与
  「截断点不可恢复」两病。影响高/代价低。
  **✅ 2026-11-10 落地销项（刀3）**：EscapeJSONLen（与 encoding/json 性质
  测试逐例相等钉死）+TruncateToEscapedBudget 迭代收敛+executeOne payload
  序列化前预算截断（信封恒完整合法 JSON——旧口径序列化后截断必劈坏 JSON
  根修）+信封 B2 三字段挪 Data 前+分页读压缩保头保尾标记。**如实缩小：
  不做 hex 视图**（gaea 无此视图，截断标记指 offset/cursor/read_spill）。
- **D2 自校验续读游标**：A1' 补强——binding（参数+root 身份+权限）+快照哈希
  +不完全遍历不发 continuation+截断原因枚举各配恢复文案。防翻页漂移与
  「续读变提权」。影响中高/代价低-中。
  **✅ 2026-11-10 落地销项（刀2，含前置刀0 两生产 bug 根修）**：grep
  `g1.<bind>.<snap>.<next>.<mac>` token（`.` 分隔防 :digits: 误判）；双快照
  （prefix 边界校验/walk 全程发射）；树变=conflict 重跑自愈、参数变=validation
  拒、缺口/取消不发。刀0 根修=compressGrep 洗掉控制行（游标在生产路径从未
  被模型看到，测试直调工具假绿）+grep 扫进 .gaea/sessions
  （wspath.IsGaeaHostNoiseRel 新谓词，exports/play 保留）+toolcache 键补
  limit。**如实缩小：不加 SessionID 绑定**（进程级 HMAC，重启失效如实过期）。
- **D3 错误码子层+applied+after-input-change**：B2 补强——9 粗码下挂子码，
  effect_state 补 applied，EXECUTOR 子阶段回答「重试有没有用」。影响中/代价低。
  **✅ 2026-11-10 落地销项（刀4）**：信封顶层 subcode+EffectApplied；applied
  真实落点仅 2 处（move_file 删源失败 / xlsx_apply 预览失败降级备注——盘查
  实证不虚构）；原子写族默认 effect=none 精确化；bash 守卫 validation+none、
  SANDBOX_UNAVAILABLE/START_FAILED/TIMEOUT/CANCELLED/EXIT_NONZERO 子码；
  bash json 模式失败改走错误通道（B2 塌缩注释不实修正）。**如实缩小：不加
  retry_policy 字段**（after-input-change 语义由 NextAction 文案承载）。
- **D4 跨文件 replace+写锁 sidecar**：gaea 无 replace 无文件级写锁；全计划
  预检零变更+逐文件原子非事务+bind-before-publish 锁协议+树哈希 postcondition。
  影响中/代价中。
  **✅ 2026-11-10 落地销项（三刀）**：①filelock.go 写锁原语（O_EXCL 哨兵
  +进程 token+bind 重核 os.SameFile+release token 防误删+stale 年龄回收；
  sidecar ~\$<base>.gaea.lock 天然被 PublishGuard 剪枝）——gaea 零 OS 锁先例
  下选 O_EXCL（attachment/store.go:231 先例），gaea_task_inbox.go:95 记载的
  「文件锁属另一刀」正式销案；②锁接 edit 族「读→CAS→写」整段+write_file
  原地写半写债收口（atomicWriteEncoded 化，effect 挪 none 组）；③replace
  工具（两阶段：全计划 32MiB 预检零变更/逐文件原子非事务 PARTIAL_PUBLISH
  +effect unknown/changed 计替换次数/发布前锁内计划哈希重核/凭据围栏写向
  跳过）+四表全做+evidence·executeOne·batch·coherence·contextview·前端
  WRITE_TOOL_NAMES 名单全量。**如实缩小：字面量 pattern 非正则/单串 include/
  锁回收年龄阈值（上游 PID liveness）/move_file 不挂锁（双路径形态不同挂
  后续）/不带 caller revision。**坑=锁创建先于 AtomicWrite 的 MkdirAll
  （新目录炸，acquireFileLock 先建父目录修）。
- **D5 三态跳过+完备性三元组+FILE_CHANGED**：搜索/读取诚实性——excluded vs
  gap 分开、complete/scanComplete/pageComplete、读前后 stat 比对。影响中/代价低。
  **✅ 2026-11-10 落地销项（刀1）**：excluded（噪声/围栏/include/二进制）vs
  gap（不可读/超长行 scanner 截断——静默丢同文件余下命中是真 bug）分开记账；
  取消不再伪装完整 no matches（假完备根修）；缺口不发续读指引。**如实缩小：
  完备性二元（scan=complete|partial）不抄三元组**（gaea 单遍全扫无页概念）；
  **read 侧不做 FILE_CHANGED**（单次 ReadFile 即快照，revision 尾标已是漂移
  检测）。
- **D6 清理债务账本**：gaea publish-quarantine 盘查未见清理面（只进不出，
  待装机实测确认增长）；「先记账后放句柄/失败推迟不遗忘/名单外整 root 跳过/
  绝不按名字扫」。gaea spill 为内存库（budget+FIFO+重启清）无此病——如实改判
  不含 spill 腿。影响中/代价低。
  **✅ 2026-11-10 落地销项（刀5）**：Quarantine 先落 ledger.jsonl pending 再
  动文件+终态补录（账本失败不阻断隔离但 stamp 无主永不自动删）；fail-loud
  回显隔离目的地；boot.Build 异步清扫（账本背书+30 天+账本外名字整目录跳过+
  自底向上+失败留账重试）；存量无账本 stamp 不动；wssearch 噪声补齐。
- **D7 作用域权威三件**（双执行体同树第二刀）：所有权仲裁（最近者赢/撤销恢复）
  +超代对账（先取消旧轮/迟到结果不落地）+变更归属传播（O(affected)）。动会话/
  注册面，代价中-高；与 DSH 同树合流场景直接对位。**待点名**。

**观察池（不急）**：ZIP preflight+DOCX SAX 分页（gaea docmd/format_convert
现状待核后再判）；权限动态复核层 L4-L6（gaea 规则授予模型不同，逐命令
allowed-once 要动审批面，须专项）；证据式跨请求读取去重（证据条件苛刻风险大，
排最后）；workdir 锚定；glob 工具；发布重入队列；配置 patch 层纪律。

**同构/不立项补充**：disk 家族登记表（gaea 无此盘面）、download-guard（gaea
无下载引流需求）、review/merge 探测（无对位）、tui 表现层、PTC 侧渐进准入
（A2 已覆盖其镜像）、Windows workspace 准备（DSH 沙箱专属）、definitions
分类器（价值低）、teardown disposer 隔离（通用卫生）。
