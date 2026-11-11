# gaea Day Log · 2026-11-12 · 办公板块上下文界面升级换代（dsh-context 0.83.x 全量复刻）

> 调研→拍板→动刀全链一天收官。调研报告 docs/gaea-contextview-upgrade-research-2026-11.md。
> 版本 v4.542.0，装机四次全冒烟。代码主体在 feat/context-upgrade 分支（合并挂账，见 §6）。

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
