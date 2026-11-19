# gaea-office-r2-recovery-20261119-316860cf

众神殿 pan_1795058214142（「优化完善办公板块的工具，特别是本地工具」，4 席 3 轮，
主席 Codex，用户签注 accepted）裁决施工收官包。

## 提交

- 打包提交：`316860cf`（fix(xlsxedit): A 案原子写落地）
- 本会话包含提交（三笔，均已装机未 push）：
  - `7cbf20f4` feat(office): R2 分档诚实恢复契约+registrymap/灵犀文档回写（action-1+action-4）
  - `4958fbd0` test(office): action-2 打开中 xlsx 占用观察 A/B 落档
  - `316860cf` fix(xlsxedit): A 案原子写落地（saveAtomic+占用归因）
- 基线：`41043ce7`（聊天/子代理系统提示词迁移，上一批）

## 内容

1. **R2 分档诚实恢复契约**（裁决唯一 P1）：office_edit/office_save 描述与成功/
   失败/保存三类回执全部携带分档撤销边界（Word=一次 Ctrl+Z 整批仅当前会话；
   Excel/PPT=助手不提供整批撤销、粒度未核验不得写成逐操作可撤；证据卡仅可
   复核无跨会话回滚）；comhelper.UndoBoundary 单源；静态契约测试三件（含源级
   守卫钉 OfficeHelper.cs undo/UndoRecord 仅 Word）。
2. **action-4 文档面**：registrymap xlsx_apply 谓词补「基线快照+证据卡」+镜像
   文档；灵犀 optimize-plan 文首终局回写。
3. **action-2 占用 A/B**（env 门控 live 测试 GAEA_XLSX_OCCUPY_AB=1）：9/9 干净
   零半写，P-a 占用归因=true → 按裁决不改提示词。
4. **A 案原子写**（用户拍板）：xlsxedit.saveAtomic（同目录临时+RenameWithRetry）
   替换 excelize 原地直写；rename 被挡时错误补占用归因（「目标正被其他程序
   占用，如 Excel/WPS 打开中——关闭该文件后重试」）。

## 产物与校验

见同目录 `SHA256SUMS.txt`。要点：

| 文件 | SHA256（前 16） |
|---|---|
| payload/gaea.exe | `84ffb2ca7f70b3c8` |
| payload/onnxruntime.dll | `68d0d5f502be4509` |
| payload/sherpa-onnx-c-api.dll | `5dd36dbd3572846b` |
| payload/sherpa-onnx-cxx-api.dll | `4c56327796cdb980` |
| gaea-office-r2-recovery-20261119-316860cf.zip | `b1be43db39d29f02` |
| gaea316860cf-source.zip | `e1ab3ad513c5ee00` |

- 三 DLL 取主树 build/bin（不入库，与装机位同源同哈希）。
- payload exe 为干净工作树（`.tmp/gaea-office-r2-316860cf`，detached 于
  `316860cf`）独立构建；与当日装机位 exe（`eb5c57ec`，主树同提交构建）内容
  同源、哈希因构建时间戳不同——两者各自冒烟通过。

## 构建与验收

- 干净工作树构建：frontend `npm.cmd run build` 37.1s（node_modules 走主树
  junction，规避无 lockfile 全新 install 的 esbuild Access denied 坑）→
  ucrt64 环境 `CGO_ENABLED=1 wails build -s -ldflags "-s -w" -trimpath` 56.2s。
- 冒烟：`scripts\smoke.ps1 -ExePath <本包 payload>\gaea.exe` → `/api/health`
  200（正式路径运行，非 .tmp）。
- 测试底账：comhelper 9.4s / tool+builtin 21.5s / app 全包 119.9s 绿；
  占用 A/B 复验 excel/wps ×3 clean-fail+P-a=true；gofmt/golangci 本批文件 0。

## 余项（不在本包）

- P-c（模型见占用错误后两步内自愈）——留用户实弹观察。
- action-3 R1 保存前快照——另立需求，须用户先定义「未保存修改」恢复语义。
- Excel/PPT 应用自身撤销粒度真机演练（约 1 小时/应用）。
