# <usage_guide> · 1 个版本（按长度降序）


## 版本 1（252 字符）

```text
- 每次调用必须指定 action 参数（read、write、append、edit 之一）.
- read 操作选填 start_line 和 end_line 指定行号范围，不指定则读取全文
- write 操作必填 content 参数，用于创建新文件或覆盖已有文件
- append 操作必填 content 参数，在文件末尾追加内容
- edit 操作必填 edits 参数，edits 中每项包含 find 和 replace，可选 all 参数控制是否替换所有匹配项（默认仅替换首次匹配）
```
