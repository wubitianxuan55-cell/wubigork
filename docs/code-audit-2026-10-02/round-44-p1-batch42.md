# 全仓审计第 42 批 · 裸 os.WriteFile 收口（IN1-02 相邻清单七处改调 AtomicWrite）· 2026-10-03

> 接续 [round-43（批次四十一）](round-43-p1-batch41.md)。取上报未动池「scene/style/export 裸 os.WriteFile」（04 分册 U2 相邻 / 06 分册 IN1-02）：**逐站点现状复核后七处收口改调 `fileutil.AtomicWrite`，一处证伪为刻意冻结**。申报行为变化（崩溃安全面：中断不再留截断文件；权限面随批 41 生效），**不抬版本**。
> 快照：开工时 HEAD = `e2b8500b`（批次四十一）；工作树干净。

---

## 一、逐站点复核（审计行号全按移动靶对待）

| 站点 | 现状复核 | 处置 |
|---|---|---|
| scene.Write（scene.go:113 正文）/ writeMeta（:258） | 存活（internal/scene 包，审计后未动） | 收口 |
| export TXT/MD（export.go:158/193） | 存活（两处同行文，replace_all 一次收口） | 收口 |
| characterlib 剧照（portrait.go:114） | 存活（saveImageFile 共享写入口，含路径穿越防御注释段原样保留） | 收口 |
| pins.Save（pins.go:58）/ SaveRecentWorkspaces（recent.go:55） | 存活（D21 两处：截断 JSON 曾致侧栏「项目」分组读空） | 收口 |
| gaea_verify.go 回滚写（:279→现 :298） | 存活 | 收口 |
| style.SaveProfile | **证伪为刻意冻结**：project_files.go `WriteStyleProfileFile` 注释明写「非原子写，与历史 style.SaveProfile 行为逐字节一致——勿顺手升级为原子写」（前批裁决+注释钉） | 跳过（申报） |

- 七处均为一行改调 + import 增补；AtomicWrite 自带 MkdirAll（pins/recent 原有 MkdirAll 与之幂等重叠，删侧不动作留原行=最小 diff）。
- 修法与批 41 同源：临时件+rename，崩溃中断不留截断正文/元数据/注册表；perm 走批 41 的新语义（0o644 精确生效）。

## 二、门禁

- gofmt 0 差 / `go build ./...` 绿 / 受影响五包测试全绿（scene/export/characterlib/pins/config）/ app 包 `TestVerify|TestRollback` 定点绿（第七处覆盖）。

## 三、余量与下一批

- 裸写清单：internal/app 尚有 ~15 处导出/报告类 os.WriteFile（gaea_export/crosslink/diagram/tools/ui_extra 等）——价值密度低于本批（多为可再生报告，非唯一用户数据），留观察池不追。
- 上报未动池余：全仓 fsync 决策（拍板）、app 包 8 处非同构 CreateTemp。
- 下一批候选：45 条 NOT_MOCKED 补 mock（dev 面单排）或对账地图其他活池；coupling ~20 与零散死码等拍板。
