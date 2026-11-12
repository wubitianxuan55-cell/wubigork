# gaea Day Log · 2026-11-12 · 办公板块上下文界面升级换代（dsh-context 0.83.x 全量复刻）

> 调研→拍板→动刀全链一天收官。调研报告 docs/gaea-contextview-upgrade-research-2026-11.md。
> 版本 v4.542.0，装机四次全冒烟。代码主体在 feat/context-upgrade 分支，**收官批刀11-15(ea21dac9..d2494ab8)后已随 d20dec1d 常规合并入 main(ff-only 因双侧 cherry-pick 并存不可行；8 处同源冲突解毕，详见 §9 收官批)**。

## 1. 批次结构（三波 + 收尾）

| 波 | 提交 | 内容 |
| --- | --- | --- |
| 主批六刀 | 刀1 e860a75f / 刀2 9f422693（main）+ 刀3 09d712c0 / 刀4 96110cc4 / 刀5 656a59eb / 刀6 88139ab8（分支）| 折叠扩展 / 跨会话洞察投影 / 洞察页 / Context tab 升级 / Fleet 编队页 / 偏好回写 |
| 完善批三刀 | 332976d1（+分批 b1c3d98d 抬号、690034ff Fleet 导入修）| assistant 三拆消费 / 洞察统计行点击过滤 / 趋势悬停 Actual；**定罪刀2 夹具 Ts 硬编码** |
| 真实数据轮 | main e2ba9187 / 分支 c05bf9a4 | estimated 会话 Turns/Steps 恒 0 根修 + ctx_realcheck 真实日志探针工具 |
| 走查轮 | 分支 33b5ff8c+9def3ca8 / main 65b07dae | handleChatTabSelect 四 tab 白名单漏新 tab 根修 + aria-pressed |
| legacy 命令轮 | main e2ba9187（并入上）+ 分支 c05bf9a4 | legacy 日志命令时间线推导（内嵌 tool_calls 即派发事实）|
| 家族流轮（刀10）| main 95c87d9a+ae56bb1f / 分支 1264e3db+fc039a10+48a9c19c | 家族事件合并流：FoldFamilyEvents + GaeaFamilyEvents + Fleet 事件检查完全体 |

## 2. 功能终态（对标 dsh-context 0.83.1 五入口）

- **洞察页**（第五 tab）：KPI 六卡（计费/估算分列）、7 天成对柱图、8 周热力图（点击过滤当日）、
  技能/工具/命令程序三统计（chip 点击过滤会话）、七分类构成环形、会话卡（估算/归档徽标、
  迷你堆叠条、点击 resumeSession）
- **Context tab 升级**：命令执行卡（bash 生命周期三态/程序分布/筛选）、趋势总览带、
  浏览器 DNA 平铺（模型读取顺序编号+头两行收尾）、助手 kind 筛选、七分类消费
- **编队页**（第六 tab）：摘要条（Agent 数/运行中/委派任务/家族消息）、协作泳道、协作任务、
  事件检查完全体（agent/kind 筛选+文本搜索+回报点击下钻）、Agent 网络
- **/context 弹窗与偏好**：既有 ContextPill/Modal；洞察默认范围回写 contextPrefs

## 3. 真实数据验证（双路）

- **HTTP 桥探针**：GAEA_HTTP_PORT+GAEA_HTTP_TOKEN 打真实绑定——洞察 200 返回真实会话卡
  （118 工具调用/真技能 genui·office-edit·xlsx/程序 python×36/缓存 99.76%）
- **ctx_realcheck 探针**（scripts/，//go:build ignore 长期留库）：真实造价工作区
  （C:\AI\bangong\*，每项目一工作区）——4 会话 1.3M 估算 tokens 诚实分列、legacy 命令
  0→78、黄甲 52 份真 transcript 家族流 dispatch/message 正确

## 4. 定罪与根修（走查/探针/测试三层各逮各的）

1. **handleChatTabSelect 四 tab 白名单**（走查逮）：新 tab 被当子代理会话 id，内容回落 chat
2. **estimated 会话 stats.Turns/Steps 恒 0**（真实数据探针逮）：只 usage 路径更新=漏账
3. **legacy 日志命令时间线恒空**（真实数据探针逮）：迁移投影零 tool_dispatch，内嵌
   tool_calls 即派发事实
4. **insights_test 夹具 Ts 硬编码**（完善批自查逮）：entry helper 无视 ts 参数，RangeFilter
   恒红且 commit 时被 tail 截断漏验——「黄金测试全绿」当时失实

## 5. 坑册新增

