# 全仓审计第 49 批 · CreateTemp 非同构八处收口（两转 AtomicWrite 单源 / 六处保留点裁定注释）· 2026-10-03

> 接续 [round-50（批次四十八）](round-50-p1-batch48.md)。上报未动池「app 包 8 处非同构 CreateTemp」：逐站点裁定——**同型收敛两处、异构保留六处（各带裁定理由）**。一处可观测行为收紧（申报），**不抬版本**。
> 快照：开工时 HEAD = `037ca754`（批次四十八）；工作树干净。

---

## 一、八处逐站点裁定（「非同构」的正解是按语义分流，不是盲并）

- **转 `fileutil.AtomicWrite` 单源**（2 处，均带「先移除旧文件再替换」手搓）：
  - `chapter_art_manifest.go saveChapterArtManifest`：旧手搓在移除与替换之间留**清单缺失的可见中间态**（崩溃/并发读空窗），单源 rename 原语消灭该窗口；权限 0o644（批 41 perm 语义）。
  - `imagehub_ledger.go 折叠重写`：同款空窗（台账即消失）；权限 0644 与追加路径 `OpenFile` 口径对齐；行折叠循环改 strings.Builder 后一次落盘。旧注释「Windows 下 os.Rename 不能覆盖已存在文件」系过时认知（Go os.Rename 走 MOVEFILE_REPLACE_EXISTING），RenameWithRetry 本就可覆盖。
- **保留+裁定注释**（2 处，fsync 能力位）：`gaea_benchmark.go`（T7-2 报告）、`gaea_ui_extra.go`（WriteFile 工具编辑主链）——两站点都有 `tmp.Sync()`，是 AtomicWrite 没有的能力；**是否统一 fsync=全仓 fsync 决策，归拍板池**。注释写明「刻意不并入」+指针，非同构从「无document」变「有裁定」。
- **保留（语义不同构，非收口对象）**（4 处）：`characterlib_score.go`（评分图 OS 暂存+调用方 cleanup 句柄）、`gaea_pptx.go`（大纲脚本 OS 暂存）、`intent_router.go`（OCR 截屏暂存）——三处是**scratch 文件**（一次性输入物，无替换语义）；`sin_booksource_handler.go`（EPUB）——载荷由 go-epub 库自写（非字节切片），临时名+RenameWithRetry 已是正确形态且注释完备。

## 二、门禁

- gofmt 0 差 / `go build ./...` 绿 / vet 绿 / 定向族绿（ChapterArtManifest 三例+ImageHubLedger 四例含 PruneKeepsLatest 折叠路径）/ **app 全包 117.5s 绿**。
- 行为差异申报：两处替换路径的**中间态窗口消失**（收紧）；错误文案由分步多条合并为单条包装（错误文本面变化，errors.Is 语义不变）；清单文件 POSIX 权限 0600→0644（批 41 perm 生效语义的延伸）。

## 三、上报未动池终账

- AtomicWrite perm（批 41 ✓）/ 裸 os.WriteFile 七处（批 42 ✓）/ **CreateTemp 八处（本批 ✓）** / 全仓 fsync 决策（拍板池）。
- **免拍板活池就此清零**。余池全拍板：coupling ~20、109 绑定删除、fsync 决策、零散死码；观察池：WhisperSubgraph 大小写疑似真机 bug。

## 四、下一批

- 请用户对拍板池批量裁决（十项+109 绑定+fsync）；或观察池 WhisperSubgraph 真机验证；或转入新审计周期。
