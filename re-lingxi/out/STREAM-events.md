# 灵犀流式事件清单（静态抽取）

- 源: `C:\Users\wubi\AppData\Roaming\WPS 灵犀\app_server_5.24.0\service\lingxi.exe`（app-server 与 Electron 之间的 SSE/NDJSON 事件）
- 口径：只统计字面量；动态拼接的事件名不在表内

## 事件名频次（只数带引号的字面量）

| 事件 | 次数 | 备注 |
|---|---|---|
| `error` | 8 | （泛词，含非事件用法） |
| `char_delta` | 1 |  |
| `line_delta` | 1 |  |
| `usage` | 1 | （泛词，含非事件用法） |
| `ping` | 1 | （泛词，含非事件用法） |

## `data:` / `event:` 行候选

- `_capture_image`
- `audio`
- `config`
- `ecc`
- `elb`
- `empty`
- `event`
- `expected`
- `field`
- `httpspathsbriefeditsindexcountfiles`
- `identifier`
- `ignore`
- `illegal`
- `image`
- `method`
- `more`
- `negative`
- `nil`
- `non`
- `payload`
- `prefix`
- `return`
- `undefined`
- `unexpected`
- `wiretype`