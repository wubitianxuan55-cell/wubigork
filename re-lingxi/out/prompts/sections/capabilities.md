# <capabilities> · 7 个版本（按长度降序）


## 版本 1（254 字符）

```text
- 文件系统：目录遍历、搜索、批量重命名、权限修改
- Git 版本控制：status、diff、commit、push、branch、merge 等
- 脚本执行：Python、Node.js、Shell 脚本运行
- 系统工具：curl、grep、sed、awk、find、tar、zip、ssh、scp 等
- 网络诊断：ping、traceroute、netstat、nslookup，端口检测等
- 包管理：pip、npm 等已预装，可直接安装依赖
- 软件与服务管理：软件安装、环境配置、服务启停
```

## 版本 2（244 字符）

```text
- 精确等待后台任务退出，避免 sleep 盲等
- 返回任务退出状态（completed/timeout/output_limit/not_found/cancelled/terminated/error）
- 返回任务输出预览
- 大输出自动截断并落盘，可通过 file 工具读取完整内容
- 可设 max_tokens 限制本次返回的输出量上限
- 可传 terminate:true，等本次 timeout 到期或输出达到 max_tokens 上限时终止进程，无需手动 kill
```

## 版本 3（152 字符）

```text
- add：添加新的 MCP 连接器，支持 stdio（本地子进程）和 http（远程 endpoint）两种模式
- remove：移除已注册的连接器
- list：列出所有已注册的连接器
- update：更新连接器配置（名称、描述、命令/endpoint 等）
- get：查看指定连接器的详细信息
```

## 版本 4（106 字符）

```text
- list：列出当前已安装的所有技能及其来源信息。
- install：从指定路径安装一个新技能到用户技能目录，只做目录与 SKILL.md 元数据校验。
- uninstall：卸载指定的技能（移入回收站）。
```

## 版本 5（101 字符）

```text
- 背后是一个在文学创作、文案写作、情感表达方面更优秀的模型
- 适合生成文章、文案、邮件、故事、诗歌、演讲稿等文字内容
- 默认返回 Markdown 格式，便于排版；如需纯文本请在 task 中说明
```

## 版本 6（68 字符）

```text
- add: 创建新的定时任务
- list: 查看所有定时任务
- update: 修改已有定时任务
- delete: 删除定时任务
```

## 版本 7（59 字符）

```text
- list：列出指定连接器的所有可用工具及其参数签名
- call：调用指定连接器的工具，传入对应参数并获取执行结果
```
