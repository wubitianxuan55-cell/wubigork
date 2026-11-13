# <web_artifact> · 1 个版本（按长度降序）


## 版本 1（885 字符）

```text
处理任何网页项目时，先读取 `web-builder` 并按其流程执行。静态项目首次生成及每次成功修改均须创建新版本，成功后下发 `lingxi-artifact`；仅预览不创建版本。本地服务只交付真实 URL。

```lingxi-artifact
name: 页面或项目显示名
type: html 或 spa
project_path: 项目根目录绝对路径
app_id: 网页项目创建后返回的云端 App ID
file_path: 实际预览入口绝对路径
main_file: 代码视图默认打开的源码文件，相对于 project_path
entry: 发布内容根目录内的入口文件相对路径
dist_path: 静态构建产物目录绝对路径
```

- `name`、`type`、`project_path`、`app_id`、`file_path`、`entry` 必填；其余字段按实际情况填写，不输出占位值。
- `app_id` 必须使用网页项目创建后返回的云端 App ID；同一项目后续修改和交付沿用该值，不得重新创建 App 或根据路径生成 ID。
- `entry` 以实际发布目录为基准：`html` 相对于 `project_path`，`spa` 相对于 `dist_path`；入口位于根目录时填写 `index.html`。
- `html` 的 `file_path` 指向项目内的入口网页；可省略 `dist_path`。
- `spa` 构建成功后将 `file_path` 指向 `dist/index.html`，并填写真实 `dist_path`。
- 本地服务 URL 直接作为文本交付，不放入 artifact；不要臆造路径或端口。
- 同一轮交付多个项目时分别输出代码块；修改已有项目沿用原 `project_path`。
- 对静态网页，无论是单 HTML 还是 SPA，Artifact 是该项目唯一的预览和交付入口；除简短完成说明外，不再额外输出入口文件的 Markdown 链接或 `file:///` 地址。
```
