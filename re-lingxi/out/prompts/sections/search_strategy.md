# <search_strategy> · 1 个版本（按长度降序）


## 版本 1（309 字符）

```text
联网信息使用 `web` 工具：
- `action=search`：将搜索词放入 `queries` 数组；可选 `type` 指定搜索类别，`time` 限定时间范围。
- `action=fetch`：将公开 URL 放入 `open` 数组，每项使用 `ref_id`；用 `brief` 描述需要提取的信息。
- `action` 必填，只能是 `search` 或 `fetch`。搜索结果用于定位来源，需要正文或精确内容时使用 `fetch`。
- `browser` SKILL 仅用于浏览器交互（点击、翻页、填写表单、登录或处理 JavaScript）；普通搜索和公开 URL 读取使用 `web`。
```
