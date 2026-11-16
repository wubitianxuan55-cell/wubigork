# gaea-wenxue-v5-20261116-94d2924b

- **源提交**：`94d2924b`（feat(wenxue): 无限制档零安全底线 + 云端模型破甲契约随书注入 + 底线实测）
- **打包方式**：主工作树构建（工作树已含该提交，无其他业务改动）。
  前端 `npm.cmd run build` 38.96s → `wails build -s -ldflags "-s -w" -trimpath`
  （1m15.8s，bindings 未跳过）；ucrt64 gcc 前置、CGO_ENABLED=1、系统默认 TMP/TEMP。
- **内容**：`payload/` = gaea.exe + onnxruntime.dll + sherpa-onnx-c-api.dll +
  sherpa-onnx-cxx-api.dll（同目录四件），SHA256 见 SHA256SUMS。
- **冒烟**：`scripts/smoke.ps1 -ExePath releases/.../payload/gaea.exe` →
  `/api/health 200 {"status":"ok"}`，冒烟进程已回收（PID 46408）。
- **本批改动**：
  - 无限制档零安全底线：guidance 改零禁止题材/零回避/零审查口径；恒守线仅非无限制档写入；
  - `maturecraft` 新增 `unrestricted` 档，功能闸与硬线拆分（无限制档不注入成年/胁迫/同意/未成年条目）；
  - 新增 `internal/maturecraft/jailbreak.go`（云端模型破甲契约），无限制档建书时写入
    `书籍信息.md`「无限制档执行契约」节（随 pinned 自动进会话）；
  - 前端 Sidebar 无限制档提示改零底线口径；`docs/ADULT_MODE.md` 第七节记四档边界；
  - 新增 `docs/gaea-cloud-model-limits-20261116.md`：Grok 4.7 / DeepSeek v4-pro 实测底线矩阵。
- **产物哈希**：
  - zip `27f73954597ba2d07a4d6e087d27ec07608ec8b7eb8ef56fe17b1ec33a0d4e14`
  - tar.gz `86abd4de53888d81ba507c6a5c65ca626ef2bd10e923e99d289c1808d4720144`
- zip/tar 留盘不入库（惯例）；PACKAGE-INFO.md 与 SHA256SUMS 入库。
