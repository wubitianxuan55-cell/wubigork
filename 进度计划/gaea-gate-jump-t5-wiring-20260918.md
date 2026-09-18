# gaea · v4.336.0：chapter-gate 通知跳转 + oh-story T5 落库接线（规格书）

> 2026-09-18 · 用户指令「继续」。两件观察池/todos 收口，小刀合版。

## 甲件：chapter-gate 通知点击跳章节分析面板（v4.331 观察池头名）

- 论点：自动门通知（v4.331）只展示不交互——体检出了契约/质量问题，作者要点
  rail「章节分析」再自己选章；通知即入口才是闭环。
- 落地：`useChapterGateNotice(onOpen?)` 增可选回调（ref 保最新回调防过期闭包），
  `message.info` +onClick 传报告章号；CreatePage 接线 `gateChapter` 覆盖态 +
  跨章时 `GetChapter(n)` 拉正文供标注锚定（失败回退空串诚实降级）；面板关闭
  清覆盖。无回调=纯通知，既有行为零变化。
- 测试：钩子 +1（onClick→onOpen(章号)/无效章号不回调/无 onOpen 不炸）。

## 乙件：oh-story T5 落库接线（todos 欠账本体）

- 摸底结论（2026-09-18）：T5 资产=`.gaea/skills/novel-agents/` 七张 Markdown
  角色卡 + SKILL.md（dca0936b），代码侧引用为零；「落库」≠ 角色数据库（角色
  数据管道已齐：matchCharacterIDs→大纲节点、characterlib、characters.json），
  规格指定消费面=**run_skill/任务子代理**——缺的是静态资产→可派发通路。
- 落地三层之第 1+2 层：
  1. SKILL.md frontmatter `runAs: subagent` + 「派发契约」段（子代理执行序=
     解析 role → read_file `.gaea/skills/novel-agents/agents/<role>.md` →
     按卡执行；七 role 枚举；主会话派发指引 arguments 格式 `role=<角色> <任务>`）。
  2. spawn 模板 `subagent_writing`（写作域角色卡前缀，第五个内置模板）+
     boot `subagentSkillToTemplateKind("novel-agents")` 映射——七角色子代理
     共享 L4 前缀缓存（V5.30 机制）。
  3. 验收钉：skill 包 `TestNovelAgentsSkill_DispatchableWiring`（仓库根为
     ProjectRoot 直读：Scope=project/RunAs=subagent/派发契约锚点/七 role
     齐全）+ boot 映射测试 + cache 计数 4→5。
- 第 3 层（创作间按钮直达 Novel 绑定派发）= 增强，不是规格欠账，候反馈另刀。

## 出口对照

- 通知点击→分析面板打开该章（跨章正文正确锚定）✅；无回调零变化 ✅
- `run_skill("novel-agents", "role=… 任务…")` 可派发隔离子代理（机制测试钉
  资产形状；活模型派发走查挂真机池）✅ 资产接线层面收官 ✅
- oh-story 余项只剩「写前契约拒绝生成硬闸（需先有一键补大纲）」（等上游）。

## 门禁

go build/vet/test、tsc、eslint、vitest 全量、drift OK@707、版本三处 4.336.0。
