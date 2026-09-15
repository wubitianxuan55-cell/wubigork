# gaea 7.2-2 判据② · 结晶技能调用计数（skill_stats）· 实施规格（v4.319.0）

> 来源：v4.317 出口对账判据②欠账「结晶技能调用 ≥5 次且成功率可见（需调用计数机制
> statsFile 先例另刀）」；docs/gaea-stage7-plan-2026-09.md §2 判据原文「技能被实际
> 调用 ≥ 5 次且成功率可见」。小刀单线，主代理直做（体量一轮，不拆子代理）。

## 0. 论点与现状缺口

现状（核实）：技能使用计数是**前端会话级**——`useToolStats` 只统计当前会话消息里的
`run_skill` 调用（`App.tsx:351` → CapabilitiesPanel `skillCounts` chip），**不持久、
不跨会话、无成功率、且漏掉主消费通道 `read_skill`**（boot 装配的按需加载器，
sysprompt.go:126 WireReadSkillResolver——结晶技能被模型实际「翻开来用」的时刻）。
判据②的「≥5 次」是跨会话累计口径，会话级计数支撑不了。

本刀补：**工具级调用计数**（read_skill + run_skill 双钩子）→ `<DataRoot>/skill_stats.json`
持久化（route_suggestions 同款容错读+原子写）→ `GaeaSkillStats` 绑定 → 能力面板技能行
「N 次 · 成功率 X%」可见。

诚实口径（V1 裁决）：**成功率=工具级**——read_skill 找到技能并交付正文=ok、找不得=fail；
run_skill 管线执行无错=ok、报错=fail。回合级成功率（用户满意与否）留观察池，不在本刀
虚构。「≥5 次」是使用事实门槛（可见性支撑），不做硬闸不弹窗。

## 1. 纯包 `internal/skillstats`（零 IO 表驱动，先例 routesuggest/taskinbox）

```go
package skillstats

const MaxSkills = 500 // 防膨胀：新名拒收（既有名更新不受限）

type Stat struct {        // 单技能累计
    Calls  int   `json:"calls"`
    Ok     int   `json:"ok"`
    LastAt int64 `json:"lastAt"` // unix 秒
}
type File struct {
    Version int             `json:"version"` // 恒 1
    Skills  map[string]Stat `json:"skills"`
}
type StatView struct {    // 绑定视图（camelCase）
    Name   string `json:"name"`
    Calls  int    `json:"calls"`
    Ok     int    `json:"ok"`
    LastAt int64  `json:"lastAt"`
}

// Record 纯函数累计：name trim 后空 → 原样返回；新名且 len(Skills)≥MaxSkills
// → 原样返回（宁少勿扰，不挤掉既有）；否则累计 calls、ok 命中加 Ok、LastAt=at。
func Record(f File, name string, ok bool, at int64) File
// View：calls 降序 → name 升序稳定；恒非 nil。
func View(f File) []StatView
```

测试（表驱动）：Record 空/纯空白名忽略 / ok 与 fail 累计 / LastAt 覆盖 /
MaxSkills 拒新不拒旧 / View 排序与稳定 / File JSON 形状钉死（camelCase）。

## 2. boot 接线（internal/gaea/boot）

- `Options.OnSkillUse func(name string, ok bool)`（缺省 nil = 零行为变化，
  CLI/TUI 宿主不传不受影响——EmitSubagentText 同款先例）。
- `buildSystemPrompt` 内 resolver 包装：
  ```go
  resolve := func(name string) (string, error) {
      body, err := readSkill(name)
      if onUse != nil { onUse(name, err == nil) }
      return body, err
  }
  builtin.WireReadSkillResolver(resolve)   // 读不到名（resolve 报错）也算一次
  // fail——技能名漂移/删除后的如实计数，不静默。
  ```
- `boot.go:405` run_skill 挂载处包一层装饰器（OnSkillUse 非 nil 时）：
  执行 `t.Execute` 后按 err 记 ok/fail；工具名/参数不动（args.name 与 read_skill
  同名字段，装饰器内解析失败不计数——宁少勿扰）。

