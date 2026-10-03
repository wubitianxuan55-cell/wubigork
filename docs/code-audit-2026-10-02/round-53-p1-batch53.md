# 全仓审计第 53 批 · A1① 109 条零调用绑定删除（绑定面 744→635）· 2026-10-03

> 接续 [round-52（批次五十）](round-52-p1-batch50.md)与[批 51 裁决材料](decision-brief-2026-10-03.md)。拍板执行 A1①：109 条零调用绑定（批 18 普查名单）**整体摘除绑定面**——App 方法体保留（防内部调用方），走生成器排除清单管线。**申报：绑定面 744→635 / legacy 面 184→75；wailsjs 门面 JS 归下次 wails build 重生成。不抬版本。**
> 快照：开工时 HEAD = `e05392cf`（批次五十二）；工作树干净。

---

## 一、正确管线（两次弯路后的定型）

1. **生成器 `excludedBindings` 排除清单**（109 名附复活动说明）：collectMethods 在 receiver/export 过滤后跳过排除名——App/子状态方法体零改动，仅不再生成门面转发。手工删生成物两条弯路（行扫描被单行函数体坑过量删除→回滚；生成物 DO-NOT-EDIT 应走上游）后定型。
2. **全产物重生成**：门面 11 文件（imports 生成器自算）/ bindingNames.ts 744→635 / legacyBindings.ts 184→75 / bindingSignatures.ts 635 名。
3. **完整性测试模板收编排除语义**：`TestBindingsCompleteness`（生成物）从 want 剔除 excludedBindings（名单物化进生成文件——测试在 app 包内无法引用 scripts 包；发射三连修：字面量逗号/行尾逗号/import 前置——**三次编译器追捕**）。
4. **wails.d.ts**（legacy App.* 类型声明层）摘 40 条在册声明；**wailsjs 门面 JS**（AddOutlineNode 等）归下次 `wails build` 重生成，零源码导入（零调用者保证）。
5. **knownDead 清册**：check-bindings-drift.ps1 §4 清空+注记（漂移检查过：bindingNames 635 一致 ✓ / legacy 75 ✓ / 在册 0 ✓）。**PS 脚本教训**：改写丢 UTF-8 BOM → PS5.1 按 ANSI 读中文注释 → 解析崩；重写必须 utf-8-sig。

## 二、门禁

- gofmt 0 差 / `go build ./...` 绿 / vet 绿 / **app 全包 135s 绿（TestBindingsCompleteness 新语义过：App 744−109=635=门面）** / builtin 全包绿 / tsc -b 绿（**legacy 面本就不在前端契约——AppBindings/桥接零改动零红，与批 46 mock 小写同理**）/ contract+spaceBindings 金样 15 例绿 / **全量 vitest 435 文件 3760 例全绿**。

## 三、申报与余量

- 申报：线上绑定面 744→635（真机不可再调 109 名——零调用者实证下无可见影响）；App/子状态方法体保留（内部调用方不受扰），其二阶死码清理归零散死码拍板项。
- 复活方式：生成器 excludedBindings 除名 + 重跑生成器 + drift §4 复检。
- **A1① 关账**。拍板池余：A2（AP4-01 任务身份）/A7（fsync 默认开关）/B1/B2（新功能立项）。

## 四、下一批

- 等用户对 A2/A7/B1/B2 裁决；或 WhisperGraphPanel 真机复走；或按需发版（42+ 批本地未推，tag+源码包归档随发版走）。
