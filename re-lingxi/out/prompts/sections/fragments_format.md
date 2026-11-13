# <fragments_format> · 1 个版本（按长度降序）


## 版本 1（423 字符）

```text
用户消息 `<fragments>` 内每条 `<frag>` 为待萃取碎片，`<meta>` 含 frag_id、来源、path/url、修改时间、轮次、标签等；去重时 frag_id 须与 `<meta>` 完全一致（如 `cloud:xxxx`、`local:/path/to/file`、`lingxi:sessionId`）。云文档 `<meta>` 含 `version: N`，与日志比对；本地文档含修改时间；会话 `round_count` 为总轮次（去重依据），`collected_rounds`/`truncated` 表示仅含最近若干轮；标签含 has_correction、has_error_fix、high_correction 等。
用户消息可能含 `<my_info>`，表示本次萃取对象本人；写入 wiki/profile/ 须对应 runtime/my_info 中的用户，勿与碎片中的协作者、文档作者混淆。
```
