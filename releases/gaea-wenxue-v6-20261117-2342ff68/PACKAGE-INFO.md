# gaea-wenxue-v6-20261117-2342ff68

- **打包点**：`2342ff68`（main，docs 台账回写；打包时主树零未提交业务改动）
- **包面**：文学板块系统性审查十批（`ac352c45..81e43300`，共 10 功能批+1 台账）：
  1. 批1 高危修复——死锁同族 loader 无锁化/预览边界去办公根/记忆面板订阅/变更去重贯通（ac352c45）
  2. 批2 一致性——任务模板图文单源/写侧会话件围栏/书籍信息契约投影同步（4eb4beeb）
  3. 批3 卫生——死链 20 文件+死键 378 行净 −7090/mock 兑现/registry -check 门（57392dce）
  4. 批4 补防线——镜像债回种 gaea（dedup/tab 单源）/ChatTabs 测试移植/wenxue_chat 首测（68fd73fe）
  5. 批5——成本账目标签分列/novel-* 三技能播种分发链/桥面误操作根修（0e4dbf21）
  6. 批6——MCP 管理四件换锚图文引擎/技能蒸馏换锚/factbase 死链摘除（350bc382）
  7. 批7——证据链读侧换锚书工作区（Journal/Verify/Rollback）/Docx 编辑链降级只读（d6cffd7e）
  8. 批8——质检族 wenxue_quality_check 五组确定性体检（fac54447）
  9. 批9——文风族 wenxue_style_report 确定性指标+书内基线漂移差值（3500d956）
  10. 批10——预览绝对分支白名单化（读根+拾取登记）/ListDir 死防线摘除（81e43300）
- **构建**：独立干净树（detached @2342ff68，junction node_modules）；Vite 47.3s+Wails 68.8s（bindings 未跳过）；wailsjs 零漂移（git diff --quiet 过，status M 全 CRLF 假象）；DLL 取主树 build/bin（同源同哈希）
- **payload SHA256**（详见 SHA256SUMS-2342ff68.txt）：
  - gaea.exe `4E9DAAB79C89…7B6E`（干净树重建哈希≠装机位 D0A95BE7…，代码同源）
  - onnxruntime.dll `68D0D5F5…`、sherpa-onnx-c-api.dll `5DD36DBD…`、sherpa-onnx-cxx-api.dll `4C5632…`
- **冒烟**：正式路径 payload `/api/health` 200（PID 47484）
- **版本**：4.543.0（wails.json productVersion；十批未抬号——按发版约定抬号随用户拍板的发版动作走）
- zip/tar 留盘不入库（P2-3）；工作树已回收
