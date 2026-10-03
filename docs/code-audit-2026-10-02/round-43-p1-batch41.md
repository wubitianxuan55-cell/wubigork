# 全仓审计第 41 批 · 上报未动池收口（AtomicWrite perm 参数生效）· 2026-10-03

> 接续 [round-42（批次四十）](round-42-p1-batch42.md)。取「上报未动池」最实一条：`fileutil.AtomicWrite` 的 `perm` 参数自实现以来从未被消费——审计上报项收口。**申报行为变化**（POSIX 面，Windows 等效 no-op），**不抬版本**。
> 快照：开工时 HEAD = `20ab29d6`（批次四十）；工作树干净。

---

## 一、问题与修法

- **问题坐实**：`os.CreateTemp` 恒以 0600 创建临时件，rename 后目标权限恒 0600——全仓 ~30 处调用方传入的 perm 静默失真。最实受害者=**editfile 工具**：`atomicWriteEncoded` 刻意读原文件权限位透传（工具描述明文承诺「file permissions are preserved」），实际从未兑现；writefile/movefile 同型意图同被吞。
- **修法**：rename 前 `os.Chmod(tmpPath, perm.Perm())`——对仍在己手的临时件改权限，不产生错权限的可见中间态。Unix 精确生效（**不经 umask**：普查 ~30 调用点全部传显式值 0o644/0o600/读原文件透传，无 umask 依赖语义要保留）；Windows 上 Chmod 仅切换只读位，0600/0644 均可写=等效 no-op（主运行时行为零变化）。
- **测试执行面设计**：perm 断言仅 POSIX 有意义（Windows `Mode()` 不模拟权限位）→ 断言用 `runtime.GOOS == "windows"` 跳过；**但 fileutil 原不在 ubuntu race job 包清单里，断言将永远没有 POSIX 执行点（测试成装饰）**→ ci.yml ubuntu race job 包清单补 `./internal/gaea/fileutil/`。Windows 侧 chmod 代码路径由既有用例（UsesRetryPath/CreatesMissingParentDirs）实际执行覆盖。

## 二、门禁

- gofmt 0 差 / `go build ./...` 绿 / `go vet` 绿 / fileutil 全套绿（Windows：4 PASS + 1 SKIP 按设计）/ 最近消费方 `./internal/gaea/tool/...` 全绿（12.5s 无涟漪）。
- POSIX 断言（0600 首写→0644 覆盖写均带新 perm，editfile 依赖路径）由 ubuntu race job 在 ci 兑现。

## 三、上报未动池余量与下一批

- 上报未动池余：全仓 fsync 决策（行为面大=拍板）、app 包 8 处非同构 CreateTemp、scene/style/export 裸 os.WriteFile（原子性缺口，可循本刀续收）。
- 下一批候选：裸 os.WriteFile 三处续收（同配方）或对账地图其他活池；coupling ~20 与零散死码等拍板。