## 3. App 层（internal/app/gaea_skill_stats.go + 接线）

- 路径 `config.DataRoot()/skill_stats.json`；`loadSkillStats` 容错读（缺失/损坏/
  version≠1 → 空 File），`saveSkillStats` temp+rename 原子写（先例 route_suggestions）。
- `recordSkillUse(name string, ok bool)`：mu 串行 → load → skillstats.Record →
  save。每次调用写盘（文件 ≤ 数 KB，无批处理必要）。
- `gaea_handler.go` Build Options 增 `OnSkillUse: ga.recordSkillUse`（ga 判空防御：
  引擎未起时回调丢弃——与 ga.ctrl 同款守卫）。
- 绑定 `GaeaSkillStats() []skillstats.StatView`（**OfficeB +1，绑定面 695→696**）：
  只读现算，load+View，损坏回空表。

## 4. 前端（CapabilitiesPanel/SkillsSection + 三语 + mock）

- `useCapabilitiesData` 或面板挂载时拉 `app.GaeaSkillStats()`（open 态一次+reload
  按钮随既有 reload 刷新；**零轮询**）。
- SkillRow：持久统计在现有会话计数 chip 旁加一行灰字 `N 次 · 成功率 X%`
  （stats 无该技能时不渲染——诚实空态；会话 chip 语义不变=「本会话」）。
  data-testid=`skill-usage-stat`。
- 三语 +2 键：`caps.skillUsage`（"{n} 次" / "{n} uses" / "{n} 次"）、
  `caps.skillSuccessRate`（"成功率 {p}%" / "{p}% success" / "成功率 {p}%"），
  zh=en=zh-TW 键集相等。
- mock/chat.ts +1 桩（样本含 1 条 ≥5 次高成功 + 1 条低次）；spaceBindings
  GaeaSkillStats → work（能力面板只在办公工作台）；bindingNames 收口时再生。

## 5. 测试与门禁

- Go：skillstats 表驱动 ~8 例；app 层 3 例（record 往返落盘/损坏文件回空/
  绑定 View 排序）；boot 不单测（装饰器 3 行，靠 app 集成与走查）。
- 前端：SkillsSection 2 例（统计行渲染/无数据不渲染）。
- 门禁：定向 → tsc -b / eslint 改动文件 / `go test ./internal/skillstats
  ./internal/app -run 'TestSkillStats' -count=1` → 全量 ci.ps1 恰一次。

## 6. 收口清单（v4.319.0）

gen_bindings → drift OK@696 → 版本四处（sync-version 口径）→ build → exe/SUMS/
桌面副本/冒烟 → releases/v4.319.0.md + CHANGELOG/README → .gaea AGENTS 迁 1 插 1
+ progress/todos → commit+tag。

## 7. 足迹

| 文件 | 动作 |
|---|---|
| internal/skillstats/**（新） | 纯包 + 测试 |
| internal/gaea/boot/boot.go、sysprompt.go | Options.OnSkillUse + 双钩子 |
| internal/app/gaea_skill_stats.go（新）+ gaea_handler.go + bindings_office.go | 持久化 + 绑定 |
| frontend/src/gaea/components/CapabilitiesPanel.tsx、capabilities/SkillsSection.tsx、hooks/useCapabilitiesData.ts、lib/mock/chat.ts、lib/spaceBindings*、locales 三语 | 统计行 + 桩 + 键 |
| 主代理收口 | types/bindingNames/gen_bindings 产物、版本四处、releases/CHANGELOG/README/.gaea |

## 8. 出口对照与观察池

- 判据②「调用 ≥5 次且成功率可见」：**可见性本刀落地 ✅**；「≥5 次」是使用事实，
  由真实使用积累（观察池：两周后清池核数）。
- 观察池：回合级成功率（用户满意信号，需点赞点踩或重生成信号另刀）；蒸馏建议 tab
  内联展示已结晶技能的计数（本刀只做能力面板单点）；skill_stats.json 无 UI 清单
  （条目数≤技能数，暂不需要）。