- `gaea_install.ps1` $SourceDir 硬编码主树 build/bin——分支树构建须 -SourceDir 指产物并先补三 DLL
- unset TMP/TEMP = wails 0xc0000135（须**设**系统默认 %LOCALAPPDATA%\Temp，非清除）
- bash cwd 漂移跑错树（vitest 丢配置假红 / python 编辑落错树 / go test 打错树）——**本日最高频坑，
  五次触发**；特征=vitest setup 0ms+environment 0ms；纪律=跨树操作一律绝对路径
- 双 dev server 并发共享 node_modules/.vite 缓存=deps 混 epoch 双 React 崩（"Cannot read useRef"）——
  二分定罪法：同 commit 异树对照+单服冷启；纯浏览器 dev 默认走 HTTP 桥（404），mock 须 ?mock=1
- gen_bindings 纪律=先 spaceBindings 认领+手写门面委托（生成器已不生成转发）再 -legacy-ts -signatures-ts
- tsc 真闸=tsc -b --force（-p tsconfig.json 漏检）
- RTL rerender 替换整棵树（Provider 不保留）；测试红灯被 tail 截断=验收必须逐行核
- 悬空工作树残壳让 git -C 静默落回主仓——cherry-pick 后必须 git branch --contains 复核落点
- Windows python open() 不认 /c/ 形态路径

## 6. 挂账与拍板池

- **合并挂账**：feat/context-upgrade → main 被另一条线（办公对话输出重设计）16 文件未提交锁住
  （App.tsx/三语言 locales 相交）；树清后 `git merge feat/context-upgrade` 一把收（Go 侧已双树同源）
- **拍板池**：promptPrefix/Suffix（行为级=拼所有会话系统提示词，须拍板）；价格表映射（须内核
  RequestHeader 带 model id 前置）；家族消息流的 legacy 命令推导已销项
- 装机终态：分支头构建，冒烟 200，PID 验证；包=releases/gaea-contextup-20261112-690034ff +
  gaea-contextup2-20261112-332976d1（zip/tar 磁盘不入库）

## 9. 收官批刀11-15 + 合并收口（2026-11-12 下午）

用户令「完整复刻」清池 + 「管它是谁的全部修复好」清测试债。提交链 ea21dac9(刀11-15)→
723f6fe0(测试债根修六例)→ 134e89d0(wailsjs 再生)→ d2494ab8(包记录)→ 665d70c6
(.worktrees ignore，用户红线代码不落工作空间外)→ d20dec1d(合并入 main)。

- **刀11 Fleet 成员卡**：trajectory.FamilyAgentRows(父+子 transcript 源日志直算，
  不吃事件流 400 截断；main 在前/子按首条消息时间)+ FamilyEventStream.Agents +
  前端成员卡(主控/子 Agent 徽标+状态点 join runs+Token/费用/委派/消息/回报)+
  点击联动事件检查筛选。
- **刀12** 步详情 detail.step/turn 1:1(共 N 步)；粒度态上提父层。
- **刀13 inject 源细分**：引用块自带 file/dir/image/referenced-session/resource
  标签，fold 解析→ContextEvent.Form 五形态+mixed+定位名串；**免改内核**(拍板池
  预判「内核 payload 带 kind」便宜一档)；顺带根修 splitInjected 首空行切分把
  第 2/N 引用块误记 user 正文的既有缺陷。
- **刀14 promptPrefix/Suffix**：config [agent] prompt_prefix/prompt_suffix +
  buildSystemPrompt 首尾缝合(子代理 L1 同 compiler Fork，一处全覆盖)+
  SettingsView 顶层显式写回(空串=停用)+ 设置中心系统提示词组。
- **刀15** 趋势耗时开关持久化(contextPrefs.trendDuration)。
- **测试债根修**(基线 fda7f7b3 定罪六例既有红)：ContextView/ContextModal mock
  漏 FamilyEvents(Wave3 债)、ChatTabs 4→6 断言(刀3/5)、ComposerAttachmentBar
  uocr2 改名、NovelSettingPage 整文件负载吞点击(waitFor 重试式)、Transcript
  scrollTop defineProperty 无 writable(全量 exit=1 unhandled error 元凶)。
  终态全量 vitest 4291/4291 绿 exit 0、Go 五包绿、tsc 0、gen_bindings 724=724。
- **包** releases/gaea-contextup3-20261112-134e89d0(exe E5C7051CC540…，三 DLL
  同上包，双冒烟 200，装机 PID 35480>构建 14:40:33)。
