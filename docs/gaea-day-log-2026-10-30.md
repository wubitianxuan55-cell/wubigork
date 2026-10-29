# 当日总账（2026-10-30）：众神殿三连发——实弹修复 + UI 走查 + 亮态补课

> 批末=d69ab614；桌面 exe=build/bin/gaea.exe 55,948,288B SHA256 前缀 88ea4cdcd835
> （桌面副本同哈希）。今日三发：**v4.528.1**（tag=7fff6dbe，实弹修复批）、
> **v4.529.0**（tag=c518c7da，UI 实拍走查五刀）、**v4.529.1**（tag=d69ab614，
> 小结标题挤压根修=亮态补课）。前夜线：v4.528.0 众神殿整包 + bed2b286 三崩点
> 首修系 10-29 深夜（见 10-29 日总账晚间段之外的众神殿线）。

## 提交索引（10 笔，按线归组）

| 提交 | 线 | 一句话 |
|---|---|---|
| e078043a | 众神殿 | 导出双门根修：壳内 `<a download>` 不落盘（v4.162 实证）改走 saveExportBlob 中立层+取消诚实反馈+Request.AgentIDs null 归一化兜底；圆桌裁决真实发现落地 |
| 60ce7f15 | 主题（DSH 并行线） | 浅色白字两根修：绘梦生成钮透明混腿实底化+运行监测幽灵令牌接 tertiary |
| 7fff6dbe | **发版** | **v4.528.1**（实弹修复批=bed2b286 三崩点+e078043a 导出双门+60ce7f15 白字；沙箱壳实弹验收：另存为 #32770 对话框弹出+取消 toast） |
| fd3ea7ee | 众神殿 | UI 实拍走查五刀：中断信息三重去重/探针前「检测中」态/grok mini 卡「内置」/CJK 断行修复（ch 限宽按拉丁 0 字形宽是坑，CJK 量宽用 em；welcome 标题手工两行断行） |
| c518c7da | **发版** | **v4.529.0**（UI 五刀；沙箱五态实拍定刀） |
| 7d6a7f4b | V5 归档 | 立项提案+Codex×ZCode 辩论纪要（D1~D7 收敛态）+五份研究底稿入库+README 索引登记 94→96；辩论期提案冻结，拍板权在用户 |
| c3ce3db5 | releases 卫生 | 守卫计数回正（526→528/716→718 两版欠账）+V4_RECENT 裁员回 34 席 |
| 2d05197c | releases 卫生 | 根目录保留策略归位：迁 v4.481~494+v4.522/523 共 16 对入 archive（根回 34 席），17 处引用随迁改写 |
| 2f01cc87 | 众神殿 | 小结标题挤压根修：verdict-head flex 行被 tag+印记预留(72px) 挤压至 256px 绞碎——head 换行+标题恒占整行+字号 24px；**亮态目检补课逮住**（v4.529.0 两行验收只在折叠栏态成立） |
| d69ab614 | **发版** | **v4.529.1**（亮暗×栏态四象限实拍验收） |

## 验证矩阵（全部真机/沙箱证据）

- 圆桌两轮实弹（10-29 深夜）：四席陈词+grok 主席裁决 JSON+签注落盘+UI 零崩
- 导出双门实弹：CDP 点导出→系统「保存文件」#32770 对话框弹出；关闭→「已取消导出」
  toast 3s 自消、错误横幅恒空；真存落盘腿留用户（原生对话框 UIA/SendKeys 自动化不可行）
- UI 五态实拍：欢迎/发起/已小结/失败/折叠栏（暗态）；亮态暖纸三态补拍
- 四象限终验：亮暗 × 栏折叠/展开，小结标题两行断句全部成立
- 测试：pantheon go test 绿；前端 19/19（PantheonPage 18+streaming 1）；tsc 零错；
  mock 契约 83/83；文档守卫全绿（孤儿 0/悬空 0/计数对账 529/719）
- **全量 CI 终验**（ci.ps1，批末三跑）：①首跑逮 gofmt 三处欠账（grok 刀提交漏跑全
  门禁溜入）→ 56be39af 收口；②go test 全量一发 flaky 复跑 EXIT=0；③末跑 go 侧全绿，
  vitest 463 文件/4050 例绿 + **2 文件并发假红**（OriginalSinPage.tabs 39.7s/
  ContextView 41.2s 超时，与今日改动零交集，隔离复跑 39/39 绿=flaky 定性成立）。
  结论=全绿（含 flaky 定性），满载假红两户挂观察池

## 新坑入册（详录各线记忆）

1. **ch 限宽按拉丁 0 字形宽**——CJK 每字 1em，`24ch` 实际只容半数预期字数，配
   `text-wrap: balance` 绞出碎句；CJK 排版量宽用 em。
2. **flex 收窄类断行 bug 的目检必须覆盖主列宽度全布局态**（档案栏折叠/展开是两个
   世界）——单态验过≠修复成立，v4.529.0 正是栽在这里。
3. 原生「另存为」对话框自动化不可行：UIA SetValue/FindAll 全 OperationTimeout、
   SetForegroundWindow 被系统拦、SendKeys 走飞进 webview；验收配方=EnumWindows
   找 #32770+WM_CLOSE 触发取消+CDP 抓 toast。
4. jsdom 下 antd Dropdown portal 挂载与 RTL waitFor 轮询相克：须实等
   `.ant-dropdown` 再点菜单项；同用例连开第二次菜单 toggle 反转=每用例独立挂载。
5. vitest 必须从 frontend/ 目录跑（repo 根落无 jsdom 环境）。
6. sed 全文替换 `}))$` 会吃掉测试文件多处闭合括号——测试文件编辑永远用 Edit。
7. heredoc 反斜杠降级再犯（python 一律 Write 脚本文件再跑）。
8. MSYS_NO_PATHCONV=1 前缀下 taskkill 用单斜杠 `/IM /PID`。
9. 沙箱 localStorage 跨壳存活（rail 态/主题），走查前显式重置。

## 余池（全部待用户拍板/实弹，无自主可动项）

- **V5 立项拍板**：8 项签字清单在 `docs/gaea-v5-zcode-debate-2026-10-29.md` 末尾；
  D6 待 Codex 答攻击者问题。未签字项不进实施。
- **众神殿实弹**：导出真存落盘腿（用户顺手点一次导出即补上）。
- 既往各线实弹池：OCR 长跑、50+ 轮长会话、30 章长跑、进度条手感、ComfyUI
  JobObject 观察等（见 10-29 日总账与各线记忆）。
- **CI 观察池**：OriginalSinPage.tabs/ContextView 满载并发假红（39~41s 超时、隔离
  复跑绿）——候选刀=重组件文件 testTimeout 单独放宽或 maxWorkers 再压，与 vite
  已载的「首个 await import 计入 testTimeout」同族。
