# gaea-wenxue-v2-20261116-a7cc7c2a

文学板块 v2 功能包：众神殿两场裁决工具收口 + 技能面动态终验 + 角色面板 v2 +
记忆左右脑 + 设定标签页（编队退役）。

- **打包点**：`a7cc7c2a`（detached 干净工作树构建，主树零未提交，HEAD=打包点）
- **提交链**（本包功能面 448743a8..a7cc7c2a 六笔）：
  `448743a8` 工具首刀收口批（fact_* 除名+run_skill 可见集合+媒体根分档+窄告警+
  census 守卫）→ `8a5bfc9a` 续议矮窗修复落库+记档 → `0de58522` 技能面动态终验
  （四段式第四腿+分发链核验）→ `12a00b4f` 角色面板 v2（master-detail+角色库只读
  导入）→ `9773026a` 记忆左右脑（图文脑独立目录+记忆 tab 脑窗退役）→ `a7cc7c2a`
  设定标签页（编队退役+世界观/时间线/规则查看编辑 AI 生成）
- **构建**：干净树 Vite 29.59s + Wails 37.4s（bindings 未跳过）；构建后 wailsjs
  内容零漂移（`git diff --quiet` 通过，warning 全为 CRLF 显示假象）；DLL 预构建件
  取主树 build/bin（同源同哈希）
- **冒烟**：payload 正式路径 `/api/health` 200（PID 49344）
- **产物哈希**（详见 SHA256SUMS-a7cc7c2a.txt）：
  - gaea.exe `0211079464f7…2153`（干净树重建哈希，与主树装机位构建哈希不同属
    预期，代码同源；主树装机位=各批次 INSTALL OK 的 exe）
  - onnxruntime.dll `68d0d5f5…` / sherpa-onnx-c-api.dll `5dd36dbd…` /
    sherpa-onnx-cxx-api.dll `4c563277…`
- **归档**：zip + tar.gz 留盘不入库（各约 29 MB）
- **功能要点**：
  - 工具面：`fact_add/list/clear` 图文除名；run_skill 接可见 Skill 集合（extras
    可达+隐藏名 fail-closed，办公双 nil 活读原样）；image_gen/ocr 媒体根分档
    （锚小说目录）；chapter_list 删假 dir+共享章号 helper+窄告警挂回执
  - 记忆：图文脑 `<MemoryUserDir>/wenxue-brain` 独立 DB（boot MemoryUserDir 缝），
    与办公脑物理隔离；记忆 tab 退役（办公脑窗）；**图文脑从零起步不迁移**
  - 角色 tab v2：master-detail+结构化六节阅读+CAS 编辑+角色库只读导入
    （红线=零写向绑定，不污染角色库）
  - 设定 tab：设定文档+时间线查看/编辑（时间线第N章锚结构化视图）+AI 生成
    （预设五类→Composer 指令→write_file 落盘→filetree 自动重载）；编队 tab 退役
  - 主区七 tab：对话/轨迹/上下文/洞察/角色/阅读/设定
- **已知遗留**：novel-agents/novel-deslop/novel-review 三件 allowlist 当前不可达
  （需用户 gaea.toml 配 `[skills] paths` 指仓内 `.gaea/skills`）；后置面（完整
  账本/12→7 合并/批量 compact/结构化 writeback/lint 收益实测）按众神殿裁决不动
- **办公零影响**：全批次双 nil 缝原样+隔离测试钉死；绑定面零变化（本批零 Go
  绑定增删，除记忆缝/boot 测试外 Go 改动集中于 boot 消费点与 wx 引擎接线）
