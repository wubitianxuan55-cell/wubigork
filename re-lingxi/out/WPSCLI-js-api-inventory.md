# wpscli JS 片段里的 WPS 对象模型调用清点

- 源: `C:\AI\wubigrok\re-lingxi\out\WPSCLI-jsapi-bodies.md`（129 条片段）
- 口径：`Application.X` / `.Method(` / `.Prop =` 三类静态计数；碎片片段会低估

## 总量

| 类型 | 片段数 | Application.* | 方法调用 | 属性赋值 |
|---|---|---|---|---|
| other | 84 | 4 | 10 | 4 |
| ppt | 19 | 4 | 8 | 0 |
| doc | 13 | 16 | 9 | 0 |
| sheet | 8 | 5 | 8 | 2 |
| image | 3 | 0 | 1 | 0 |
| pdf | 2 | 2 | 2 | 0 |

## doc

`Application.*`：`Documents`×10、`ActiveDocument`×2、`Content`×1、`BuiltInDocumentPropertiesApplication`×1、`TablesApplication`×1、`Paragraphs`×1

方法调用：`Item`、`Open`、`Add`


## sheet

`Application.*`：`Workbooks`×5

方法调用：`Range`、`Open`、`Add`、`SetSourceData`、`Item`

属性赋值：`Formula`、`Value2`


## ppt

`Application.*`：`Presentations`×4

方法调用：`Item`、`Add`、`Open`


## pdf

`Application.*`：`Documents`×2

方法调用：`Open`


## image

方法调用：`AddPicture`


## other

`Application.*`：`ActiveWorkbook`×2、`ActiveWindow`×1、`DisplayAlerts`×1

方法调用：`Item`、`SaveAs`、`AddTextbox`

属性赋值：`Text`、`DisplayAlerts`
