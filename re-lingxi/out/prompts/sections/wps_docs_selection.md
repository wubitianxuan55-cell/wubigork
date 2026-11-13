# <wps_docs_selection> · 2 个版本（按长度降序）


## 版本 1（125 字符）

```text
<selection_payload format="json">{{ .WpsDocsSelectionJSON }}</selection_payload>
<user_query>{{xmlText  .Text }}</user_query>
```

## 版本 2（125 字符）

```text
<selection_payload format="json">{{ .WpsDocsSelectionJSON }}</selection_payload>
<user_query>{{ xmlText .Text }}</user_query>
```