- **上游事实源**：本机 DSH 包+上游 0.83.1 lib/client.js 内嵌字典 734 键全提取
  (docs/dsh-context-locale-zh-upstream.txt)；收官记录
  docs/gaea-contextview-complete-parity-2026-11.md；拍板池清空(余项=入口可见性/
  deltaBase 产品结构差异非缺口)。
- **合并收口 d20dec1d**：办公对话输出线休眠在制(最后改动 11-11 22:18)保全至
  office-conversation-redesign-wip 分支，主树清场；8 处同源冲突解毕(ctx_realcheck
  取 main 家族流面超集，余取分支超集)；合并闸 go build+gen_bindings 724=724+
  Go 四包+tsc+定向 vitest 111/111 全绿。
- **坑册新增**：全量 vitest 管道接 tail 吞失败清单(完整输出必须落盘)；
  worktree add 必须显式 commit(裸 HEAD 在主仓=main)；双树共享 .vite 缓存整文件
  与单测结果可分叉(隔离复跑+真基线双对照再定罪)；defineProperty 无 writable=
  只读；冒烟二次拉起与装机壳互扰(装机壳消失假象，重拉即愈)。

## 10. 超大附件改造：>25MB 不再失败（2026-11-12 晚，v4.543.0）

> 用户口径「办公板块输入框上传文件附件超过 25M 就失败，改成超过的可以识别路径使用」。单会话收官：
> 构建+冒烟 200+装机 PID 验证全绿，未打包（日常装机）。

- **诊断**：25MB 硬闸三层=前端 splitOversizeAttachments/拖拽 metadata 拒收 + 后端
  attachment.MaxFileBytes（单次 base64 整包过桥护栏）；发送侧本无限制——附件提交只注入
  @路径 文本，办公线 ResolveRefs 仅注入头部 64KB，AI 用 read_file 补读全文 → **路径引用天然
  支持任意大小**，闸只在「拖拽/粘贴必须搬字节」与「对话框本来有路径却白拦」两处错位。
- **两路改法**：
  - 选择对话框（handlePickFiles）：文件已在盘有路径 → 非图片附件取消大小闸，任何体积直接挂
    chip 按路径引用；图片仍守 10MB（识图链限制面）。
  - 拖拽/粘贴（attachDroppedFiles）：WebView2 拿不到源路径必须搬字节 → 新增**分块上传**四绑定
    （GaeaAttachmentChunkBegin/Append/Finish/Abort，落 OfficeB），前端 8MiB 3 字节对齐切片
    （每片独立可 base64 解码）流式上送，Go 侧 os.Root 内逐片落盘（.gaea/uploads，命名与单次
    路径同规 file-<token><ext>），任一时刻桥上只有单片；总量上限 MaxChunkedFileBytes=1GiB
    （预算检查在 DecodeString 之前，超限大串不进内存）；idle TTL 30min 清扫兜底泄漏会话；
    Finish 空文件拒收与单次同契约；失败 Abort 清理后 toast。
- **验证**：attachment 包 7 测（round-trip/空拒收/超预算/abort 清理/未知 token/TTL 清扫）+
  handler 回归（TestGaeaAttachmentRoundTrip 加分块腿）全绿；前端 hook 15 例绿（25MB+1 走分块、
  分块中段失败 Abort+toast、1GiB 元数据拒收、PickFiles 超大文件挂 chip、超大图片仍拒）；
  tsc -b 0；绑定面三处再生（bindingNames 728/legacy 86→82/签名 728）+漂移锁过（本批 4 名净，
  余 2 死绑定 EngineUpdateStatus/StrataModelStart=既有欠账非本批）。
- **坑册新增**：
  1. gen_bindings -names 只打印不写盘，bindingNames.ts 要拿输出物化（文件头注释「勿手改」
     是指内容口径，不是不用工具输出重写）；
  2. **legacyBindings.ts 必须在 AppBindings 认领之后再再生**——先再生后认领，4 个新名同时
     挂 legacy 面（Go 全集−认领集）与 AppBindings，锁四（legacy∩认领=∅）炸 tsc -b 且报错点
     在方向二（Exclude 语义被 ExcludeNames 掩护的误导性报错位置）；
  3. `npx tsc --noEmit` 与 `npm run build`（tsc -b）口径不同——前者放过 drift 断言错误，
     发版门必须走 `tsc -b`；
  4. Git Bash 调 powershell 传 `build\bin\gaea.exe` 反斜杠被吞成 `buildbingaea.exe`，冒烟
     传参一律正斜杠；
  5. 临时起意 git stash push 想对照基线差点卷走在制改动（stash list 有 10 月旧条目易误认），
     改用 `git show HEAD:<file>` 对照，stash 动作禁用。
