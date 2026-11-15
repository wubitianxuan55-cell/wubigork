# gaea-novel-retirement-20261115-eb287ecc

小说板块退役整体删除 + 文学板块接替一级导航位 功能包。

- **打包点**：`eb287ecc`（detached 干净工作树构建，主树零未提交，HEAD=打包点）
- **提交链**：`a875743a` 拆除主刀（582 文件，−108,871 行）→ `074ea154` go.mod 剪枝+wailsjs 摘除 → `2651f39a` 记档 → `eb287ecc` 代码图谱精算刷新（143 包/237,688 行源码，不变量三项绿）
- **构建**：干净树 Vite 35.07s + Wails 45.4s（bindings 未跳过）；构建后 wailsjs 内容零漂移（`git diff --quiet` 通过，warning 全为 CRLF 显示假象）；DLL 预构建件取主树 build/bin（同源同哈希）
- **冒烟**：payload 正式路径 `/api/health` 200（PID 44772）
- **产物哈希**（详见 SHA256SUMS-eb287ecc.txt）：
  - gaea.exe `c05a3a4c…fbc88490`（干净树重建哈希，与主树非 clean-tree 构建的 `41B9A1EC…` 不同属预期，代码同源）
  - onnxruntime.dll `68d0d5f5…` / sherpa-onnx-c-api.dll `5dd36dbd…` / sherpa-onnx-cxx-api.dll `4c563277…`
- **归档**：zip + tar.gz 留盘不入库（各约 29 MB）
- **功能要点**：文学板块接替小说板块一级位（menuOrder 2 / ReadOutlined / Ctrl+2 / 面包屑锚点）；绑定面 721/12→681/11；保留角色库导入链、书架 CoreB 层、prompt 引擎、三技能资产；`C:\AI\xiaoshuo` 用户数据零触碰
- **已知挂池**：角色库「加入项目」opener 休眠（书架打开链随板块退役）；存量 imagehub `source_board=novel` 记录显示保留
